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
	"github.com/aws/smithy-go"

	"github.com/CMS-Enterprise/ztmf/backend/cmd/lambda-cert-rotation/internal/awsclients"
	"github.com/CMS-Enterprise/ztmf/backend/cmd/lambda-cert-rotation/internal/certvalidator"
	"github.com/CMS-Enterprise/ztmf/backend/cmd/lambda-cert-rotation/internal/config"
	"github.com/CMS-Enterprise/ztmf/backend/internal/notifications"
	"github.com/CMS-Enterprise/ztmf/backend/internal/secrets"
)

const (
	certKeyName  = "cert.pem"
	keyKeyName   = "key.pem"
	chainKeyName = "chain.pem"
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

type handler struct {
	cfg      config.Config
	s3       s3API
	acm      acmAPI
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

	lambda.Start((&handler{cfg: cfg, s3: clients.S3, acm: clients.ACM, notifier: n}).Handle)
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

	s3Location := fmt.Sprintf("s3://%s/%s/", bucket, envPrefix)

	wantCert := path.Join(envPrefix, certKeyName)
	wantKey := path.Join(envPrefix, keyKeyName)
	wantChain := path.Join(envPrefix, chainKeyName)

	existsCert, err := h.objectExists(ctx, bucket, wantCert)
	if err != nil {
		return h.notifyInfraFailure(ctx, envPrefix, envCfg.Domain, s3Location, fmt.Errorf("check cert.pem: %w", err))
	}
	existsKey, err := h.objectExists(ctx, bucket, wantKey)
	if err != nil {
		return h.notifyInfraFailure(ctx, envPrefix, envCfg.Domain, s3Location, fmt.Errorf("check key.pem: %w", err))
	}
	existsChain, err := h.objectExists(ctx, bucket, wantChain)
	if err != nil {
		return h.notifyInfraFailure(ctx, envPrefix, envCfg.Domain, s3Location, fmt.Errorf("check chain.pem: %w", err))
	}
	if !existsCert || !existsKey || !existsChain {
		// Quiet exit until the full bundle is present.
		return nil
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
		_ = h.notifier.SendCertRotationNotification(ctx, notifications.CertRotationResult{
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
		return h.notifyInfraFailure(ctx, envPrefix, envCfg.Domain, s3Location, fmt.Errorf("ACM import: %w", err))
	}

	backupPayload := map[string]string{
		"cert_pem":  string(certPEM),
		"key_pem":   string(keyPEM),
		"chain_pem": string(chainPEM),
		"domain":    envCfg.Domain,
		"not_after": res.NotAfter.UTC().Format(time.RFC3339),
	}
	backupSecret, err := secrets.NewSecret(envCfg.BackupSecretArn)
	if err != nil {
		return h.notifyInfraFailure(ctx, envPrefix, envCfg.Domain, s3Location, fmt.Errorf("open backup secret: %w", err))
	}
	if err := backupSecret.Put(ctx, backupPayload); err != nil {
		return h.notifyInfraFailure(ctx, envPrefix, envCfg.Domain, s3Location, fmt.Errorf("backup put: %w", err))
	}

	archiveBase := path.Join(h.cfg.ArchivePrefix, envPrefix, now.Format("20060102T150405Z"))
	if err := h.archiveOne(ctx, bucket, wantCert, path.Join(archiveBase, certKeyName)); err != nil {
		return h.notifyInfraFailure(ctx, envPrefix, envCfg.Domain, s3Location, fmt.Errorf("archive cert.pem: %w", err))
	}
	if err := h.archiveOne(ctx, bucket, wantKey, path.Join(archiveBase, keyKeyName)); err != nil {
		return h.notifyInfraFailure(ctx, envPrefix, envCfg.Domain, s3Location, fmt.Errorf("archive key.pem: %w", err))
	}
	if err := h.archiveOne(ctx, bucket, wantChain, path.Join(archiveBase, chainKeyName)); err != nil {
		return h.notifyInfraFailure(ctx, envPrefix, envCfg.Domain, s3Location, fmt.Errorf("archive chain.pem: %w", err))
	}

	_ = h.notifier.SendCertRotationNotification(ctx, notifications.CertRotationResult{
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

func (h *handler) objectExists(ctx context.Context, bucket, key string) (bool, error) {
	_, err := h.s3.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err == nil {
		return true, nil
	}
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		if apiErr.ErrorCode() == "NotFound" {
			return false, nil
		}
	}
	// Some S3 responses surface 404 as a generic ResponseError; fall back to
	// a string match so a NotFound HEAD does not force a Lambda retry.
	if strings.Contains(err.Error(), "status code: 404") {
		return false, nil
	}
	return false, err
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

func (h *handler) archiveOne(ctx context.Context, bucket, srcKey, dstKey string) error {
	if _, err := h.s3.CopyObject(ctx, &s3.CopyObjectInput{
		Bucket:     aws.String(bucket),
		CopySource: aws.String(pathEscape(bucket + "/" + srcKey)),
		Key:        aws.String(dstKey),
	}); err != nil {
		return err
	}
	_, err := h.s3.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(srcKey),
	})
	return err
}

// pathEscape URL-encodes an S3 CopySource value while preserving forward
// slashes. CopySource must be URL-encoded per the S3 API.
func pathEscape(s string) string {
	u := url.URL{Path: s}
	return strings.TrimPrefix(u.EscapedPath(), "/")
}

// notifyValidationFailure sends a Slack notification for operator-correctable
// input errors (bad PEM, wrong domain, expired cert). Always returns nil so
// Lambda does not retry the invocation.
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
	_ = h.notifier.SendCertRotationNotification(ctx, result)
	return nil
}

// notifyInfraFailure sends a Slack notification for infrastructure errors
// (S3, ACM, Secrets Manager) and returns the original error so Lambda retries.
func (h *handler) notifyInfraFailure(ctx context.Context, envPrefix, domain, s3Location string, infraErr error) error {
	_ = h.notifier.SendCertRotationNotification(ctx, notifications.CertRotationResult{
		Environment:  envPrefix,
		Domain:       domain,
		ErrorMessage: infraErr.Error(),
		S3Location:   s3Location,
	})
	return infraErr
}
