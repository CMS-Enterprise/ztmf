
package main

import (
	"context"
	"encoding/json"
	"log"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"

	"github.com/CMS-Enterprise/ztmf/backend/internal/config"
	"github.com/CMS-Enterprise/ztmf/backend/cmd/lambda/internal/sync"
)

// SyncEvent represents the event payload for triggering data sync
type SyncEvent struct {
	TriggerType string   `json:"trigger_type"` // "scheduled" | "manual"
	Tables      []string `json:"tables"`       // Optional: specific tables to sync
	FullRefresh bool     `json:"full_refresh"` // Force truncate/reload
	DryRun      bool     `json:"dry_run"`      // Validation only
}

func handler(ctx context.Context, event json.RawMessage) error {
	cfg := config.GetInstance()
	
	log.Printf("ZTMF Data Sync Lambda started - Environment: %s", cfg.Env)
	
	// Parse the incoming event
	var syncEvent SyncEvent
	
	// Try to parse as SyncEvent first, fall back to CloudWatch event
	if err := json.Unmarshal(event, &syncEvent); err != nil {
		// If direct parsing fails, try CloudWatch scheduled event
		var cwEvent events.CloudWatchEvent
		if cwErr := json.Unmarshal(event, &cwEvent); cwErr != nil {
			log.Printf("Failed to parse event as SyncEvent or CloudWatch event: %v", err)
			return err
		}
		
		// For scheduled events, use default configuration
		syncEvent = SyncEvent{
			TriggerType: "scheduled",
			FullRefresh: false,
			DryRun:      cfg.Env != "prod", // Dry run in non-prod
		}
		
		log.Printf("Received CloudWatch scheduled event: %s", cwEvent.Source)
	}
	
	log.Printf("Sync configuration: TriggerType=%s, FullRefresh=%t, DryRun=%t, Tables=%v", 
		syncEvent.TriggerType, syncEvent.FullRefresh, syncEvent.DryRun, syncEvent.Tables)
	
	// Initialize the synchronizer
	synchronizer, err := sync.NewSynchronizer(ctx, syncEvent.DryRun)
	if err != nil {
		log.Printf("Failed to initialize synchronizer: %v", err)
		return err
	}
	defer synchronizer.Close()
	
	// Execute the sync
	result, err := synchronizer.ExecuteSync(ctx, sync.SyncOptions{
		Tables:      syncEvent.Tables,
		FullRefresh: syncEvent.FullRefresh,
	})
	
	if err != nil {
		log.Printf("Sync failed: %v", err)
		return err
	}
	
	log.Printf("Sync completed successfully: %s", result.Summary())
	return nil
}

func main() {
	// Initialize configuration
	cfg := config.GetInstance()
	log.Printf("Starting ZTMF Data Sync Lambda - Environment: %s", cfg.Env)
	
	// Start the Lambda runtime
	lambda.Start(handler)
}