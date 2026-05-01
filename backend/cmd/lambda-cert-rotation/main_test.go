package main

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"io"
	"math/big"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
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
		{"cert.pem trigger exits silently (only chain.pem drives rotation)", "ztmf-cert-rotation-dev", "dev/cert.pem"},
		{"key.pem trigger exits silently (only chain.pem drives rotation)", "ztmf-cert-rotation-dev", "dev/key.pem"},
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

// forbiddenErr is what S3 returns for a missing object when the caller has
// no s3:ListBucket permission. The cert-rotation Lambda role intentionally
// omits ListBucket, so concurrent invocations observing a just-archived
// (deleted) bundle file see this shape rather than the NotFound code above.
type forbiddenErr struct{}

func (forbiddenErr) Error() string                 { return "Forbidden: Forbidden" }
func (forbiddenErr) ErrorCode() string             { return "Forbidden" }
func (forbiddenErr) ErrorMessage() string          { return "Forbidden" }
func (forbiddenErr) ErrorFault() smithy.ErrorFault { return smithy.FaultClient }

func TestHeadIfExists_TreatsForbiddenAsMissing(t *testing.T) {
	h := &handler{
		s3: stubHeadS3{err: forbiddenErr{}},
	}
	out, err := h.headIfExists(context.Background(), "bucket", "dev/cert.pem")
	if err != nil {
		t.Fatalf("403 should not surface as error, got: %v", err)
	}
	if out != nil {
		t.Fatalf("403 should resolve to nil HeadObjectOutput, got %+v", out)
	}
}

// stubHeadS3 returns a fixed error for every HeadObject call. Used by the
// targeted headIfExists unit test above.
type stubHeadS3 struct {
	err error
}

func (s stubHeadS3) HeadObject(context.Context, *s3.HeadObjectInput, ...func(*s3.Options)) (*s3.HeadObjectOutput, error) {
	return nil, s.err
}
func (s stubHeadS3) GetObject(context.Context, *s3.GetObjectInput, ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	return nil, errors.New("unexpected")
}
func (s stubHeadS3) CopyObject(context.Context, *s3.CopyObjectInput, ...func(*s3.Options)) (*s3.CopyObjectOutput, error) {
	return nil, errors.New("unexpected")
}
func (s stubHeadS3) DeleteObject(context.Context, *s3.DeleteObjectInput, ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
	return nil, errors.New("unexpected")
}

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
			// Only chain.pem present; cert.pem and key.pem missing.
			heads: map[string]*s3.HeadObjectOutput{
				"dev/chain.pem": {LastModified: &now},
			},
		},
		acm:      panicACM{t: t},
		secrets:  panicSecrets{t: t},
		notifier: notif,
	}

	rec := events.S3EventRecord{}
	rec.S3.Bucket.Name = "ztmf-cert-rotation-dev"
	rec.S3.Object.Key = "dev/chain.pem"

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
	rec.S3.Object.Key = "dev/chain.pem"

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

// ----------------------------------------------------------------------------
// End-to-end happy-path test using real PEM bytes and recording AWS fakes.
// The original Blocker A (backup-secret first-rotation failure) would have
// been caught by this test because the pre-fix code never reached
// PutSecretValue; the recording stub asserts it is called exactly once with a
// non-empty SecretString and a 32-character ClientRequestToken.

// happyPathBundle generates a valid (leaf, key, chain) PEM triple for tests.
// Duplicated from certvalidator/validator_test.go intentionally; the helper
// only exists for tests and is not in the package's public API. Keeping the
// generator here lets main_test.go be self-contained.
func happyPathBundle(t *testing.T, domain string, notBefore, notAfter time.Time) (certPEM, keyPEM, chainPEM []byte) {
	t.Helper()

	rootKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("gen root key: %v", err)
	}
	rootTmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "Happy Path Root"},
		NotBefore:             notBefore.Add(-time.Hour),
		NotAfter:              notAfter.Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		IsCA:                  true,
		BasicConstraintsValid: true,
	}
	rootDER, err := x509.CreateCertificate(rand.Reader, rootTmpl, rootTmpl, &rootKey.PublicKey, rootKey)
	if err != nil {
		t.Fatalf("create root: %v", err)
	}
	rootCert, err := x509.ParseCertificate(rootDER)
	if err != nil {
		t.Fatalf("parse root: %v", err)
	}

	interKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("gen inter key: %v", err)
	}
	interTmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(2),
		Subject:               pkix.Name{CommonName: "Happy Path Intermediate"},
		NotBefore:             notBefore.Add(-time.Hour),
		NotAfter:              notAfter.Add(180 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		IsCA:                  true,
		BasicConstraintsValid: true,
	}
	interDER, err := x509.CreateCertificate(rand.Reader, interTmpl, rootCert, &interKey.PublicKey, rootKey)
	if err != nil {
		t.Fatalf("create intermediate: %v", err)
	}
	interCert, err := x509.ParseCertificate(interDER)
	if err != nil {
		t.Fatalf("parse intermediate: %v", err)
	}

	leafKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("gen leaf key: %v", err)
	}
	leafTmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(3),
		Subject:               pkix.Name{CommonName: domain},
		DNSNames:              []string{domain},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}
	leafDER, err := x509.CreateCertificate(rand.Reader, leafTmpl, interCert, &leafKey.PublicKey, interKey)
	if err != nil {
		t.Fatalf("create leaf: %v", err)
	}
	leafKeyBytes, err := x509.MarshalECPrivateKey(leafKey)
	if err != nil {
		t.Fatalf("marshal leaf key: %v", err)
	}

	certPEM = bytes.TrimSpace(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: leafDER}))
	keyPEM = bytes.TrimSpace(pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: leafKeyBytes}))
	chainPEM = bytes.TrimSpace(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: interDER}))
	return
}

// servingS3 serves HeadObject LastModified metadata and GetObject bytes from
// an in-memory map keyed by object key. Copy and Delete succeed and count the
// calls. Optional failKey injects a per-key error for targeted failure tests.
type servingS3 struct {
	heads        map[string]*s3.HeadObjectOutput
	bodies       map[string][]byte
	copyCount    int32
	deleteCount  int32
	copyKeys     []string
	deleteKeys   []string
	deletedFirst bool
}

func (s *servingS3) HeadObject(_ context.Context, in *s3.HeadObjectInput, _ ...func(*s3.Options)) (*s3.HeadObjectOutput, error) {
	if out, ok := s.heads[*in.Key]; ok {
		return out, nil
	}
	return nil, notFoundErr{}
}
func (s *servingS3) GetObject(_ context.Context, in *s3.GetObjectInput, _ ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	body, ok := s.bodies[*in.Key]
	if !ok {
		return nil, notFoundErr{}
	}
	return &s3.GetObjectOutput{Body: io.NopCloser(bytes.NewReader(body))}, nil
}
func (s *servingS3) CopyObject(_ context.Context, in *s3.CopyObjectInput, _ ...func(*s3.Options)) (*s3.CopyObjectOutput, error) {
	atomic.AddInt32(&s.copyCount, 1)
	s.copyKeys = append(s.copyKeys, *in.Key)
	if s.deletedFirst {
		// Regression check: archive should never delete before copies finish.
		// If deletes run before copies, the partial-archive bug returns.
		return nil, errors.New("source deleted before copies completed")
	}
	return &s3.CopyObjectOutput{}, nil
}
func (s *servingS3) DeleteObject(_ context.Context, in *s3.DeleteObjectInput, _ ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
	atomic.AddInt32(&s.deleteCount, 1)
	if atomic.LoadInt32(&s.copyCount) == 0 {
		s.deletedFirst = true
	}
	s.deleteKeys = append(s.deleteKeys, *in.Key)
	return &s3.DeleteObjectOutput{}, nil
}

// recordingACM records every ImportCertificate call.
type recordingACM struct {
	calls []*acm.ImportCertificateInput
}

func (r *recordingACM) ImportCertificate(_ context.Context, in *acm.ImportCertificateInput, _ ...func(*acm.Options)) (*acm.ImportCertificateOutput, error) {
	r.calls = append(r.calls, in)
	return &acm.ImportCertificateOutput{CertificateArn: in.CertificateArn}, nil
}

// recordingSecrets records every PutSecretValue call.
type recordingSecrets struct {
	calls []*secretsmanager.PutSecretValueInput
}

func (r *recordingSecrets) PutSecretValue(_ context.Context, in *secretsmanager.PutSecretValueInput, _ ...func(*secretsmanager.Options)) (*secretsmanager.PutSecretValueOutput, error) {
	r.calls = append(r.calls, in)
	return &secretsmanager.PutSecretValueOutput{}, nil
}

func TestHandleRecord_HappyPath_EndToEnd(t *testing.T) {
	domain := "dev.ztmf.cms.gov"
	now := time.Now().UTC()
	certPEM, keyPEM, chainPEM := happyPathBundle(t, domain, now.Add(-time.Hour), now.Add(365*24*time.Hour))

	backupArn := "arn:aws:secretsmanager:us-east-1:111111111111:secret:ztmf-cert-rotation-backup-dev-abcdef"
	acmArn := "arn:aws:acm:us-east-1:111111111111:certificate/abcdef00-1111-2222-3333-abcdef012345"

	cfg := config.Config{
		CertBucket:    "ztmf-cert-rotation-dev",
		ArchivePrefix: "processed",
		DryRun:        false,
		EnvPrefixesToCfg: map[string]config.EnvConfig{
			"dev": {
				Domain:            domain,
				AcmCertificateArn: acmArn,
				BackupSecretArn:   backupArn,
			},
		},
	}

	fresh := now
	s3Fake := &servingS3{
		heads: map[string]*s3.HeadObjectOutput{
			"dev/cert.pem":  {LastModified: aws.Time(fresh)},
			"dev/key.pem":   {LastModified: aws.Time(fresh.Add(10 * time.Second))},
			"dev/chain.pem": {LastModified: aws.Time(fresh.Add(20 * time.Second))},
		},
		bodies: map[string][]byte{
			"dev/cert.pem":  certPEM,
			"dev/key.pem":   keyPEM,
			"dev/chain.pem": chainPEM,
		},
	}
	acmFake := &recordingACM{}
	secretsFake := &recordingSecrets{}
	notif := &captureNotifier{}

	h := &handler{
		cfg:      cfg,
		s3:       s3Fake,
		acm:      acmFake,
		secrets:  secretsFake,
		notifier: notif,
	}

	rec := events.S3EventRecord{}
	rec.S3.Bucket.Name = "ztmf-cert-rotation-dev"
	rec.S3.Object.Key = "dev/chain.pem"

	if err := h.handleRecord(context.Background(), rec); err != nil {
		t.Fatalf("handleRecord returned error: %v", err)
	}

	// ACM import called once with the expected ARN and the exact PEM bytes.
	if len(acmFake.calls) != 1 {
		t.Fatalf("want 1 ImportCertificate call, got %d", len(acmFake.calls))
	}
	call := acmFake.calls[0]
	if call.CertificateArn == nil || *call.CertificateArn != acmArn {
		t.Errorf("ACM arn = %v, want %s", call.CertificateArn, acmArn)
	}
	if !bytes.Equal(call.Certificate, certPEM) {
		t.Errorf("ACM Certificate bytes do not match input cert.pem")
	}
	if !bytes.Equal(call.PrivateKey, keyPEM) {
		t.Errorf("ACM PrivateKey bytes do not match input key.pem")
	}
	if !bytes.Equal(call.CertificateChain, chainPEM) {
		t.Errorf("ACM CertificateChain bytes do not match input chain.pem")
	}

	// PutSecretValue called exactly once. This is the regression guard for
	// the original Blocker A: pre-fix code never reached PutSecretValue
	// because secrets.NewSecret failed at construction.
	if len(secretsFake.calls) != 1 {
		t.Fatalf("want 1 PutSecretValue call, got %d", len(secretsFake.calls))
	}
	sv := secretsFake.calls[0]
	if sv.SecretId == nil || *sv.SecretId != backupArn {
		t.Errorf("backup SecretId = %v, want %s", sv.SecretId, backupArn)
	}
	if sv.SecretString == nil || *sv.SecretString == "" {
		t.Errorf("backup SecretString is empty")
	}
	if sv.ClientRequestToken == nil || len(*sv.ClientRequestToken) != 32 {
		t.Errorf("backup ClientRequestToken must be 32 chars; got %v", sv.ClientRequestToken)
	}
	if !strings.Contains(*sv.SecretString, `"domain":"dev.ztmf.cms.gov"`) {
		t.Errorf("backup SecretString missing expected domain field; got %q", *sv.SecretString)
	}

	// Archive phase ordering: all three copies happen before any delete.
	if s3Fake.copyCount != 3 {
		t.Errorf("want 3 CopyObject calls, got %d", s3Fake.copyCount)
	}
	if s3Fake.deleteCount != 3 {
		t.Errorf("want 3 DeleteObject calls, got %d", s3Fake.deleteCount)
	}
	if s3Fake.deletedFirst {
		t.Errorf("delete ran before all copies completed; partial-archive regression")
	}

	// Exactly one success notification, no validation/infra failure.
	if len(notif.calls) != 1 {
		t.Fatalf("want 1 notification, got %d: %+v", len(notif.calls), notif.calls)
	}
	got := notif.calls[0]
	if !got.Success {
		t.Errorf("notification should be Success; got %+v", got)
	}
	if got.DryRun {
		t.Errorf("notification should not be DryRun; got %+v", got)
	}
	if got.ValidationFailed {
		t.Errorf("notification should not be ValidationFailed; got %+v", got)
	}
	if got.AcmCertificateArn != acmArn {
		t.Errorf("notification AcmCertificateArn = %q, want %q", got.AcmCertificateArn, acmArn)
	}
}
