# lambda-cert-rotation

S3-triggered Lambda that validates a TLS certificate bundle uploaded to S3 and (when not in dry-run mode) imports it into ACM, stores a backup in Secrets Manager, and archives the uploaded files.

## Expected S3 layout

The Lambda looks for these exact objects under each configured prefix:

- `cert.pem`
- `key.pem`
- `chain.pem`

Uploads are expected at:

- `s3://$CERT_BUCKET/<prefix>/{cert.pem,key.pem,chain.pem}`

## Event behavior (S3 notifications)

S3 can emit one event per uploaded object, so uploading all 3 files quickly may trigger **3 separate Lambda runs**. This is expected.

The ACM `ImportCertificate` call is idempotent for a fixed certificate ARN, but you may see duplicate Slack success messages if all 3 events observe “all files present” around the same time.

