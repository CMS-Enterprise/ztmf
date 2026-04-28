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

The ACM `ImportCertificate` call is idempotent for a fixed certificate ARN, but you may see duplicate Slack success messages if all 3 events observe "all files present" around the same time.

## Bundle freshness window

Before importing, the Lambda reads `LastModified` on all three source files via `HeadObject`. If the oldest and newest timestamps span **more than one hour**, the bundle is rejected as a validation failure and a Slack alert is sent.

This rule exists to defend against a specific failure mode: a prior rotation that archived only some of the three files before erroring out leaves `key.pem` and `chain.pem` at the source. A later operator upload of just `cert.pem` would otherwise pair the fresh cert with stale intermediate and private key, and if the operator reused the key pair the inconsistent bundle would silently import into ACM.

**Operator contract:**

- Always upload all three files within the same `aws s3 cp` session. Sequential uploads are fine; the window is wide enough for normal retry scripts.
- If you need to replace a single file (typo, wrong version), delete and re-upload **all three** rather than only the one you want to change.
- If the Slack alert says the bundle "spans X across LastModified timestamps", delete every `*.pem` at the prefix and re-upload from scratch:

  ```bash
  aws s3 rm "s3://ztmf-cert-rotation-<env>/<env>/" --recursive --exclude "*" --include "*.pem"
  aws s3 cp cert.pem  s3://ztmf-cert-rotation-<env>/<env>/cert.pem
  aws s3 cp key.pem   s3://ztmf-cert-rotation-<env>/<env>/key.pem
  aws s3 cp chain.pem s3://ztmf-cert-rotation-<env>/<env>/chain.pem
  ```

