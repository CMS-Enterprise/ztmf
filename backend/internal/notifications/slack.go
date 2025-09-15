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

// SlackNotifier handles sending notifications to Slack
type SlackNotifier struct {
	webhookURL string
	environment string
}

// SyncResult represents the results of a data sync operation for notifications
type SyncResult struct {
	Environment    string
	TriggerType    string
	SuccessCount   int
	FailureCount   int
	TotalRows      int64
	Duration       time.Duration
	FailedTables   []string
	ErrorMessages  []string
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
		Primary   string `json:"primary"`   // Main alerts channel
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

// buildSyncMessage creates formatted Slack message based on sync results
func (s *SlackNotifier) buildSyncMessage(result SyncResult) string {
	quarter := getCurrentQuarter()
	scheduleType := getScheduleType(result.Environment)
	envUpper := strings.ToUpper(result.Environment)
	
	if result.FailureCount == 0 {
		// Success message
		return fmt.Sprintf(`✅ ZTMF Data Sync SUCCESS (%s - %s)
📊 %d tables synced: %s rows
⏱️ Duration: %s
📅 %s data now available in Snowflake`,
			envUpper,
			scheduleType,
			result.SuccessCount,
			formatNumber(result.TotalRows),
			formatDuration(result.Duration),
			quarter)
	} else {
		// Failure message
		failedTablesStr := strings.Join(result.FailedTables, ", ")
		errorSummary := ""
		if len(result.ErrorMessages) > 0 {
			// Get first error for summary
			errorSummary = result.ErrorMessages[0]
			if len(errorSummary) > 100 {
				errorSummary = errorSummary[:100] + "..."
			}
		}
		
		return fmt.Sprintf(`🚨 ZTMF Data Sync FAILURE (%s - %s)
❌ %d table(s) failed: %s
✅ %d tables successful: %s rows
🔧 Action Required: %s`,
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
	
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
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