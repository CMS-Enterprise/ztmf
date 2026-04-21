# ztmf-kion-key-rotate

Scheduled AWS Lambda that rotates the Kion App API key stored in AWS Secrets Manager. Runs daily at 06:00 UTC per ZTMF account; short-circuits if the stored key was rotated in the last `ROTATE_AFTER_DAYS` days (default 4). Kion App API keys expire 7 days after issue, so a 4-day window leaves a 3-day recovery margin.

## Architecture

```
EventBridge (cron: 0 6 * * ? *)
        |
        v
  ztmf-kion-key-rotate-<env>  ---> CloudWatch Logs + Metrics (ZTMF/Kion/DaysSinceRotation)
        |                     ---> DLQ: ztmf-kion-key-rotate-dlq-<env>
        |                     ---> Slack (via ztmf_slack_webhook)
        v
  Secrets Manager: ztmf_kion_<env>
        |
        v
  Kion tenant (POST /api/v3/app-api-key/rotate)
```

## Secret payload

Secret name per environment: `ztmf_kion_dev` or `ztmf_kion_prod`.

JSON body:

```json
{
  "api_key":    "<kion_app_api_key>",
  "base_url":   "https://<kion-host>",
  "rotated_at": "2026-04-21T06:00:00Z"
}
```

The base URL lives in the secret payload (not in Lambda environment or Terraform) so the tenant is never committed to git. The Lambda reads `base_url` from the secret at every invocation.

## Environment variables (set by Terraform)

| Variable | Purpose |
|----------|---------|
| `ENVIRONMENT` | `dev` or `prod`; selects the secret and tags the CloudWatch metric. |
| `KION_SECRET_ID` | Secret name to read and write, e.g. `ztmf_kion_dev`. |
| `SLACK_SECRET_ID` | Name of the shared `ztmf_slack_webhook` secret. |
| `ROTATE_AFTER_DAYS` | Idempotency threshold. Default 4. |

## Event payload

```json
{
  "trigger_type": "scheduled",
  "dry_run":      false,
  "force":        false
}
```

- `trigger_type`: `scheduled` (EventBridge) or `manual` (operator-invoked).
- `dry_run`: Lambda loads the secret, checks idempotency, logs what it would do, posts a dry-run Slack message, and returns without calling Kion or writing the secret. EventBridge sets this to `true` for dev and `false` for prod.
- `force`: bypass the `ROTATE_AFTER_DAYS` check. Use for recovery invocations only.

## Initial bootstrap

Terraform creates the secret resource but does not set its value. After the first apply per environment:

1. Sign into Kion and generate a new App API Key for the ZTMF service account in that environment (two distinct keys, one for dev and one for prod, against the same prod Kion tenant).
2. Put the key into Secrets Manager:

   ```
   aws secretsmanager put-secret-value \
     --secret-id ztmf_kion_dev \
     --secret-string '{"api_key":"<KEY>","base_url":"https://<kion-host>","rotated_at":"2026-04-21T00:00:00Z"}'
   ```

3. Smoke test with dry-run:

   ```
   aws lambda invoke \
     --function-name ztmf-kion-key-rotate-dev \
     --payload '{"trigger_type":"manual","dry_run":true,"force":true}' \
     --cli-binary-format raw-in-base64-out \
     /tmp/out.json && cat /tmp/out.json
   ```

4. Real rotation test:

   ```
   aws lambda invoke \
     --function-name ztmf-kion-key-rotate-dev \
     --payload '{"trigger_type":"manual","dry_run":false,"force":true}' \
     --cli-binary-format raw-in-base64-out \
     /tmp/out.json && cat /tmp/out.json
   ```

   Confirm:
   - Secrets Manager shows a new `AWSCURRENT` version with a fresh `rotated_at`.
   - `AWSPREVIOUS` holds the seeded value.
   - Slack receives a success message.
   - CloudWatch metric `ZTMF/Kion/DaysSinceRotation` published with value 0.

5. Let the scheduled run own the secret from this point. No manual rotation needed unless something fails.

## Runbook

### Alarm: Lambda errors > 0

Check CloudWatch Logs for the most recent invocation. Common causes:

- **Kion `401 Unauthorized`**: the current key is dead. Someone rotated it outside of the Lambda (UI, another script). Recovery:
  1. Generate a new key in Kion UI.
  2. Paste the new value into the secret with `put-secret-value` as in bootstrap step 2.
  3. Invoke with `{"force":true}` to verify the new key works.
- **Secrets Manager `ResourceNotFoundException`**: the secret has not been seeded in this environment. Follow the bootstrap steps.
- **VPC egress timeout**: Kion host is unreachable from Lambda subnets. Check the `ztmf_sync_lambda` security group egress rules and the NAT gateway route.

### Alarm: DLQ depth > 0

Inspect the failed invocation:

```
aws sqs receive-message --queue-url <dlq-url> --max-number-of-messages 5
```

The message body is the original EventBridge payload. After fixing the root cause, drain the DLQ:

```
aws sqs purge-queue --queue-url <dlq-url>
```

### Alarm: DaysSinceRotation >= 6

The Lambda has not successfully rotated for nearly a full Kion expiry window. Treat as urgent:

1. Check the schedule rule is ENABLED: `aws events describe-rule --name ztmf-kion-key-rotate-schedule-<env>`.
2. Check Lambda logs for the most recent run.
3. If the scheduler is broken, invoke the Lambda manually with `{"force":true}` to rotate immediately.

### Critical Slack alert with a recovery key

The Lambda emits a Slack alert containing the new Kion key only when Kion accepted the rotation but writing the secret back failed after all retries. The previous key is now dead. Recovery:

1. Copy the key from the Slack alert.
2. Put it into the secret:

   ```
   aws secretsmanager put-secret-value \
     --secret-id ztmf_kion_<env> \
     --secret-string '{"api_key":"<KEY>","base_url":"<url>","rotated_at":"<now>"}'
   ```

3. Delete the Slack message after you have confirmed the secret is in place.
4. Investigate why the Secrets Manager write failed (IAM drift, KMS key revoked, API throttling).

## Local development

The Lambda talks to AWS Secrets Manager, CloudWatch, and the public Kion tenant. There is no local-development compose for this binary. Unit tests under `internal/kion` and `internal/rotate` run offline with stub implementations:

```
cd backend
go test ./cmd/lambda-kion-key-rotate/...
```

## Design decision: no AWS Secrets Manager native rotation

AWS Foundational Security Best Practices (FSBP) check `secretsmanager-auto-rotation-enabled-check` expects every secret to have an `aws_secretsmanager_secret_rotation` resource attached that drives the four-step rotation contract (`createSecret`, `setSecret`, `testSecret`, `finishSecret`) with an `AWSPENDING` staging label for safe cutover. We deliberately do not use that contract here.

**Why:** the native contract assumes AWS owns the rotation primitive (as with RDS auto-rotation, DocDB, Redshift). Kion owns the rotation primitive for its API keys. `POST /api/v3/app-api-key/rotate` is a single atomic operation: one call, old key invalidated server-side, new key returned. There is no provider-side concept of a staged or pending credential that our Lambda could promote, so faking out the four-step contract would mean calling Kion during `createSecret`, no-oping `setSecret`, stubbing `testSecret`, and shuffling version stages in `finishSecret`. That is more code and more failure modes for no operational win.

**What we do instead:**

- Scheduled EventBridge invocation with an idempotency check.
- Direct `PutSecretValue` on success; AWS still moves the previous version to `AWSPREVIOUS` automatically.
- A narrow recovery window: if Kion rotates but `PutSecretValue` fails after retries, the orchestrator emits a critical Slack alert containing the new key so an operator can paste it in.

If a CMS Security Hub scan flags this secret for missing native rotation, route the finding here for context. File an exception or re-evaluate if the organization's policy changes.

## Related

- Issue: https://github.com/CMS-Enterprise/ztmf-misc/issues/167
- Prior art: `backend/cmd/lambda-cfacts-snowflake/` for the Lambda pattern.
- Secrets helper: `backend/internal/secrets/secrets.go` (`Put` was added for this Lambda).
- Slack helper: `backend/internal/notifications/slack.go` (`SendRotationNotification` was added for this Lambda).
- Retired local cron: `~/projects/prod/cms-kion-automation` (`uv run kion rotate`).
