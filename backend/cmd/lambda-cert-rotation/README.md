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

S3 emits one `ObjectCreated` event per uploaded object, so uploading all 3 files triggers 3 Lambda invocations. **Only the invocation triggered by `chain.pem` performs the rotation work.** The `cert.pem` and `key.pem` invocations exit silently. This guarantees:

- Exactly one Slack notification per real rotation, regardless of upload order.
- No concurrent races on validation, ACM import, Secrets Manager backup, or archival.
- The `chain.pem` upload is the canonical commit point of a new bundle.

**Required upload order:** upload `chain.pem` last. If `chain.pem` is uploaded before the other two files, the bundle-completeness check exits cleanly (other two missing) and no rotation runs. Re-upload `chain.pem` after the other two files are in place to fire the rotation.

The ACM `ImportCertificate` call remains idempotent for a fixed certificate ARN, so re-running the rotation by re-uploading `chain.pem` against an already-current bundle is safe.

## Bundle freshness window

Before importing, the Lambda reads `LastModified` on all three source files via `HeadObject`. If the oldest and newest timestamps span **more than one hour**, the bundle is rejected as a validation failure and a Slack alert is sent.

This rule exists to defend against a specific failure mode: a prior rotation that archived only some of the three files before erroring out leaves `key.pem` and `chain.pem` at the source. A later operator upload of just `cert.pem` would otherwise pair the fresh cert with stale intermediate and private key, and if the operator reused the key pair the inconsistent bundle would silently import into ACM.

**Operator contract:**

- Always upload all three files within the same `aws s3 cp` session. Sequential uploads are fine; the window is wide enough for normal retry scripts.
- If you need to replace a single file (typo, wrong version), delete and re-upload **all three** rather than only the one you want to change.
- If the Slack alert says the bundle "spans X across LastModified timestamps", delete every `*.pem` at the prefix and re-upload from scratch. Upload `chain.pem` last, since it is the trigger for the rotation:

  ```bash
  aws s3 rm "s3://ztmf-cert-rotation-<env>/<env>/" --recursive --exclude "*" --include "*.pem"
  aws s3 cp cert.pem  s3://ztmf-cert-rotation-<env>/<env>/cert.pem
  aws s3 cp key.pem   s3://ztmf-cert-rotation-<env>/<env>/key.pem
  aws s3 cp chain.pem s3://ztmf-cert-rotation-<env>/<env>/chain.pem
  ```

