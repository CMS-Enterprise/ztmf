// Package rotate orchestrates one Kion API key rotation cycle.
//
// Flow:
//  1. Load the secret for this environment and unmarshal api_key/base_url/rotated_at.
//  2. If the stored rotated_at is newer than ROTATE_AFTER_DAYS and force is not set,
//     short-circuit as Skipped and return. This keeps a daily schedule safe to run
//     without burning a fresh Kion key each day.
//  3. Call Kion to exchange the current key for a new one.
//  4. Persist the new key back to the same secret. AWS Secrets Manager rotates the
//     AWSCURRENT/AWSPREVIOUS version stages automatically.
//  5. Emit a CloudWatch custom metric and a Slack message.
//
// The dangerous window is between steps 3 and 4: if Kion accepts the rotation but
// the subsequent secret write fails, the old key is dead and the new one exists
// only in memory. The orchestrator retries the write and, if all retries fail,
// emits a critical Slack alert that carries the new key so an operator can paste
// it into Secrets Manager by hand. That is the only code path that ever serializes
// key material outside of AWS.
package rotate

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math"
	"time"

	"github.com/CMS-Enterprise/ztmf/backend/internal/notifications"
)

const (
	serviceLabel     = "Kion API Key"
	secretWriteTries = 5
)

// backoffBase is the base unit for exponential backoff on secret-write retries.
// Exposed as a package-level variable so tests can shrink it to keep the suite
// fast and deterministic without changing the production behavior.
var backoffBase = 1 * time.Second

// recoveryNotifyTimeout is how long we give the Slack webhook in the recovery
// branch after upstream rotation succeeded but persisting the new value failed.
// The parent Lambda context may be nearly exhausted at that point, so the
// recovery path uses a fresh detached context to maximize the chance the
// operator receives the alert carrying the recovery key.
const recoveryNotifyTimeout = 15 * time.Second

// KionClient rotates the Kion App API key at a given base URL.
type KionClient interface {
	Rotate(ctx context.Context, currentKey string) (string, error)
}

// KionClientFactory constructs a KionClient for a given base URL. The base URL
// lives inside the secret payload, so the client must be built after the secret
// is loaded.
type KionClientFactory func(baseURL string) KionClient

// SecretStore reads and writes the rotation secret. Both *secrets.Secret methods
// match this shape exactly.
type SecretStore interface {
	Unmarshal(v any) error
	Put(ctx context.Context, v any) error
}

// Notifier posts a rotation outcome to Slack.
type Notifier interface {
	SendRotationNotification(ctx context.Context, r notifications.RotationResult) error
}

// MetricPublisher emits the DaysSinceRotation gauge metric. Implementations
// should not block rotation on metric failures.
type MetricPublisher interface {
	PublishDaysSinceRotation(ctx context.Context, days float64) error
}

// Secret is the JSON shape stored in AWS Secrets Manager. Both api_key and
// base_url are required on read; rotated_at is nil on first bootstrap and
// populated after the first successful rotation. RotatedAt is a pointer
// because stdlib encoding/json does not apply omitempty to time.Time structs,
// and we want the absence of a previous rotation to be unambiguous on the
// wire.
type Secret struct {
	APIKey    string     `json:"api_key"`
	BaseURL   string     `json:"base_url"`
	RotatedAt *time.Time `json:"rotated_at,omitempty"`
}

// Input controls one invocation.
type Input struct {
	Environment     string
	SecretName      string
	RotateAfterDays int
	DryRun          bool
	Force           bool
}

// Result captures what happened in a single run.
type Result struct {
	Rotated          bool
	Skipped          bool
	DaysSinceRotated int
	Duration         time.Duration
}

// Orchestrator wires a single rotation cycle. Construct one per invocation.
type Orchestrator struct {
	in           Input
	secret       SecretStore
	kionFactory  KionClientFactory
	notifier     Notifier
	metrics      MetricPublisher
	now          func() time.Time
}

// New builds an Orchestrator. Now may be nil, in which case time.Now is used.
func New(in Input, secret SecretStore, kionFactory KionClientFactory, notifier Notifier, metrics MetricPublisher, now func() time.Time) *Orchestrator {
	if now == nil {
		now = time.Now
	}
	return &Orchestrator{
		in:          in,
		secret:      secret,
		kionFactory: kionFactory,
		notifier:    notifier,
		metrics:     metrics,
		now:         now,
	}
}

// Run executes one rotation cycle. Returns error if the rotation itself failed;
// Slack and metric failures are logged but do not propagate as errors.
func (o *Orchestrator) Run(ctx context.Context) (*Result, error) {
	start := o.now()

	var current Secret
	if err := o.secret.Unmarshal(&current); err != nil {
		return nil, fmt.Errorf("load secret %q: %w", o.in.SecretName, err)
	}
	if current.APIKey == "" {
		return nil, fmt.Errorf("secret %q: api_key is empty, seed it before running", o.in.SecretName)
	}
	if current.BaseURL == "" {
		return nil, fmt.Errorf("secret %q: base_url is empty, seed it before running", o.in.SecretName)
	}

	daysSince := daysSinceRotation(start, current.RotatedAt)

	if !o.in.Force && current.RotatedAt != nil && !current.RotatedAt.IsZero() && daysSince < o.in.RotateAfterDays {
		log.Printf("rotation skipped: %d day(s) since last rotation, threshold is %d", daysSince, o.in.RotateAfterDays)
		o.publishMetric(ctx, daysSince)
		o.notify(ctx, notifications.RotationResult{
			Environment:      o.in.Environment,
			Service:          serviceLabel,
			SecretName:       o.in.SecretName,
			Success:          true,
			Skipped:          true,
			DaysSinceRotated: daysSince,
			Duration:         o.now().Sub(start),
		})
		return &Result{Skipped: true, DaysSinceRotated: daysSince, Duration: o.now().Sub(start)}, nil
	}

	if o.in.DryRun {
		log.Printf("rotation dry-run: would rotate (days since last = %d)", daysSince)
		o.publishMetric(ctx, daysSince)
		o.notify(ctx, notifications.RotationResult{
			Environment:      o.in.Environment,
			Service:          serviceLabel,
			SecretName:       o.in.SecretName,
			Success:          true,
			DryRun:           true,
			DaysSinceRotated: daysSince,
			Duration:         o.now().Sub(start),
		})
		return &Result{Duration: o.now().Sub(start)}, nil
	}

	log.Printf("rotating: %d day(s) since last rotation", daysSince)
	kionClient := o.kionFactory(current.BaseURL)
	newKey, err := kionClient.Rotate(ctx, current.APIKey)
	if err != nil {
		log.Printf("Kion rotation failed: %v", err)
		o.notify(ctx, notifications.RotationResult{
			Environment:      o.in.Environment,
			Service:          serviceLabel,
			SecretName:       o.in.SecretName,
			Success:          false,
			DaysSinceRotated: daysSince,
			Duration:         o.now().Sub(start),
			ErrorMessage:     err.Error(),
		})
		return nil, err
	}

	now := o.now().UTC()
	updated := Secret{
		APIKey:    newKey,
		BaseURL:   current.BaseURL,
		RotatedAt: &now,
	}

	if writeErr := o.putWithRetries(ctx, updated); writeErr != nil {
		// Kion rotated successfully but we could not persist. The old key is
		// dead. Emit the recovery alert so an operator can paste the new key
		// manually. Use a detached context so the Slack POST is not starved
		// by a nearly-exhausted Lambda deadline: this is the one notification
		// we must not lose.
		log.Printf("secret persist failed after Kion rotation: %v", writeErr)
		notifyCtx, cancel := context.WithTimeout(context.Background(), recoveryNotifyTimeout)
		defer cancel()
		o.notify(notifyCtx, notifications.RotationResult{
			Environment:      o.in.Environment,
			Service:          serviceLabel,
			SecretName:       o.in.SecretName,
			Success:          false,
			DaysSinceRotated: 0,
			Duration:         o.now().Sub(start),
			ErrorMessage:     writeErr.Error(),
			RecoveryKey:      newKey,
		})
		return nil, fmt.Errorf("persist new key to secret: %w", writeErr)
	}

	log.Print("rotation complete")
	o.publishMetric(ctx, 0)
	o.notify(ctx, notifications.RotationResult{
		Environment:      o.in.Environment,
		Service:          serviceLabel,
		SecretName:       o.in.SecretName,
		Success:          true,
		DaysSinceRotated: 0,
		Duration:         o.now().Sub(start),
	})
	return &Result{Rotated: true, Duration: o.now().Sub(start)}, nil
}

// putWithRetries retries PutSecretValue up to secretWriteTries times with
// exponential backoff (1s, 2s, 4s, 8s, 16s). Only called in the narrow failure
// window where Kion has already rotated the key and losing the new value means
// losing Kion access until manual recovery.
func (o *Orchestrator) putWithRetries(ctx context.Context, s Secret) error {
	var lastErr error
	for attempt := range secretWriteTries {
		if attempt > 0 {
			delay := time.Duration(1<<uint(attempt-1)) * backoffBase
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
		}
		if err := o.secret.Put(ctx, s); err != nil {
			log.Printf("secret put attempt %d failed: %v", attempt+1, err)
			lastErr = err
			continue
		}
		return nil
	}
	if lastErr == nil {
		lastErr = errors.New("unknown error")
	}
	return lastErr
}

func (o *Orchestrator) publishMetric(ctx context.Context, days int) {
	if o.metrics == nil {
		return
	}
	if err := o.metrics.PublishDaysSinceRotation(ctx, float64(days)); err != nil {
		log.Printf("failed to publish DaysSinceRotation metric: %v", err)
	}
}

func (o *Orchestrator) notify(ctx context.Context, r notifications.RotationResult) {
	if o.notifier == nil {
		return
	}
	if err := o.notifier.SendRotationNotification(ctx, r); err != nil {
		log.Printf("failed to post Slack notification: %v", err)
	}
}

// daysSinceRotation returns whole days elapsed between the stored rotation
// timestamp and now. When the stored timestamp is missing (first bootstrap)
// or zero-valued, it returns math.MaxInt32 so the idempotency check always
// treats the secret as stale and proceeds to rotate.
func daysSinceRotation(now time.Time, past *time.Time) int {
	if past == nil || past.IsZero() {
		return math.MaxInt32
	}
	return int(now.Sub(*past).Hours() / 24)
}
