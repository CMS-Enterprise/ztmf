# Cert rotation automation plan

## Goal

Automate validation + import of DigiCert TLS certificate bundles received by the team, to prevent:

- incomplete intermediate CA chain imports (CloudFront 502)
- wrong cert selection when multiple ACM certs exist for same domain

## Scope (phase 1)

**Target**: `dev.ztmf.cms.gov` (S3 prefix `dev/` in dev account).

Phase 2 extends the same Lambda to `impl/` in the dev account and a separate deployment in the prod account for `prod/`.

## Workflow

1. Team member uploads 3 files to S3:
   - `s3://ztmf-cert-rotation-dev/dev/cert.pem`
   - `s3://ztmf-cert-rotation-dev/dev/key.pem`
   - `s3://ztmf-cert-rotation-dev/dev/chain.pem`
2. S3 `ObjectCreated:*` triggers Lambda for `*.pem`.
3. Lambda checks presence of all 3 objects under the prefix.
   - If any are missing: **exit cleanly** (no Slack noise).
4. Lambda validates:
   - PEM is parseable for each file
   - `chain.pem` includes at least 1 intermediate CA
   - private key matches server cert public key
   - server cert hostname matches expected environment domain
   - server cert not expired
   - server cert verifies against the provided intermediate pool (server → intermediate)
5. Lambda imports into ACM using a **fixed ACM certificate ARN** (re-import).
6. Lambda backs up bundle to Secrets Manager.
7. Lambda archives the three objects to `processed/<env>/<timestamp>/`.
8. Lambda posts Slack success/failure.

## Architecture (text diagram)

S3 bucket `ztmf-cert-rotation-<env>`

- `<prefix>/cert.pem`
- `<prefix>/key.pem`
- `<prefix>/chain.pem`
    |
    | (ObjectCreated:* on `*.pem`)
    v
Lambda `ztmf-cert-rotation-<env>`
    |
    +--> validate + import (ACM)
    |
    +--> backup (Secrets Manager)
    |
    +--> archive (S3 `processed/...`)
    |
    +--> notify (Slack webhook)

## Configuration

Lambda environment variables:

- `CERT_BUCKET`: e.g. `ztmf-cert-rotation-dev`
- `ENV_PREFIXES_JSON`: JSON map keyed by prefix (e.g. `dev`) with:
  - `domain`
  - `acmCertificateArn`
  - `backupSecretArn`
  - `slackWebhookUrl`
- `ARCHIVE_PREFIX`: default `processed`
- `DRY_RUN`: `true` to validate/notify only (no import/backup/archive)

## Slack message formats

Success:

- `TLS CERT ROTATION SUCCESS (DEV)`
- Domain, expiry date + days remaining
- Chain summary (server + N intermediates)
- ACM ARN

Failure:

- `TLS CERT ROTATION FAILED (DEV)`
- Domain
- Error text
- Action required
- S3 location to upload corrected files

## Test plan

Unit tests for validator:

- valid bundle passes
- missing/empty chain fails with action message
- expired cert fails
- wrong domain fails
- key mismatch fails
- invalid PEM fails

Integration testing (IMPL recommended):

- deploy with `DRY_RUN=true`, verify validation + Slack messages
- negative tests by uploading intentionally wrong bundles
- then enable import/backup/archive, run with known-good cert, verify:
  - ACM `ImportCertificate` succeeded for fixed ARN
  - Secrets Manager has latest bundle JSON
  - S3 archived files created and originals removed

