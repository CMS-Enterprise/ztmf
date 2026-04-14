package main

import (
	"context"
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

	"github.com/cms/ztmf-cert-rotation-lambda/internal/awsclients"
	"github.com/cms/ztmf-cert-rotation-lambda/internal/certvalidator"
	"github.com/cms/ztmf-cert-rotation-lambda/internal/config"
	"github.com/cms/ztmf-cert-rotation-lambda/internal/slack"
)

const (
	certKeyName  = "cert.pem"
	keyKeyName   = "key.pem"
	chainKeyName = "chain.pem"
)

type handler struct {
	cfg     config.Config
	clients awsclients.Clients
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

	lambda.Start((&handler{cfg: cfg, clients: clients}).Handle)
}

func (h *handler) Handle(ctx context.Context, evt events.S3Event) error {
	// Multiple records are possible; process independently.
	for _, r := range evt.Records {
		if err := h.handleRecord(ctx, r); err != nil {
			// Returning error will retry; we only want retries for transient AWS issues.
			// Slack notification is sent inside handleRecord on failures.
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
		// Ignore other buckets.
		return nil
	}
	if !strings.HasSuffix(strings.ToLower(key), ".pem") {
		return nil
	}

	envPrefix, envCfg, ok := h.matchPrefix(key)
	if !ok {
		// Not one of the configured env prefixes.
		return nil
	}

	webhookURL, err := h.resolveSlackWebhook(ctx, envCfg)
	if err != nil {
		// Can't notify Slack if Slack isn't resolvable; treat as transient (retry) because
		// Secrets Manager/permissions/temporary errors are possible.
		return err
	}
	sl := slack.Client{WebhookURL: webhookURL}

	// Only react to exact filenames under prefix: <env>/cert.pem etc.
	base := path.Base(key)
	if base != certKeyName && base != keyKeyName && base != chainKeyName {
		return nil
	}

	// Check if all three exist; if not, exit cleanly.
	wantCert := path.Join(envPrefix, certKeyName)
	wantKey := path.Join(envPrefix, keyKeyName)
	wantChain := path.Join(envPrefix, chainKeyName)

	existsCert, err := h.objectExists(ctx, bucket, wantCert)
	if err != nil {
		return notifyAndReturn(ctx, sl, fmt.Sprintf("TLS CERT ROTATION FAILED (%s)\nError: unable to check S3 object existence: %v", strings.ToUpper(envPrefix), err), err)
	}
	existsKey, err := h.objectExists(ctx, bucket, wantKey)
	if err != nil {
		return notifyAndReturn(ctx, sl, fmt.Sprintf("TLS CERT ROTATION FAILED (%s)\nError: unable to check S3 object existence: %v", strings.ToUpper(envPrefix), err), err)
	}
	existsChain, err := h.objectExists(ctx, bucket, wantChain)
	if err != nil {
		return notifyAndReturn(ctx, sl, fmt.Sprintf("TLS CERT ROTATION FAILED (%s)\nError: unable to check S3 object existence: %v", strings.ToUpper(envPrefix), err), err)
	}
	if !existsCert || !existsKey || !existsChain {
		// Quiet exit until all parts present.
		return nil
	}

	certPEM, err := h.getObjectBytes(ctx, bucket, wantCert)
	if err != nil {
		return notifyAndReturn(ctx, sl, fmt.Sprintf("TLS CERT ROTATION FAILED (%s)\nError: unable to read cert.pem: %v", strings.ToUpper(envPrefix), err), err)
	}
	keyPEM, err := h.getObjectBytes(ctx, bucket, wantKey)
	if err != nil {
		return notifyAndReturn(ctx, sl, fmt.Sprintf("TLS CERT ROTATION FAILED (%s)\nError: unable to read key.pem: %v", strings.ToUpper(envPrefix), err), err)
	}
	chainPEM, err := h.getObjectBytes(ctx, bucket, wantChain)
	if err != nil {
		return notifyAndReturn(ctx, sl, fmt.Sprintf("TLS CERT ROTATION FAILED (%s)\nError: unable to read chain.pem: %v", strings.ToUpper(envPrefix), err), err)
	}

	now := time.Now().UTC()
	res, err := certvalidator.Validate(certvalidator.Bundle{
		CertPEM:  certPEM,
		KeyPEM:   keyPEM,
		ChainPEM: chainPEM,
	}, envCfg.Domain, now)
	if err != nil {
		msg := fmt.Sprintf(
			"TLS CERT ROTATION FAILED (%s)\nDomain: %s\nError: %v\nAction Required: Upload valid files to s3://%s/%s/{cert.pem,key.pem,chain.pem}",
			strings.ToUpper(envPrefix),
			envCfg.Domain,
			err,
			bucket,
			envPrefix,
		)
		if ve, ok := err.(certvalidator.ValidationError); ok && strings.TrimSpace(ve.ActionRequired) != "" {
			msg = fmt.Sprintf(
				"TLS CERT ROTATION FAILED (%s)\nDomain: %s\nError: %s\nAction Required: %s\nLocation: s3://%s/%s/",
				strings.ToUpper(envPrefix),
				envCfg.Domain,
				ve.Msg,
				ve.ActionRequired,
				bucket,
				envPrefix,
			)
		}
		_ = sl.PostText(ctx, msg)
		// Validation errors should not retry.
		return nil
	}

	if h.cfg.DryRun {
		_ = sl.PostText(ctx, fmt.Sprintf(
			"TLS CERT ROTATION SUCCESS (%s) [DRY RUN]\nDomain: %s\nExpires: %s (%d days remaining)\nChain: Server cert + %d intermediate CA",
			strings.ToUpper(envPrefix),
			res.Domain,
			res.NotAfter.UTC().Format("2006-01-02"),
			res.DaysRemaining,
			res.IntermediateCount,
		))
		return nil
	}

	// 1) Import to ACM by re-importing over known ARN.
	_, err = h.clients.ACM.ImportCertificate(ctx, &acm.ImportCertificateInput{
		CertificateArn:  aws.String(envCfg.AcmCertificateArn),
		Certificate:     certPEM,
		PrivateKey:      keyPEM,
		CertificateChain: chainPEM,
	})
	if err != nil {
		msg := fmt.Sprintf(
			"TLS CERT ROTATION FAILED (%s)\nDomain: %s\nError: ACM import failed: %v\nAction Required: Investigate ACM permissions and certificate ARN.",
			strings.ToUpper(envPrefix),
			envCfg.Domain,
			err,
		)
		return notifyAndReturn(ctx, sl, msg, err)
	}

	// 2) Backup to Secrets Manager.
	backupPayload, _ := json.Marshal(map[string]string{
		"cert_pem":  string(certPEM),
		"key_pem":   string(keyPEM),
		"chain_pem": string(chainPEM),
		"domain":    envCfg.Domain,
		"not_after": res.NotAfter.UTC().Format(time.RFC3339),
	})

	_, err = h.clients.Secrets.PutSecretValue(ctx, &secretsmanager.PutSecretValueInput{
		SecretId:     aws.String(envCfg.BackupSecretArn),
		SecretString: aws.String(string(backupPayload)),
	})
	if err != nil {
		msg := fmt.Sprintf(
			"TLS CERT ROTATION FAILED (%s)\nDomain: %s\nError: Secrets Manager backup failed: %v\nAction Required: Investigate Secrets Manager permissions/ARN.",
			strings.ToUpper(envPrefix),
			envCfg.Domain,
			err,
		)
		return notifyAndReturn(ctx, sl, msg, err)
	}

	// 3) Archive files to processed/<env>/<timestamp>/... then delete originals.
	ts := now.Format("20060102T150405Z")
	archiveBase := path.Join(h.cfg.ArchivePrefix, envPrefix, ts)

	if err := h.archiveOne(ctx, bucket, wantCert, path.Join(archiveBase, certKeyName)); err != nil {
		return notifyAndReturn(ctx, sl, fmt.Sprintf("TLS CERT ROTATION FAILED (%s)\nDomain: %s\nError: archiving cert.pem failed: %v", strings.ToUpper(envPrefix), envCfg.Domain, err), err)
	}
	if err := h.archiveOne(ctx, bucket, wantKey, path.Join(archiveBase, keyKeyName)); err != nil {
		return notifyAndReturn(ctx, sl, fmt.Sprintf("TLS CERT ROTATION FAILED (%s)\nDomain: %s\nError: archiving key.pem failed: %v", strings.ToUpper(envPrefix), envCfg.Domain, err), err)
	}
	if err := h.archiveOne(ctx, bucket, wantChain, path.Join(archiveBase, chainKeyName)); err != nil {
		return notifyAndReturn(ctx, sl, fmt.Sprintf("TLS CERT ROTATION FAILED (%s)\nDomain: %s\nError: archiving chain.pem failed: %v", strings.ToUpper(envPrefix), envCfg.Domain, err), err)
	}

	_ = sl.PostText(ctx, fmt.Sprintf(
		"TLS CERT ROTATION SUCCESS (%s)\nDomain: %s\nExpires: %s (%d days remaining)\nChain: Server cert + %d intermediate CA\nACM ARN: %s",
		strings.ToUpper(envPrefix),
		res.Domain,
		res.NotAfter.UTC().Format("2006-01-02"),
		res.DaysRemaining,
		res.IntermediateCount,
		envCfg.AcmCertificateArn,
	))

	return nil
}

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

func (h *handler) resolveSlackWebhook(ctx context.Context, envCfg config.EnvConfig) (string, error) {
	if strings.TrimSpace(envCfg.SlackWebhookURL) != "" {
		return strings.TrimSpace(envCfg.SlackWebhookURL), nil
	}
	arn := strings.TrimSpace(envCfg.SlackWebhookSecretArn)
	if arn == "" {
		return "", errors.New("slack webhook configuration missing")
	}
	out, err := h.clients.Secrets.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(arn),
	})
	if err != nil {
		return "", fmt.Errorf("get slack webhook secret value: %w", err)
	}
	if out.SecretString == nil || strings.TrimSpace(*out.SecretString) == "" {
		return "", errors.New("slack webhook secret is empty")
	}
	return strings.TrimSpace(*out.SecretString), nil
}

func (h *handler) objectExists(ctx context.Context, bucket, key string) (bool, error) {
	_, err := h.clients.S3.HeadObject(ctx, &s3.HeadObjectInput{
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
	// Some S3 errors present as ResponseError with 404; best-effort treat those as non-existent.
	if strings.Contains(err.Error(), "status code: 404") {
		return false, nil
	}
	return false, err
}

func (h *handler) getObjectBytes(ctx context.Context, bucket, key string) ([]byte, error) {
	out, err := h.clients.S3.GetObject(ctx, &s3.GetObjectInput{
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
	_, err := h.clients.S3.CopyObject(ctx, &s3.CopyObjectInput{
		Bucket:     aws.String(bucket),
		CopySource: aws.String(pathEscape(bucket + "/" + srcKey)),
		Key:        aws.String(dstKey),
	})
	if err != nil {
		return err
	}
	_, err = h.clients.S3.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(srcKey),
	})
	return err
}

func pathEscape(s string) string {
	// S3 CopySource uses URL-encoding but keeps '/'.
	u := url.URL{Path: s}
	return strings.TrimPrefix(u.EscapedPath(), "/")
}

func notifyAndReturn(ctx context.Context, sl slack.Client, message string, err error) error {
	_ = sl.PostText(ctx, message)
	return err
}

