package notifications

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/CMS-Enterprise/ztmf/backend/internal/config"
	"github.com/CMS-Enterprise/ztmf/backend/internal/secrets"
)

// slackHTTPClient is reused across SendSlack calls. CheckRedirect returns
// http.ErrUseLastResponse so the client does not follow 3xx: the Slack webhook
// should never redirect, and following a redirect would replay the POST body
// (which can include credential material on the RecoveryKey path) to an
// unintended destination.
var slackHTTPClient = &http.Client{
	Timeout: 10 * time.Second,
	CheckRedirect: func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	},
}

// SlackNotifier handles sending notifications to Slack
type SlackNotifier struct {
	webhookURL  string
	environment string
}

// SyncResult represents the results of a data sync operation for notifications
type SyncResult struct {
	Environment   string
	TriggerType   string
	DryRun        bool
	SuccessCount  int
	FailureCount  int
	TotalRows     int64
	Duration      time.Duration
	FailedTables  []string
	ErrorMessages []string
}

// NewSlackNotifier creates a new Slack notifier
func NewSlackNotifier(ctx context.Context) (*SlackNotifier, error) {
	cfg := config.GetInstance()

	// Load Slack webhook URLs from secrets
	webhookSecret, err := secrets.NewSecret("ztmf_slack_webhook")
	if err != nil {
		return nil, fmt.Errorf("failed to load Slack webhook secret: %w", err)
	}

	type webhookConfig struct {
		Primary   string `json:"primary"`             // Main alerts channel
		Secondary string `json:"secondary,omitempty"` // Optional secondary channel
		Critical  string `json:"critical,omitempty"`  // Optional critical alerts channel
	}

	var webhook webhookConfig
	if err := webhookSecret.Unmarshal(&webhook); err != nil {
		return nil, fmt.Errorf("failed to unmarshal Slack webhook config: %w", err)
	}

	// Use primary webhook URL (can be extended for multiple channels later)
	return &SlackNotifier{
		webhookURL:  webhook.Primary,
		environment: cfg.Env,
	}, nil
}

// SendSyncNotification sends a notification about sync results
func (s *SlackNotifier) SendSyncNotification(ctx context.Context, result SyncResult) error {
	message := s.buildSyncMessage(result)

	payload := map[string]interface{}{
		"text": message,
	}

	return s.sendToSlack(ctx, payload)
}

// RotationResult represents the outcome of a credential rotation job for notifications.
// RecoveryKey must only be populated in the narrow failure window where the upstream
// rotation succeeded but persisting the new value to AWS Secrets Manager failed; it is
// emitted so an operator can paste the key into Secrets Manager manually. All other
// paths must leave RecoveryKey empty so no key material is ever written to Slack.
type RotationResult struct {
	Environment      string
	Service          string
	SecretName       string
	Success          bool
	DryRun           bool
	Skipped          bool
	DaysSinceRotated int
	Duration         time.Duration
	ErrorMessage     string
	RecoveryKey      string
}

// SendRotationNotification posts a credential rotation result to Slack. The recovery
// path is flagged as critical and is the only code path that serializes key material.
func (s *SlackNotifier) SendRotationNotification(ctx context.Context, r RotationResult) error {
	message := s.buildRotationMessage(r)

	payload := map[string]interface{}{
		"text": message,
	}

	return s.sendToSlack(ctx, payload)
}

// buildRotationMessage formats a human-readable Slack message for a rotation outcome.
func (s *SlackNotifier) buildRotationMessage(r RotationResult) string {
	envUpper := strings.ToUpper(r.Environment)
	dur := formatDuration(r.Duration)

	if r.RecoveryKey != "" {
		// Critical: the upstream provider rotated but the secret write failed.
		// The operator must paste this value into the secret manually.
		return fmt.Sprintf(`🚨 %s ROTATION CRITICAL (%s)
❌ Secret: %s
❌ Upstream rotation succeeded, but persisting to Secrets Manager failed after retries.
🔧 Paste this value into AWSCURRENT for %s immediately:
`+"```%s```"+`
Error: %s
⏱️ Duration: %s`,
			r.Service, envUpper, r.SecretName, r.SecretName, r.RecoveryKey, r.ErrorMessage, dur)
	}

	if !r.Success {
		errorSummary := r.ErrorMessage
		if len(errorSummary) > 200 {
			errorSummary = errorSummary[:200] + "..."
		}
		return fmt.Sprintf(`🚨 %s ROTATION FAILURE (%s)
❌ Secret: %s
🔧 Action Required: %s
⏱️ Duration: %s`,
			r.Service, envUpper, r.SecretName, errorSummary, dur)
	}

	if r.Skipped {
		return fmt.Sprintf(`⏭️ %s ROTATION SKIPPED (%s)
ℹ️ Secret: %s (last rotated %d day(s) ago, under threshold)`,
			r.Service, envUpper, r.SecretName, r.DaysSinceRotated)
	}

	if r.DryRun {
		return fmt.Sprintf(`✅ %s ROTATION DRY RUN (%s)
🧪 Secret: %s (no changes written; upstream rotation and secret put were skipped)
⏱️ Duration: %s`,
			r.Service, envUpper, r.SecretName, dur)
	}

	return fmt.Sprintf(`✅ %s ROTATION SUCCESS (%s)
🔐 Secret: %s (AWSCURRENT updated, previous moved to AWSPREVIOUS)
⏱️ Duration: %s`,
		r.Service, envUpper, r.SecretName, dur)
}

// CertRotationResult describes the outcome of one cert-rotation Lambda invocation
// for Slack notification. Success/DryRun/ValidationFailed are mutually exclusive
// with Success: a Success run was a real (or dry-run) successful rotation, a
// ValidationFailed run is an operator-correctable input problem (bad PEM, wrong
// domain, expired cert), and any other non-success is treated as an infra
// failure (ACM import, Secrets Manager put, S3 archive, etc.).
type CertRotationResult struct {
	Environment       string
	Domain            string
	Success           bool
	DryRun            bool
	ValidationFailed  bool
	NotAfter          time.Time
	DaysRemaining     int
	IntermediateCount int
	AcmCertificateArn string
	ActionRequired    string
	ErrorMessage      string
	S3Location        string
}

// SendCertRotationNotification posts a cert-rotation result to Slack using the
// shared notifier plumbing (webhook secret lookup, redirect-safe HTTP client).
func (s *SlackNotifier) SendCertRotationNotification(ctx context.Context, r CertRotationResult) error {
	message := s.buildCertRotationMessage(r)

	payload := map[string]interface{}{
		"text": message,
	}

	return s.sendToSlack(ctx, payload)
}

// buildCertRotationMessage formats a Slack message for a cert-rotation outcome.
func (s *SlackNotifier) buildCertRotationMessage(r CertRotationResult) string {
	envUpper := strings.ToUpper(r.Environment)

	if r.Success {
		if r.DryRun {
			return fmt.Sprintf(`✅ TLS CERT ROTATION SUCCESS (%s) [DRY RUN]
🔐 Domain: %s
📅 Expires: %s (%d days remaining)
🔗 Chain: Server cert + %d intermediate CA`,
				envUpper,
				r.Domain,
				r.NotAfter.UTC().Format("2006-01-02"),
				r.DaysRemaining,
				r.IntermediateCount)
		}
		return fmt.Sprintf(`✅ TLS CERT ROTATION SUCCESS (%s)
🔐 Domain: %s
📅 Expires: %s (%d days remaining)
🔗 Chain: Server cert + %d intermediate CA
🪪 ACM ARN: %s`,
			envUpper,
			r.Domain,
			r.NotAfter.UTC().Format("2006-01-02"),
			r.DaysRemaining,
			r.IntermediateCount,
			r.AcmCertificateArn)
	}

	errorSummary := r.ErrorMessage
	if len(errorSummary) > 300 {
		errorSummary = errorSummary[:300] + "..."
	}

	if r.ValidationFailed {
		action := strings.TrimSpace(r.ActionRequired)
		if action == "" {
			action = "Upload valid certificate files and retry."
		}
		return fmt.Sprintf(`🚨 TLS CERT ROTATION FAILED (%s)
🔐 Domain: %s
❌ Error: %s
🔧 Action Required: %s
📍 Location: %s`,
			envUpper,
			r.Domain,
			errorSummary,
			action,
			r.S3Location)
	}

	return fmt.Sprintf(`🚨 TLS CERT ROTATION FAILED (%s)
🔐 Domain: %s
❌ Error: %s
🔧 Action Required: Investigate Lambda logs and AWS resource permissions, then retry rotation.
📍 Location: %s`,
		envUpper,
		r.Domain,
		errorSummary,
		r.S3Location)
}

// getSyncLabel returns a human-readable label for the sync direction
func getSyncLabel(triggerType string) string {
	if strings.HasPrefix(triggerType, "cfacts-snowflake") {
		return "CFACTS Import (SDL → ZTMF)"
	}
	return "Data Export (ZTMF → SDL)"
}

// buildSyncMessage creates formatted Slack message based on sync results
func (s *SlackNotifier) buildSyncMessage(result SyncResult) string {
	quarter := getCurrentQuarter()
	scheduleType := getScheduleType(result.Environment)
	envUpper := strings.ToUpper(result.Environment)
	syncLabel := getSyncLabel(result.TriggerType)

	if result.FailureCount == 0 {
		var dataMessage string
		if result.DryRun {
			dataMessage = fmt.Sprintf("🧪 %s dry-run validation completed successfully", quarter)
		} else if result.Environment == "prod" {
			dataMessage = fmt.Sprintf("📅 %s data now available", quarter)
		} else {
			dataMessage = fmt.Sprintf("📅 %s data synced to %s", quarter, strings.ToLower(result.Environment))
		}

		return fmt.Sprintf(`✅ %s SUCCESS (%s - %s)
📊 %d tables synced: %s rows
⏱️ Duration: %s
%s`,
			syncLabel,
			envUpper,
			scheduleType,
			result.SuccessCount,
			formatNumber(result.TotalRows),
			formatDuration(result.Duration),
			dataMessage)
	} else {
		failedTablesStr := strings.Join(result.FailedTables, ", ")
		errorSummary := ""
		if len(result.ErrorMessages) > 0 {
			errorSummary = result.ErrorMessages[0]
			if len(errorSummary) > 100 {
				errorSummary = errorSummary[:100] + "..."
			}
		}

		return fmt.Sprintf(`🚨 %s FAILURE (%s - %s)
❌ %d table(s) failed: %s
✅ %d tables successful: %s rows
🔧 Action Required: %s`,
			syncLabel,
			envUpper,
			scheduleType,
			result.FailureCount,
			failedTablesStr,
			result.SuccessCount,
			formatNumber(result.TotalRows),
			errorSummary)
	}
}

// sendToSlack sends payload to Slack webhook
func (s *SlackNotifier) sendToSlack(ctx context.Context, payload map[string]interface{}) error {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal Slack payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", s.webhookURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create Slack request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := slackHTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send Slack notification: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("slack webhook returned status %d", resp.StatusCode)
	}

	return nil
}

// getCurrentQuarter returns the current quarter (e.g., "Q3 2025")
func getCurrentQuarter() string {
	now := time.Now()
	year := now.Year()
	month := int(now.Month())

	var quarter string
	switch {
	case month >= 1 && month <= 3:
		quarter = "Q1"
	case month >= 4 && month <= 6:
		quarter = "Q2"
	case month >= 7 && month <= 9:
		quarter = "Q3"
	case month >= 10 && month <= 12:
		quarter = "Q4"
	}

	return fmt.Sprintf("%s %d", quarter, year)
}

// getScheduleType returns schedule description based on environment
func getScheduleType(env string) string {
	if env == "prod" {
		return "Quarterly"
	}
	return "Weekly"
}

// formatNumber formats large numbers with commas
func formatNumber(n int64) string {
	str := fmt.Sprintf("%d", n)
	if len(str) <= 3 {
		return str
	}

	var result []byte
	for i, char := range str {
		if i > 0 && (len(str)-i)%3 == 0 {
			result = append(result, ',')
		}
		result = append(result, byte(char))
	}

	return string(result)
}

// formatDuration formats duration in human-readable format
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	} else if d < time.Hour {
		return fmt.Sprintf("%.1fm", d.Minutes())
	}
	return fmt.Sprintf("%.1fh", d.Hours())
}
