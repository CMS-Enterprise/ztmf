# ztmf-cert-rotation-lambda

S3-triggered Lambda that validates TLS certificate files and (if valid) re-imports a known ACM certificate ARN, backs up the bundle to Secrets Manager, archives the input files, and posts Slack notifications.

## Dev workflow (dev.ztmf.cms.gov)

Upload the 3 PEM files:

```bash
aws s3 cp cert.pem  s3://ztmf-cert-rotation-dev/dev/cert.pem
aws s3 cp key.pem   s3://ztmf-cert-rotation-dev/dev/key.pem
aws s3 cp chain.pem s3://ztmf-cert-rotation-dev/dev/chain.pem
```

The Lambda exits cleanly until all 3 objects exist for the environment prefix.

## Configuration (environment variables) (placeholder until we get webhook)

- `CERT_BUCKET`: S3 bucket name (e.g. `ztmf-cert-rotation-dev`)
- `ENV_PREFIXES_JSON`: JSON map of prefix -> config. Example:
  - `{"dev":{"domain":"dev.ztmf.cms.gov","acmCertificateArn":"arn:aws:acm:...","backupSecretArn":"arn:aws:secretsmanager:...","slackWebhookUrl":"https://hooks.slack.com/services/..."}}`
- `ARCHIVE_PREFIX`: base prefix for processed archives (default `processed`)
- `DRY_RUN`: `true` to validate only (no ACM import/Secrets backup/archive)

## Build

```bash
cd ztmf-cert-rotation-lambda
go test ./...
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o bootstrap ./cmd/ztmf-cert-rotation
```

## Notes

- This repo intentionally uses **ACM re-import to a fixed certificate ARN** to avoid the “two certs for the same domain” selection hazard.
- Rotate any AWS credentials that were ever pasted into chats/logs.

