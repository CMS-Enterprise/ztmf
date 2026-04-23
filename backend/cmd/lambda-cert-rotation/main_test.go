package main

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/service/acm"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/smithy-go"

	"github.com/CMS-Enterprise/ztmf/backend/cmd/lambda-cert-rotation/internal/config"
	"github.com/CMS-Enterprise/ztmf/backend/internal/notifications"
)

func TestMatchPrefix(t *testing.T) {
	cfg := config.Config{
		EnvPrefixesToCfg: map[string]config.EnvConfig{
			"dev": {Domain: "dev.ztmf.cms.gov"},
		},
	}
	h := &handler{cfg: cfg}

	cases := []struct {
		name       string
		key        string
		wantPrefix string
		wantOK     bool
	}{
		{"exact match under prefix", "dev/cert.pem", "dev", true},
		{"leading slash tolerated", "/dev/cert.pem", "dev", true},
		{"unknown prefix reports not ok", "prod/cert.pem", "", false},
		{"no prefix segment reports not ok", "cert.pem", "", false},
		{"empty key reports not ok", "", "", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			prefix, _, ok := h.matchPrefix(tc.key)
			if ok != tc.wantOK {
				t.Fatalf("ok = %t, want %t", ok, tc.wantOK)
			}
			// Prefix is only meaningful when ok is true; ignore otherwise.
			if ok && prefix != tc.wantPrefix {
				t.Errorf("prefix = %q, want %q", prefix, tc.wantPrefix)
			}
		})
	}
}

func TestPathEscape(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"bucket/dev/cert.pem", "bucket/dev/cert.pem"},
		{"bucket/dev/name with space.pem", "bucket/dev/name%20with%20space.pem"},
		{"bucket/unicode/fü.pem", "bucket/unicode/f%c3%bc.pem"},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			got := pathEscape(tc.in)
			// url.PathEscape lowercases hex; accept either case.
			if !equalFoldEscaped(got, tc.want) {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func equalFoldEscaped(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	// url.URL escaping may produce uppercase hex; compare case-insensitively
	// so the test tolerates either.
	return toLower(a) == toLower(b)
}

func toLower(s string) string {
	out := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		out[i] = c
	}
	return string(out)
}

func TestPayloadRequestToken_DeterministicLength32(t *testing.T) {
	payload := []byte(`{"foo":"bar","baz":"qux"}`)
	a := payloadRequestToken(payload)
	b := payloadRequestToken(payload)
	if a != b {
		t.Fatalf("token not deterministic: %q vs %q", a, b)
	}
	if len(a) != 32 {
		t.Fatalf("token length %d, want 32 (Secrets Manager requires 32-64)", len(a))
	}
	other := payloadRequestToken([]byte(`{"foo":"different"}`))
	if other == a {
		t.Fatalf("different payloads produced identical token")
	}
}

func TestVerifyBundleFreshness(t *testing.T) {
	base := time.Date(2026, 4, 23, 12, 0, 0, 0, time.UTC)
	headAt := func(offset time.Duration) *s3.HeadObjectOutput {
		t := base.Add(offset)
		return &s3.HeadObjectOutput{LastModified: &t}
	}

	cases := []struct {
		name    string
		cert    *s3.HeadObjectOutput
		key     *s3.HeadObjectOutput
		chain   *s3.HeadObjectOutput
		window  time.Duration
		wantErr bool
	}{
		{
			name:   "all three within window",
			cert:   headAt(0),
			key:    headAt(30 * time.Second),
			chain:  headAt(90 * time.Second),
			window: time.Hour,
		},
		{
			name:   "equal timestamps",
			cert:   headAt(0),
			key:    headAt(0),
			chain:  headAt(0),
			window: time.Hour,
		},
		{
			name:    "cert fresh, key 3 days stale",
			cert:    headAt(0),
			key:     headAt(-72 * time.Hour),
			chain:   headAt(-72 * time.Hour),
			window:  time.Hour,
			wantErr: true,
		},
		{
			name:    "chain just over window",
			cert:    headAt(0),
			key:     headAt(0),
			chain:   headAt(time.Hour + time.Second),
			window:  time.Hour,
			wantErr: true,
		},
		{
			name:    "missing last modified",
			cert:    headAt(0),
			key:     &s3.HeadObjectOutput{},
			chain:   headAt(0),
			window:  time.Hour,
			wantErr: true,
		},
		{
			name:    "nil head treated as missing",
			cert:    nil,
			key:     headAt(0),
			chain:   headAt(0),
			window:  time.Hour,
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := verifyBundleFreshness(tc.cert, tc.key, tc.chain, tc.window)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

// captureNotifier records every notification for assertion in tests.
type captureNotifier struct {
	calls []notifications.CertRotationResult
}

func (c *captureNotifier) SendCertRotationNotification(_ context.Context, r notifications.CertRotationResult) error {
	c.calls = append(c.calls, r)
	return nil
}

// panicS3 fails the test if any S3 call is made. Used to prove early-exit
// branches in handleRecord never reach the S3 client.
type panicS3 struct {
	t *testing.T
}

func (p panicS3) HeadObject(context.Context, *s3.HeadObjectInput, ...func(*s3.Options)) (*s3.HeadObjectOutput, error) {
	p.t.Fatalf("unexpected HeadObject call")
	return nil, nil
}
func (p panicS3) GetObject(context.Context, *s3.GetObjectInput, ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	p.t.Fatalf("unexpected GetObject call")
	return nil, nil
}
func (p panicS3) CopyObject(context.Context, *s3.CopyObjectInput, ...func(*s3.Options)) (*s3.CopyObjectOutput, error) {
	p.t.Fatalf("unexpected CopyObject call")
	return nil, nil
}
func (p panicS3) DeleteObject(context.Context, *s3.DeleteObjectInput, ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
	p.t.Fatalf("unexpected DeleteObject call")
	return nil, nil
}

// panicACM fails the test if any ACM call is made.
type panicACM struct {
	t *testing.T
}

func (p panicACM) ImportCertificate(context.Context, *acm.ImportCertificateInput, ...func(*acm.Options)) (*acm.ImportCertificateOutput, error) {
	p.t.Fatalf("unexpected ImportCertificate call")
	return nil, nil
}

// panicSecrets fails the test if any Secrets Manager call is made.
type panicSecrets struct {
	t *testing.T
}

func (p panicSecrets) PutSecretValue(context.Context, *secretsmanager.PutSecretValueInput, ...func(*secretsmanager.Options)) (*secretsmanager.PutSecretValueOutput, error) {
	p.t.Fatalf("unexpected PutSecretValue call")
	return nil, nil
}

func TestHandleRecord_EarlyExits(t *testing.T) {
	cfg := config.Config{
		CertBucket:    "ztmf-cert-rotation-dev",
		ArchivePrefix: "processed",
		EnvPrefixesToCfg: map[string]config.EnvConfig{
			"dev": {Domain: "dev.ztmf.cms.gov"},
		},
	}

	cases := []struct {
		name   string
		bucket string
		key    string
	}{
		{"wrong bucket is ignored", "some-other-bucket", "dev/cert.pem"},
		{"non-pem suffix is ignored", "ztmf-cert-rotation-dev", "dev/notes.txt"},
		{"unknown env prefix is ignored", "ztmf-cert-rotation-dev", "staging/cert.pem"},
		{"non-bundle file name under known prefix is ignored", "ztmf-cert-rotation-dev", "dev/other.pem"},
		{"empty bucket is ignored", "", "dev/cert.pem"},
		{"empty key is ignored", "ztmf-cert-rotation-dev", ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			notif := &captureNotifier{}
			h := &handler{
				cfg:      cfg,
				s3:       panicS3{t: t},
				acm:      panicACM{t: t},
				secrets:  panicSecrets{t: t},
				notifier: notif,
			}

			rec := events.S3EventRecord{}
			rec.S3.Bucket.Name = tc.bucket
			rec.S3.Object.Key = tc.key

			if err := h.handleRecord(context.Background(), rec); err != nil {
				t.Fatalf("handleRecord returned error: %v", err)
			}
			if len(notif.calls) != 0 {
				t.Errorf("unexpected notifications: %+v", notif.calls)
			}
		})
	}
}

// notFoundErr mirrors the smithy.APIError surface that the real S3 client
// returns for a 404 HeadObject. Handler uses errors.As to detect it.
type notFoundErr struct{}

func (notFoundErr) Error() string                 { return "NotFound: Not Found" }
func (notFoundErr) ErrorCode() string             { return "NotFound" }
func (notFoundErr) ErrorMessage() string          { return "Not Found" }
func (notFoundErr) ErrorFault() smithy.ErrorFault { return smithy.FaultClient }

// fakeS3 is a programmable S3 stub. Keys present in `heads` return the
// associated HeadObjectOutput; keys absent return 404. Copy/Delete simply
// succeed unless failKeys contains the key.
type fakeS3 struct {
	heads    map[string]*s3.HeadObjectOutput
	headErr  error
	failCopy map[string]error
	failDel  map[string]error
}

func (f fakeS3) HeadObject(_ context.Context, in *s3.HeadObjectInput, _ ...func(*s3.Options)) (*s3.HeadObjectOutput, error) {
	if f.headErr != nil {
		return nil, f.headErr
	}
	if out, ok := f.heads[*in.Key]; ok {
		return out, nil
	}
	return nil, notFoundErr{}
}
func (f fakeS3) GetObject(context.Context, *s3.GetObjectInput, ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	return nil, errors.New("unexpected GetObject")
}
func (f fakeS3) CopyObject(_ context.Context, in *s3.CopyObjectInput, _ ...func(*s3.Options)) (*s3.CopyObjectOutput, error) {
	if err, ok := f.failCopy[*in.Key]; ok {
		return nil, err
	}
	return &s3.CopyObjectOutput{}, nil
}
func (f fakeS3) DeleteObject(_ context.Context, in *s3.DeleteObjectInput, _ ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
	if err, ok := f.failDel[*in.Key]; ok {
		return nil, err
	}
	return &s3.DeleteObjectOutput{}, nil
}

func TestHandleRecord_IncompleteBundleExitsQuietly(t *testing.T) {
	cfg := config.Config{
		CertBucket:    "ztmf-cert-rotation-dev",
		ArchivePrefix: "processed",
		EnvPrefixesToCfg: map[string]config.EnvConfig{
			"dev": {Domain: "dev.ztmf.cms.gov"},
		},
	}

	now := time.Now().UTC()
	notif := &captureNotifier{}
	h := &handler{
		cfg: cfg,
		s3: fakeS3{
			// Only cert.pem present; key.pem and chain.pem missing.
			heads: map[string]*s3.HeadObjectOutput{
				"dev/cert.pem": {LastModified: &now},
			},
		},
		acm:      panicACM{t: t},
		secrets:  panicSecrets{t: t},
		notifier: notif,
	}

	rec := events.S3EventRecord{}
	rec.S3.Bucket.Name = "ztmf-cert-rotation-dev"
	rec.S3.Object.Key = "dev/cert.pem"

	if err := h.handleRecord(context.Background(), rec); err != nil {
		t.Fatalf("handleRecord returned error: %v", err)
	}
	if len(notif.calls) != 0 {
		t.Errorf("incomplete bundle should not notify; got %+v", notif.calls)
	}
}

func TestHandleRecord_StaleBundleIsValidationFailure(t *testing.T) {
	cfg := config.Config{
		CertBucket:    "ztmf-cert-rotation-dev",
		ArchivePrefix: "processed",
		EnvPrefixesToCfg: map[string]config.EnvConfig{
			"dev": {Domain: "dev.ztmf.cms.gov"},
		},
	}

	// Operator uploaded a fresh cert.pem over 3-day-old key.pem/chain.pem.
	// This is the exact Codex scenario the freshness check defends against.
	fresh := time.Now().UTC()
	stale := fresh.Add(-72 * time.Hour)
	notif := &captureNotifier{}
	h := &handler{
		cfg: cfg,
		s3: fakeS3{
			heads: map[string]*s3.HeadObjectOutput{
				"dev/cert.pem":  {LastModified: &fresh},
				"dev/key.pem":   {LastModified: &stale},
				"dev/chain.pem": {LastModified: &stale},
			},
		},
		// ACM and Secrets must not be called when the bundle is rejected
		// as stale; the panic stubs prove the short-circuit.
		acm:      panicACM{t: t},
		secrets:  panicSecrets{t: t},
		notifier: notif,
	}

	rec := events.S3EventRecord{}
	rec.S3.Bucket.Name = "ztmf-cert-rotation-dev"
	rec.S3.Object.Key = "dev/cert.pem"

	if err := h.handleRecord(context.Background(), rec); err != nil {
		t.Fatalf("stale bundle should be a validation failure (nil return), got: %v", err)
	}
	if len(notif.calls) != 1 {
		t.Fatalf("want 1 notification, got %d: %+v", len(notif.calls), notif.calls)
	}
	got := notif.calls[0]
	if !got.ValidationFailed {
		t.Errorf("notification should have ValidationFailed=true; got %+v", got)
	}
	if got.Success {
		t.Errorf("notification should not be Success; got %+v", got)
	}
	if got.Environment != "dev" {
		t.Errorf("environment = %q, want dev", got.Environment)
	}
}
