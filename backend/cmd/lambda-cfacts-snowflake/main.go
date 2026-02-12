package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"

	"github.com/CMS-Enterprise/ztmf/backend/cmd/lambda-cfacts-snowflake/internal/sync"
	"github.com/CMS-Enterprise/ztmf/backend/internal/config"
	"github.com/CMS-Enterprise/ztmf/backend/internal/notifications"
)

// CfactsSyncEvent represents the event payload for triggering CFACTS Snowflake sync.
type CfactsSyncEvent struct {
	TriggerType string `json:"trigger_type"` // "scheduled" | "manual"
	DryRun      bool   `json:"dry_run"`
}

func handler(ctx context.Context, event json.RawMessage) error {
	cfg := config.GetInstance()

	log.Printf("ZTMF CFACTS Snowflake Sync Lambda started - Environment: %s", cfg.Env)

	// Parse the incoming event
	var syncEvent CfactsSyncEvent

	if err := json.Unmarshal(event, &syncEvent); err != nil {
		// Fall back to CloudWatch scheduled event
		var cwEvent events.CloudWatchEvent
		if cwErr := json.Unmarshal(event, &cwEvent); cwErr != nil {
			log.Printf("Failed to parse event: %v", err)
			return err
		}

		syncEvent = CfactsSyncEvent{
			TriggerType: "scheduled",
			DryRun:      cfg.Env != "prod",
		}

		log.Printf("Received CloudWatch scheduled event: %s", cwEvent.Source)
	}

	// Apply defaults for empty manual test events
	if syncEvent.TriggerType == "" {
		syncEvent.TriggerType = "manual"
		if cfg.Env != "prod" {
			syncEvent.DryRun = true
			log.Printf("Defaulting to dry-run mode for manual test in %s environment", cfg.Env)
		}
	}

	log.Printf("CFACTS Snowflake sync: TriggerType=%s, DryRun=%t", syncEvent.TriggerType, syncEvent.DryRun)

	// Initialize synchronizer
	synchronizer, err := sync.NewSynchronizer(ctx, syncEvent.DryRun)
	if err != nil {
		log.Printf("Failed to initialize synchronizer: %v", err)
		return err
	}
	defer synchronizer.Close()

	// Execute sync
	result, err := synchronizer.ExecuteSync(ctx)
	if err != nil {
		log.Printf("CFACTS Snowflake sync failed: %v", err)
		// Still try to send failure notification
		sendSlackNotification(ctx, syncEvent, 0, err)
		return err
	}

	log.Printf("CFACTS Snowflake sync completed: %d rows synced in %v", result.RowsInserted, result.Duration)

	// Send success notification
	if notifyErr := sendSlackNotification(ctx, syncEvent, result.RowsInserted, nil); notifyErr != nil {
		log.Printf("Failed to send Slack notification: %v", notifyErr)
	}

	return nil
}

func sendSlackNotification(ctx context.Context, event CfactsSyncEvent, rowCount int64, syncErr error) error {
	notifier, err := notifications.NewSlackNotifier(ctx)
	if err != nil {
		return fmt.Errorf("failed to initialize Slack notifier: %w", err)
	}

	failureCount := 0
	var failedTables []string
	var errorMessages []string

	if syncErr != nil {
		failureCount = 1
		failedTables = []string{"cfacts_systems"}
		errorMessages = []string{syncErr.Error()}
	}

	result := notifications.SyncResult{
		Environment:   config.GetInstance().Env,
		TriggerType:   fmt.Sprintf("cfacts-snowflake/%s", event.TriggerType),
		SuccessCount:  1 - failureCount,
		FailureCount:  failureCount,
		TotalRows:     rowCount,
		FailedTables:  failedTables,
		ErrorMessages: errorMessages,
	}

	return notifier.SendSyncNotification(ctx, result)
}

func main() {
	cfg := config.GetInstance()
	log.Printf("Starting ZTMF CFACTS Snowflake Sync Lambda - Environment: %s", cfg.Env)
	lambda.Start(handler)
}
