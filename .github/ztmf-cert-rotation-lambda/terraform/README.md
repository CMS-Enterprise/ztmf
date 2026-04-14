# Terraform (dev)

This folder deploys the **dev** cert-rotation Lambda and wires it to the S3 prefix trigger.

## Prereqs

- An existing S3 bucket: `ztmf-cert-rotation-dev` (or your chosen name)
- Existing ACM certificate ARN for `dev.ztmf.cms.gov` (fixed ARN to re-import)
- Existing Secrets Manager secret ARN for backup storage (the Lambda writes latest JSON bundle)
- Existing Secrets Manager secret ARN that contains the Slack webhook URL as the SecretString
- A pre-built Lambda zip containing `bootstrap` at the root

## Build zip (example)

From repo root:

```bash
# Requires Go toolchain.
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o bootstrap ./cmd/ztmf-cert-rotation
zip -j lambda.zip bootstrap
```

## Apply

```bash
cd terraform
terraform init

terraform apply \
  -var="cert_bucket_name=ztmf-cert-rotation-dev" \
  -var="lambda_zip_path=../lambda.zip" \
  -var="acm_certificate_arn=arn:aws:acm:us-east-1:123456789012:certificate/..." \
  -var="backup_secret_arn=arn:aws:secretsmanager:us-east-1:123456789012:secret:ztmf-cert-rotation-dev-backup-..." \
  -var="slack_webhook_secret_arn=arn:aws:secretsmanager:us-east-1:123456789012:secret:ztmf-cert-rotation-dev-slack-webhook-..."
```

## Upload files

```bash
aws s3 cp cert.pem  s3://ztmf-cert-rotation-dev/dev/cert.pem
aws s3 cp key.pem   s3://ztmf-cert-rotation-dev/dev/key.pem
aws s3 cp chain.pem s3://ztmf-cert-rotation-dev/dev/chain.pem
```

