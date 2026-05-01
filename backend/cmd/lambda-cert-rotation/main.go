// Command lambda-cert-rotation validates and imports TLS certificate bundles
// uploaded to a well-known S3 prefix into ACM. Expected uploads under
// <prefix>/{cert.pem,key.pem,chain.pem}: once all three are present the Lambda
// validates the bundle, re-imports to ACM over the configured ARN, backs the
// bundle up to Secrets Manager, and archives the source files.
//
// Notifications are emitted to Slack via the shared notifications.SlackNotifier,
// which reads the ztmf_slack_webhook secret identified by SLACK_SECRET_ID. The
// Lambda is idempotent at the ACM layer; rapid upload of the three parts may
// trigger three invocations but only the one that observes all three files
// does meaningful work.
package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/acm"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/smithy-go"

	"github.com/CMS-Enterprise/ztmf/backend/cmd/lambda-cert-rotation/internal/awsclients"
	"github.com/CMS-Enterprise/ztmf/backend/cmd/lambda-cert-rotation/internal/certvalidator"
	"github.com/CMS-Enterprise/ztmf/backend/cmd/lambda-cert-rotation/internal/config"
	"github.com/CMS-Enterprise/ztmf/backend/internal/notifications"
)

const (
	certKeyName  = "cert.pem"
	keyKeyName   = "key.pem"
	chainKeyName = "chain.pem"

	// bundleFreshnessWindow bounds how far apart cert.pem, key.pem, and
	// chain.pem uploads may be before the bundle is considered stale. A stale
	// bundle typically indicates leftover files from a previously-failed
	// rotation that were paired with a freshly-uploaded replacement. Rejecting
	// those pairings prevents a valid new cert from being imported with an
	// outdated intermediate chain.
	bundleFreshnessWindow = time.Hour
)

// notifier abstracts notifications.SlackNotifier so handleRecord can be
// exercised without hitting the webhook secret or Slack itself.
type notifier interface {
	SendCertRotationNotification(ctx context.Context, r notifications.CertRotationResult) error
}

// nopNotifier is used when Slack configuration cannot be resolved; it lets
// rotation continue without blocking on notification infrastructure.
type nopNotifier struct{}

func (nopNotifier) SendCertRotationNotification(context.Context, notifications.CertRotationResult) error {
	return nil
}

// s3API is the minimal subset of the S3 client used by handleRecord. Defined
// here so tests can substitute a fake without the real SDK.
type s3API interface {
	HeadObject(ctx context.Context, in *s3.HeadObjectInput, opts ...func(*s3.Options)) (*s3.HeadObjectOutput, error)
	GetObject(ctx context.Context, in *s3.GetObjectInput, opts ...func(*s3.Options)) (*s3.GetObjectOutput, error)
	CopyObject(ctx context.Context, in *s3.CopyObjectInput, opts ...func(*s3.Options)) (*s3.CopyObjectOutput, error)
	DeleteObject(ctx context.Context, in *s3.DeleteObjectInput, opts ...func(*s3.Options)) (*s3.DeleteObjectOutput, error)
}

// acmAPI is the minimal subset of the ACM client used by handleRecord.
type acmAPI interface {
	ImportCertificate(ctx context.Context, in *acm.ImportCertificateInput, opts ...func(*acm.Options)) (*acm.ImportCertificateOutput, error)
}

// secretsAPI is the minimal subset of Secrets Manager used by handleRecord.
type secretsAPI interface {
	PutSecretValue(ctx context.Context, in *secretsmanager.PutSecretValueInput, opts ...func(*secretsmanager.Options)) (*secretsmanager.PutSecretValueOutput, error)
}

type handler struct {
	cfg      config.Config
	s3       s3API
	acm      acmAPI
	secrets  secretsAPI
	notifier notifier
}

func main() {
	ctx := context.Background()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	clients, err := awsclients.New(ctx)
	if err != nil {
		log.Fatalf("aws clients: %v", err)
	}

	var n notifier
	slackNotifier, err := notifications.NewSlackNotifier(ctx)
	if err != nil {
		// Notification failure must not block rotation; log and fall back.
		log.Printf("Slack notifier unavailable, continuing without notifications: %v", err)
		n = nopNotifier{}
	} else {
		n = slackNotifier
	}

	lambda.Start((&handler{
		cfg:      cfg,
		s3:       clients.S3,
		acm:      clients.ACM,
		secrets:  clients.Secrets,
		notifier: n,
	}).Handle)
}

func (h *handler) Handle(ctx context.Context, evt events.S3Event) error {
	for _, r := range evt.Records {
		if err := h.handleRecord(ctx, r); err != nil {
			// Returning an error triggers Lambda retry; only transient AWS
			// errors (S3/ACM/Secrets Manager) take this path. Validation
			// problems are reported via Slack and return nil.
			return err
		}
	}
	return nil
}

func (h *handler) handleRecord(ctx context.Context, r events.S3EventRecord) error {
	bucket := r.S3.Bucket.Name
	key, err := url.QueryUnescape(r.S3.Object.Key)
	if err != nil {
		key = r.S3.Object.Key
	}

	if bucket == "" || key == "" {
		return nil
	}
	if bucket != h.cfg.CertBucket {
		return nil
	}
	if !strings.HasSuffix(strings.ToLower(key), ".pem") {
		return nil
	}

	envPrefix, envCfg, ok := h.matchPrefix(key)
	if !ok {
		return nil
	}

	base := path.Base(key)
	if base != certKeyName && base != keyKeyName && base != chainKeyName {
		return nil
	}

	// Only the chain.pem upload event drives the rotation. The cert.pem
	// and key.pem upload events exit silently. The operator contract
	// (documented in README.md) requires uploading all three files
	// together for any rotation, so requiring chain.pem as the trigger
	// is safe; in exchange we eliminate the three-way concurrent race
	// on validate / archive / notify and produce exactly one Slack
	// notification per real rotation regardless of upload ordering.
	if base != chainKeyName {
		return nil
	}

	s3Location := fmt.Sprintf("s3://%s/%s/", bucket, envPrefix)

	wantCert := path.Join(envPrefix, certKeyName)
	wantKey := path.Join(envPrefix, keyKeyName)
	wantChain := path.Join(envPrefix, chainKeyName)

	certHead, err := h.headIfExists(ctx, bucket, wantCert)
	if err != nil {
		return h.notifyInfraFailure(ctx, envPrefix, envCfg.Domain, s3Location, fmt.Errorf("head cert.pem: %w", err))
	}
	keyHead, err := h.headIfExists(ctx, bucket, wantKey)
	if err != nil {
		return h.notifyInfraFailure(ctx, envPrefix, envCfg.Domain, s3Location, fmt.Errorf("head key.pem: %w", err))
	}
	chainHead, err := h.headIfExists(ctx, bucket, wantChain)
	if err != nil {
		return h.notifyInfraFailure(ctx, envPrefix, envCfg.Domain, s3Location, fmt.Errorf("head chain.pem: %w", err))
	}
	if certHead == nil || keyHead == nil || chainHead == nil {
		// Quiet exit until the full bundle is present.
		return nil
	}

	// Freshness check defends against partial-rotation poison: if a prior run
	// archived only some of cert/key/chain before failing, a subsequent upload
	// of one file would otherwise pair a fresh cert with a stale intermediate
	// chain and silently import an inconsistent bundle into ACM.
	if err := verifyBundleFreshness(certHead, keyHead, chainHead, bundleFreshnessWindow); err != nil {
		return h.notifyValidationFailure(ctx, envPrefix, envCfg.Domain, s3Location, err)
	}

	certPEM, err := h.getObjectBytes(ctx, bucket, wantCert)
	if err != nil {
		return h.notifyInfraFailure(ctx, envPrefix, envCfg.Domain, s3Location, fmt.Errorf("read cert.pem: %w", err))
	}
	keyPEM, err := h.getObjectBytes(ctx, bucket, wantKey)
	if err != nil {
		return h.notifyInfraFailure(ctx, envPrefix, envCfg.Domain, s3Location, fmt.Errorf("read key.pem: %w", err))
	}
	chainPEM, err := h.getObjectBytes(ctx, bucket, wantChain)
	if err != nil {
		return h.notifyInfraFailure(ctx, envPrefix, envCfg.Domain, s3Location, fmt.Errorf("read chain.pem: %w", err))
	}

	now := time.Now().UTC()
	res, err := certvalidator.Validate(certvalidator.Bundle{
		CertPEM:  certPEM,
		KeyPEM:   keyPEM,
		ChainPEM: chainPEM,
	}, envCfg.Domain, now)
	if err != nil {
		return h.notifyValidationFailure(ctx, envPrefix, envCfg.Domain, s3Location, err)
	}

	if h.cfg.DryRun {
		h.notify(ctx, notifications.CertRotationResult{
			Environment:       envPrefix,
			Domain:            res.Domain,
			Success:           true,
			DryRun:            true,
			NotAfter:          res.NotAfter,
			DaysRemaining:     res.DaysRemaining,
			IntermediateCount: res.IntermediateCount,
		})
		return nil
	}

	if _, err = h.acm.ImportCertificate(ctx, &acm.ImportCertificateInput{
		CertificateArn:   aws.String(envCfg.AcmCertificateArn),
		Certificate:      certPEM,
		PrivateKey:       keyPEM,
		CertificateChain: chainPEM,
	}); err != nil {
		return h.notifyInfraFailure(ctx, envPrefix, envCfg.Domain, s3Location, fmt.Errorf("ACM ImportCertificate %s: %w", envCfg.AcmCertificateArn, err))
	}

	if err := h.putBackup(ctx, envCfg, res, certPEM, keyPEM, chainPEM); err != nil {
		return h.notifyInfraFailure(ctx, envPrefix, envCfg.Domain, s3Location, err)
	}

	// Archive in two phases so a mid-sequence failure cannot leave the source
	// prefix with a mixed-age bundle. Phase 1 copies all three files to
	// processed/; a failure here leaves the sources intact and the retry is
	// safe. Phase 2 deletes the sources; partial deletes are captured into a
	// joined error so the freshness check on the next invocation will reject
	// any leftover-pair scenario rather than rotate with stale chain.
	archiveBase := path.Join(h.cfg.ArchivePrefix, envPrefix, now.Format("20060102T150405Z"))
	bundleKeys := []string{wantCert, wantKey, wantChain}
	bundleBasenames := []string{certKeyName, keyKeyName, chainKeyName}
	for i, src := range bundleKeys {
		if err := h.copyObject(ctx, bucket, src, path.Join(archiveBase, bundleBasenames[i])); err != nil {
			return h.notifyInfraFailure(ctx, envPrefix, envCfg.Domain, s3Location, fmt.Errorf("archive copy %s: %w", src, err))
		}
	}
	if err := h.deleteSources(ctx, bucket, bundleKeys); err != nil {
		return h.notifyInfraFailure(ctx, envPrefix, envCfg.Domain, s3Location, err)
	}

	h.notify(ctx, notifications.CertRotationResult{
		Environment:       envPrefix,
		Domain:            res.Domain,
		Success:           true,
		NotAfter:          res.NotAfter,
		DaysRemaining:     res.DaysRemaining,
		IntermediateCount: res.IntermediateCount,
		AcmCertificateArn: envCfg.AcmCertificateArn,
	})
	return nil
}

// matchPrefix returns the first path segment of objectKey if it is a configured
// env prefix. Trailing slashes and leading slashes are tolerated.
func (h *handler) matchPrefix(objectKey string) (string, config.EnvConfig, bool) {
	objectKey = strings.TrimPrefix(objectKey, "/")
	parts := strings.SplitN(objectKey, "/", 2)
	if len(parts) < 2 {
		return "", config.EnvConfig{}, false
	}
	prefix := parts[0]
	cfg, ok := h.cfg.EnvPrefixesToCfg[prefix]
	return prefix, cfg, ok
}

// headIfExists returns the HeadObject result for key, or (nil, nil) if the
// object does not exist. Non-404 errors are propagated so the caller can
// surface them as infra failures.
func (h *handler) headIfExists(ctx context.Context, bucket, key string) (*s3.HeadObjectOutput, error) {
	out, err := h.s3.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err == nil {
		return out, nil
	}
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		// "NotFound" is the standard SDK error code for a missing object.
		// "Forbidden" is what S3 returns instead of NotFound when the caller
		// lacks s3:ListBucket; we deliberately omit ListBucket from the
		// Lambda role for least privilege, so a sibling invocation observing
		// a just-archived (deleted) bundle file gets a 403 here. Treat both
		// as "object does not exist" so the bundle-completeness check exits
		// quietly instead of returning a transient-failure error to Lambda
		// (which would otherwise fire async retries and spam Slack).
		switch apiErr.ErrorCode() {
		case "NotFound", "Forbidden":
			return nil, nil
		}
	}
	// Some S3 responses surface the same conditions as a generic
	// ResponseError without the typed APIError code; fall back to a string
	// match for both 404 and 403 so a missing-or-inaccessible HEAD does not
	// force a Lambda retry.
	if strings.Contains(err.Error(), "status code: 404") ||
		strings.Contains(err.Error(), "status code: 403") {
		return nil, nil
	}
	return nil, err
}

// verifyBundleFreshness returns a ValidationError when the LastModified
// timestamps of the three bundle files span more than window. Operator
// uploads are normally seconds apart; a multi-hour spread means at least one
// of the three files is left over from a prior rotation.
func verifyBundleFreshness(certHead, keyHead, chainHead *s3.HeadObjectOutput, window time.Duration) error {
	heads := map[string]*s3.HeadObjectOutput{
		certKeyName:  certHead,
		keyKeyName:   keyHead,
		chainKeyName: chainHead,
	}
	var minTime, maxTime time.Time
	first := true
	for name, head := range heads {
		if head == nil || head.LastModified == nil {
			return certvalidator.ValidationError{
				Msg:            fmt.Sprintf("%s is missing LastModified metadata", name),
				ActionRequired: fmt.Sprintf("Re-upload %s alongside the other bundle files.", name),
			}
		}
		t := head.LastModified.UTC()
		if first {
			minTime = t
			maxTime = t
			first = false
			continue
		}
		if t.Before(minTime) {
			minTime = t
		}
		if t.After(maxTime) {
			maxTime = t
		}
	}
	if spread := maxTime.Sub(minTime); spread > window {
		return certvalidator.ValidationError{
			Msg: fmt.Sprintf(
				"bundle files span %s across LastModified timestamps; %s/%s/%s must be uploaded within %s of each other",
				spread.Round(time.Second), certKeyName, keyKeyName, chainKeyName, window,
			),
			ActionRequired: fmt.Sprintf(
				"Delete any lingering %s / %s / %s at the prefix and re-upload all three files together.",
				certKeyName, keyKeyName, chainKeyName,
			),
		}
	}
	return nil
}

func (h *handler) getObjectBytes(ctx context.Context, bucket, key string) ([]byte, error) {
	out, err := h.s3.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, err
	}
	defer out.Body.Close()
	return io.ReadAll(out.Body)
}

func (h *handler) copyObject(ctx context.Context, bucket, srcKey, dstKey string) error {
	_, err := h.s3.CopyObject(ctx, &s3.CopyObjectInput{
		Bucket:     aws.String(bucket),
		CopySource: aws.String(pathEscape(bucket + "/" + srcKey)),
		Key:        aws.String(dstKey),
	})
	return err
}

// deleteSources deletes every key in srcKeys, attempting every delete even if
// earlier ones fail. The returned error aggregates any per-key failures so the
// caller can surface a single infra failure message.
func (h *handler) deleteSources(ctx context.Context, bucket string, srcKeys []string) error {
	var deleteErrs []error
	for _, src := range srcKeys {
		if _, err := h.s3.DeleteObject(ctx, &s3.DeleteObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(src),
		}); err != nil {
			deleteErrs = append(deleteErrs, fmt.Errorf("delete %s: %w", src, err))
		}
	}
	if len(deleteErrs) > 0 {
		return fmt.Errorf("archive delete: %w", errors.Join(deleteErrs...))
	}
	return nil
}

// putBackup writes the validated bundle to the backup secret using a
// deterministic ClientRequestToken so retries with the same payload do not
// create duplicate secret versions. A SDK call rather than
// backend/internal/secrets.NewSecret is used here because the backup secret
// has no AWSCURRENT value on first use, which would cause NewSecret's initial
// GetSecretValue to fail (see awsclients package doc).
func (h *handler) putBackup(ctx context.Context, envCfg config.EnvConfig, res certvalidator.Result, certPEM, keyPEM, chainPEM []byte) error {
	payload, err := json.Marshal(map[string]string{
		"cert_pem":  string(certPEM),
		"key_pem":   string(keyPEM),
		"chain_pem": string(chainPEM),
		"domain":    envCfg.Domain,
		"not_after": res.NotAfter.UTC().Format(time.RFC3339),
	})
	if err != nil {
		return fmt.Errorf("marshal backup payload: %w", err)
	}
	_, err = h.secrets.PutSecretValue(ctx, &secretsmanager.PutSecretValueInput{
		SecretId:           aws.String(envCfg.BackupSecretArn),
		SecretString:       aws.String(string(payload)),
		ClientRequestToken: aws.String(payloadRequestToken(payload)),
	})
	if err != nil {
		return fmt.Errorf("put backup secret %s: %w", envCfg.BackupSecretArn, err)
	}
	return nil
}

// payloadRequestToken returns a deterministic 32-character hex token derived
// from the payload. Identical payloads produce identical tokens, so Secrets
// Manager PutSecretValue retries with the same payload do not create
// duplicate versions. Mirrors the helper in backend/internal/secrets.
func payloadRequestToken(payload []byte) string {
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])[:32]
}

// pathEscape URL-encodes an S3 CopySource value while preserving forward
// slashes. CopySource must be URL-encoded per the S3 API.
func pathEscape(s string) string {
	u := url.URL{Path: s}
	return strings.TrimPrefix(u.EscapedPath(), "/")
}

// notify sends a Slack notification and logs any send error. Slack failures
// are never allowed to block rotation, but silent discards leave operators
// with no audit trail if the webhook was unreachable; the log line keeps a
// CloudWatch breadcrumb for every such drop.
func (h *handler) notify(ctx context.Context, r notifications.CertRotationResult) {
	if err := h.notifier.SendCertRotationNotification(ctx, r); err != nil {
		log.Printf("slack notification failed (env=%s domain=%s success=%t dry_run=%t validation_failed=%t): %v",
			r.Environment, r.Domain, r.Success, r.DryRun, r.ValidationFailed, err)
	}
}

// notifyValidationFailure sends a Slack notification for operator-correctable
// input errors (bad PEM, wrong domain, expired cert, stale bundle). Always
// returns nil so Lambda does not retry the invocation.
func (h *handler) notifyValidationFailure(ctx context.Context, envPrefix, domain, s3Location string, verr error) error {
	result := notifications.CertRotationResult{
		Environment:      envPrefix,
		Domain:           domain,
		ValidationFailed: true,
		ErrorMessage:     verr.Error(),
		S3Location:       s3Location,
	}
	if ve, ok := verr.(certvalidator.ValidationError); ok {
		result.ErrorMessage = ve.Msg
		result.ActionRequired = ve.ActionRequired
	}
	h.notify(ctx, result)
	return nil
}

// notifyInfraFailure sends a Slack notification for infrastructure errors
// (S3, ACM, Secrets Manager) and returns the original error so Lambda retries.
func (h *handler) notifyInfraFailure(ctx context.Context, envPrefix, domain, s3Location string, infraErr error) error {
	h.notify(ctx, notifications.CertRotationResult{
		Environment:  envPrefix,
		Domain:       domain,
		ErrorMessage: infraErr.Error(),
		S3Location:   s3Location,
	})
	return infraErr
}
