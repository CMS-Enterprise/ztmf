package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"

	"github.com/CMS-Enterprise/ztmf/backend/cmd/lambda-cfacts-s3/internal/sync"
	"github.com/CMS-Enterprise/ztmf/backend/internal/config"
	"github.com/CMS-Enterprise/ztmf/backend/internal/notifications"
)

// CfactsS3Event represents a manual invocation event for CFACTS S3 CSV sync.
type CfactsS3Event struct {
	TriggerType string `json:"trigger_type"` // "s3" | "manual"
	Bucket      string `json:"bucket"`
	Key         string `json:"key"`
	DryRun      bool   `json:"dry_run"`
}

func handler(ctx context.Context, event json.RawMessage) error {
	cfg := config.GetInstance()

	log.Printf("ZTMF CFACTS S3 CSV Sync Lambda started - Environment: %s", cfg.Env)

	bucket, key, dryRun, triggerType, err := parseEvent(event)
	if err != nil {
		log.Printf("Failed to parse event: %v", err)
		return err
	}

	// Validate the S3 key
	if !strings.HasPrefix(key, "incoming/") || !strings.HasSuffix(key, ".csv") {
		return fmt.Errorf("invalid S3 key %q: must start with 'incoming/' and end with '.csv'", key)
	}

	log.Printf("CFACTS S3 sync: Bucket=%s, Key=%s, DryRun=%t, TriggerType=%s", bucket, key, dryRun, triggerType)

	// Initialize synchronizer
	synchronizer, err := sync.NewSynchronizer(ctx, dryRun)
	if err != nil {
		log.Printf("Failed to initialize synchronizer: %v", err)
		return err
	}
	defer synchronizer.Close()

	// Execute sync
	result, err := synchronizer.ExecuteSync(ctx, bucket, key)
	if err != nil {
		log.Printf("CFACTS S3 sync failed: %v", err)
		sendSlackNotification(ctx, triggerType, 0, err)
		return err
	}

	log.Printf("CFACTS S3 sync completed: %d rows synced in %v", result.RowsInserted, result.Duration)

	// Send success notification
	if notifyErr := sendSlackNotification(ctx, triggerType, result.RowsInserted, nil); notifyErr != nil {
		log.Printf("Failed to send Slack notification: %v", notifyErr)
	}

	return nil
}

// parseEvent extracts bucket, key, dryRun from either an S3 event or manual event.
func parseEvent(event json.RawMessage) (bucket, key string, dryRun bool, triggerType string, err error) {
	// Try S3 event first
	var s3Event events.S3Event
	if jsonErr := json.Unmarshal(event, &s3Event); jsonErr == nil && len(s3Event.Records) > 0 {
		record := s3Event.Records[0]
		return record.S3.Bucket.Name, record.S3.Object.Key, false, "s3", nil
	}

	// Try manual event
	var manual CfactsS3Event
	if jsonErr := json.Unmarshal(event, &manual); jsonErr == nil && manual.Bucket != "" && manual.Key != "" {
		if manual.TriggerType == "" {
			manual.TriggerType = "manual"
		}
		return manual.Bucket, manual.Key, manual.DryRun, manual.TriggerType, nil
	}

	return "", "", false, "", fmt.Errorf("could not parse event as S3 or manual event")
}

func sendSlackNotification(ctx context.Context, triggerType string, rowCount int64, syncErr error) error {
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
		TriggerType:   fmt.Sprintf("cfacts-s3/%s", triggerType),
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
	log.Printf("Starting ZTMF CFACTS S3 CSV Sync Lambda - Environment: %s", cfg.Env)
	lambda.Start(handler)
}
