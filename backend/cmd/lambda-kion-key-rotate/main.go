// Command lambda-kion-key-rotate runs one Kion API key rotation cycle per
// invocation. It is triggered on a daily EventBridge schedule and is safe to
// run as often as the scheduler fires because the orchestrator enforces an
// idempotency window (ROTATE_AFTER_DAYS) around the stored rotated_at
// timestamp.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	awscfg "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"

	"github.com/CMS-Enterprise/ztmf/backend/cmd/lambda-kion-key-rotate/internal/kion"
	"github.com/CMS-Enterprise/ztmf/backend/cmd/lambda-kion-key-rotate/internal/rotate"
	"github.com/CMS-Enterprise/ztmf/backend/internal/config"
	"github.com/CMS-Enterprise/ztmf/backend/internal/notifications"
	"github.com/CMS-Enterprise/ztmf/backend/internal/secrets"
)

const (
	metricNamespace  = "ZTMF/Kion"
	metricName       = "DaysSinceRotation"
	defaultRotateDays = 4
)

// Event is the JSON input delivered by EventBridge or a manual invocation.
type Event struct {
	TriggerType string `json:"trigger_type"`
	DryRun      bool   `json:"dry_run"`
	Force       bool   `json:"force"`
}

func handler(ctx context.Context, raw json.RawMessage) error {
	cfg := config.GetInstance()

	log.Printf("Kion key rotation Lambda start: environment=%s", cfg.Env)

	evt := parseEvent(raw, cfg.Env)
	log.Printf("event: trigger_type=%s dry_run=%t force=%t", evt.TriggerType, evt.DryRun, evt.Force)

	secretID := os.Getenv("KION_SECRET_ID")
	if secretID == "" {
		return fmt.Errorf("KION_SECRET_ID env var is required")
	}

	rotateAfterDays := defaultRotateDays
	if raw := os.Getenv("ROTATE_AFTER_DAYS"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			rotateAfterDays = parsed
		}
	}

	secret, err := secrets.NewSecret(secretID)
	if err != nil {
		return fmt.Errorf("load secret %q: %w", secretID, err)
	}

	notifier, err := notifications.NewSlackNotifier(ctx)
	if err != nil {
		// Slack failure must not block rotation; log and proceed with a no-op notifier.
		log.Printf("Slack notifier unavailable, continuing without notifications: %v", err)
		notifier = nil
	}

	metrics, err := newCloudWatchMetrics(ctx, cfg.Env)
	if err != nil {
		log.Printf("CloudWatch metrics unavailable, continuing without metrics: %v", err)
		metrics = nil
	}

	kionFactory := func(baseURL string) rotate.KionClient {
		return kion.NewClient(baseURL)
	}

	in := rotate.Input{
		Environment:     cfg.Env,
		SecretName:      secretID,
		RotateAfterDays: rotateAfterDays,
		DryRun:          evt.DryRun,
		Force:           evt.Force,
	}

	var rotNotifier rotate.Notifier
	if notifier != nil {
		rotNotifier = notifier
	}
	var rotMetrics rotate.MetricPublisher
	if metrics != nil {
		rotMetrics = metrics
	}

	orch := rotate.New(in, secret, kionFactory, rotNotifier, rotMetrics, nil)
	res, err := orch.Run(ctx)
	if err != nil {
		log.Printf("rotation failed: %v", err)
		return err
	}
	log.Printf("rotation result: rotated=%t skipped=%t duration=%s", res.Rotated, res.Skipped, res.Duration)
	return nil
}

func parseEvent(raw json.RawMessage, env string) Event {
	var evt Event
	if err := json.Unmarshal(raw, &evt); err == nil && evt.TriggerType != "" {
		return evt
	}

	// Fall back to CloudWatch scheduled-event shape.
	var cw events.CloudWatchEvent
	if err := json.Unmarshal(raw, &cw); err == nil && cw.Source != "" {
		return Event{
			TriggerType: "scheduled",
			DryRun:      env != "prod",
		}
	}

	// Empty or manual-test event.
	return Event{
		TriggerType: "manual",
		DryRun:      env != "prod",
	}
}

// cloudWatchMetrics publishes the DaysSinceRotation gauge.
type cloudWatchMetrics struct {
	client      *cloudwatch.Client
	environment string
}

func newCloudWatchMetrics(ctx context.Context, env string) (*cloudWatchMetrics, error) {
	sdkCfg, err := awscfg.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, err
	}
	return &cloudWatchMetrics{
		client:      cloudwatch.NewFromConfig(sdkCfg),
		environment: env,
	}, nil
}

func (m *cloudWatchMetrics) PublishDaysSinceRotation(ctx context.Context, days float64) error {
	_, err := m.client.PutMetricData(ctx, &cloudwatch.PutMetricDataInput{
		Namespace: aws.String(metricNamespace),
		MetricData: []types.MetricDatum{
			{
				MetricName: aws.String(metricName),
				Value:      aws.Float64(days),
				Unit:       types.StandardUnitCount,
				Dimensions: []types.Dimension{
					{Name: aws.String("Environment"), Value: aws.String(m.environment)},
				},
			},
		},
	})
	return err
}

func main() {
	cfg := config.GetInstance()
	log.Printf("Starting ZTMF Kion key rotation Lambda - environment=%s", cfg.Env)
	lambda.Start(handler)
}
