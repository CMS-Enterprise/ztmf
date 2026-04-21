package rotate

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/CMS-Enterprise/ztmf/backend/internal/notifications"
)

// fakeSecret stores a JSON-encoded Secret and simulates read/write failures on demand.
type fakeSecret struct {
	payload  []byte
	putErr   error
	putFails int
	putCalls int
}

func newFakeSecret(s Secret) *fakeSecret {
	b, _ := json.Marshal(s)
	return &fakeSecret{payload: b}
}

func (f *fakeSecret) Unmarshal(v any) error {
	return json.Unmarshal(f.payload, v)
}

func (f *fakeSecret) Put(ctx context.Context, v any) error {
	f.putCalls++
	if f.putCalls <= f.putFails {
		return f.putErr
	}
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	f.payload = b
	return nil
}

type fakeKion struct {
	newKey string
	err    error
	calls  int
}

func (f *fakeKion) Rotate(ctx context.Context, current string) (string, error) {
	f.calls++
	if f.err != nil {
		return "", f.err
	}
	return f.newKey, nil
}

type fakeNotifier struct {
	results []notifications.RotationResult
}

func (f *fakeNotifier) SendRotationNotification(ctx context.Context, r notifications.RotationResult) error {
	f.results = append(f.results, r)
	return nil
}

type fakeMetrics struct {
	days []float64
}

func (f *fakeMetrics) PublishDaysSinceRotation(ctx context.Context, d float64) error {
	f.days = append(f.days, d)
	return nil
}

func fixedNow(t time.Time) func() time.Time {
	return func() time.Time { return t }
}

func buildOrchestrator(t *testing.T, secret *fakeSecret, k *fakeKion, in Input) (*Orchestrator, *fakeNotifier, *fakeMetrics) {
	t.Helper()
	n := &fakeNotifier{}
	m := &fakeMetrics{}
	now := fixedNow(time.Date(2026, 4, 21, 6, 0, 0, 0, time.UTC))
	o := New(in, secret, func(string) KionClient { return k }, n, m, now)
	return o, n, m
}

func TestRun_IdempotencySkip(t *testing.T) {
	s := newFakeSecret(Secret{
		APIKey:    "abc",
		BaseURL:   "https://kion",
		RotatedAt: time.Date(2026, 4, 19, 6, 0, 0, 0, time.UTC), // 2 days ago
	})
	k := &fakeKion{newKey: "should-not-be-used"}

	o, n, m := buildOrchestrator(t, s, k, Input{
		Environment:     "dev",
		SecretName:      "ztmf_kion_dev",
		RotateAfterDays: 4,
	})

	res, err := o.Run(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.Skipped {
		t.Errorf("expected skipped, got %+v", res)
	}
	if k.calls != 0 {
		t.Errorf("expected 0 Kion calls, got %d", k.calls)
	}
	if len(n.results) != 1 || !n.results[0].Skipped {
		t.Errorf("expected one Skipped notification, got %+v", n.results)
	}
	if len(m.days) != 1 || m.days[0] != 2 {
		t.Errorf("expected metric=2, got %v", m.days)
	}
}

func TestRun_ForceBypassesIdempotency(t *testing.T) {
	s := newFakeSecret(Secret{
		APIKey:    "abc",
		BaseURL:   "https://kion",
		RotatedAt: time.Date(2026, 4, 20, 6, 0, 0, 0, time.UTC), // 1 day ago
	})
	k := &fakeKion{newKey: "rotated"}

	o, n, _ := buildOrchestrator(t, s, k, Input{
		Environment:     "dev",
		SecretName:      "ztmf_kion_dev",
		RotateAfterDays: 4,
		Force:           true,
	})

	res, err := o.Run(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.Rotated {
		t.Errorf("expected rotated, got %+v", res)
	}
	if k.calls != 1 {
		t.Errorf("expected 1 Kion call, got %d", k.calls)
	}
	var reloaded Secret
	if err := s.Unmarshal(&reloaded); err != nil {
		t.Fatal(err)
	}
	if reloaded.APIKey != "rotated" {
		t.Errorf("stored api_key = %q, want rotated", reloaded.APIKey)
	}
	if len(n.results) != 1 || !n.results[0].Success || n.results[0].Skipped {
		t.Errorf("expected single success notification, got %+v", n.results)
	}
}

func TestRun_DryRun_NoRotateNoWrite(t *testing.T) {
	s := newFakeSecret(Secret{
		APIKey:    "abc",
		BaseURL:   "https://kion",
		RotatedAt: time.Date(2026, 4, 10, 6, 0, 0, 0, time.UTC), // old enough to rotate
	})
	k := &fakeKion{newKey: "would-be-new"}

	o, n, _ := buildOrchestrator(t, s, k, Input{
		Environment:     "dev",
		SecretName:      "ztmf_kion_dev",
		RotateAfterDays: 4,
		DryRun:          true,
	})

	res, err := o.Run(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Rotated || res.Skipped {
		t.Errorf("dry-run should be neither rotated nor skipped, got %+v", res)
	}
	if k.calls != 0 {
		t.Errorf("dry-run must not call Kion, got %d", k.calls)
	}
	if s.putCalls != 0 {
		t.Errorf("dry-run must not write secret, got %d", s.putCalls)
	}
	if len(n.results) != 1 || !n.results[0].DryRun {
		t.Errorf("expected dry-run notification, got %+v", n.results)
	}
}

func TestRun_HappyPath(t *testing.T) {
	s := newFakeSecret(Secret{
		APIKey:    "abc",
		BaseURL:   "https://kion",
		RotatedAt: time.Date(2026, 4, 16, 6, 0, 0, 0, time.UTC), // 5 days ago
	})
	k := &fakeKion{newKey: "shiny-new"}

	o, n, m := buildOrchestrator(t, s, k, Input{
		Environment:     "prod",
		SecretName:      "ztmf_kion_prod",
		RotateAfterDays: 4,
	})

	res, err := o.Run(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.Rotated {
		t.Errorf("expected rotated, got %+v", res)
	}
	var reloaded Secret
	if err := s.Unmarshal(&reloaded); err != nil {
		t.Fatal(err)
	}
	if reloaded.APIKey != "shiny-new" {
		t.Errorf("stored api_key = %q, want shiny-new", reloaded.APIKey)
	}
	if reloaded.BaseURL != "https://kion" {
		t.Errorf("base_url changed: %q", reloaded.BaseURL)
	}
	if len(n.results) != 1 || !n.results[0].Success {
		t.Errorf("expected success notification, got %+v", n.results)
	}
	if len(m.days) != 1 || m.days[0] != 0 {
		t.Errorf("expected metric=0, got %v", m.days)
	}
}

func TestRun_KionFailure_NoSecretWrite(t *testing.T) {
	s := newFakeSecret(Secret{
		APIKey:    "abc",
		BaseURL:   "https://kion",
		RotatedAt: time.Date(2026, 4, 16, 6, 0, 0, 0, time.UTC),
	})
	k := &fakeKion{err: errors.New("kion 401")}

	o, n, _ := buildOrchestrator(t, s, k, Input{
		Environment:     "dev",
		SecretName:      "ztmf_kion_dev",
		RotateAfterDays: 4,
	})

	_, err := o.Run(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if s.putCalls != 0 {
		t.Errorf("secret must not be written on Kion failure, got %d puts", s.putCalls)
	}
	if len(n.results) != 1 || n.results[0].Success || n.results[0].RecoveryKey != "" {
		t.Errorf("expected plain failure notification, got %+v", n.results)
	}
}

func TestRun_PutFailureEmitsRecoveryKey(t *testing.T) {
	s := newFakeSecret(Secret{
		APIKey:    "abc",
		BaseURL:   "https://kion",
		RotatedAt: time.Date(2026, 4, 16, 6, 0, 0, 0, time.UTC),
	})
	s.putErr = errors.New("secrets manager down")
	s.putFails = 99 // all attempts fail

	k := &fakeKion{newKey: "rotated-but-unpersisted"}

	// Use shorter delays by passing a short-deadline context. The retries wait
	// 1s,2s,4s,8s between attempts, so allow ~20s budget.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	o, n, _ := buildOrchestrator(t, s, k, Input{
		Environment:     "prod",
		SecretName:      "ztmf_kion_prod",
		RotateAfterDays: 4,
	})

	_, err := o.Run(ctx)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if len(n.results) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(n.results))
	}
	got := n.results[0]
	if got.Success {
		t.Error("expected failure notification")
	}
	if got.RecoveryKey != "rotated-but-unpersisted" {
		t.Errorf("recovery key = %q, want rotated-but-unpersisted", got.RecoveryKey)
	}
	if s.putCalls != secretWriteTries {
		t.Errorf("expected %d put attempts, got %d", secretWriteTries, s.putCalls)
	}
}

func TestRun_EmptyAPIKeyFails(t *testing.T) {
	s := newFakeSecret(Secret{BaseURL: "https://kion"}) // no api_key
	k := &fakeKion{}
	o, _, _ := buildOrchestrator(t, s, k, Input{SecretName: "s", RotateAfterDays: 4})
	if _, err := o.Run(context.Background()); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestRun_EmptyBaseURLFails(t *testing.T) {
	s := newFakeSecret(Secret{APIKey: "abc"}) // no base_url
	k := &fakeKion{}
	o, _, _ := buildOrchestrator(t, s, k, Input{SecretName: "s", RotateAfterDays: 4})
	if _, err := o.Run(context.Background()); err == nil {
		t.Fatal("expected error, got nil")
	}
}
