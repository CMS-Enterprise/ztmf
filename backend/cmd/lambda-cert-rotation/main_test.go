package main

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/service/acm"
	"github.com/aws/aws-sdk-go-v2/service/s3"
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

// fakeS3 is a programmable S3 stub used for the "incomplete bundle" path.
type fakeS3 struct {
	exists  map[string]bool
	headErr error
}

func (f fakeS3) HeadObject(_ context.Context, in *s3.HeadObjectInput, _ ...func(*s3.Options)) (*s3.HeadObjectOutput, error) {
	if f.headErr != nil {
		return nil, f.headErr
	}
	if f.exists[*in.Key] {
		return &s3.HeadObjectOutput{}, nil
	}
	return nil, notFoundErr{}
}
func (f fakeS3) GetObject(context.Context, *s3.GetObjectInput, ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	return nil, errors.New("unexpected GetObject")
}
func (f fakeS3) CopyObject(context.Context, *s3.CopyObjectInput, ...func(*s3.Options)) (*s3.CopyObjectOutput, error) {
	return nil, errors.New("unexpected CopyObject")
}
func (f fakeS3) DeleteObject(context.Context, *s3.DeleteObjectInput, ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
	return nil, errors.New("unexpected DeleteObject")
}

func TestHandleRecord_IncompleteBundleExitsQuietly(t *testing.T) {
	cfg := config.Config{
		CertBucket:    "ztmf-cert-rotation-dev",
		ArchivePrefix: "processed",
		EnvPrefixesToCfg: map[string]config.EnvConfig{
			"dev": {Domain: "dev.ztmf.cms.gov"},
		},
	}

	notif := &captureNotifier{}
	h := &handler{
		cfg: cfg,
		s3: fakeS3{
			// Only cert.pem present; key.pem and chain.pem missing.
			exists: map[string]bool{"dev/cert.pem": true},
		},
		acm:      panicACM{t: t},
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
