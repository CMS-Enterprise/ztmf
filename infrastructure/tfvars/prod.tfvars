# specific to ztmf prod account
environment            = "prod"
domain_name_prefix     = ""
ecs_service_task_count = 1
# job_code = "ZTMF_SCORING_USER"

# Entra dual-IdP. Validated on dev since #350 (2026-06-17), so flipping this to
# true adds the per-IdP ALB rules, moves /api/* off ALB OIDC to backend session
# validation, and injects the Entra + session env into the API task. Okta login
# is unchanged. Required for the 2026 data call: HHS/OpDiv participants
# authenticate via Entra (users.identity_provider, migration 0030) and have no
# login path in prod while this is false.
entra_enabled = true

# Login/OIDC alarm routing. Subscribes this address to the ztmf-alarms SNS topic.
# A shared inbox rather than an individual, so paging never depends on one
# person being reachable. AWS sends a subscription confirmation email that has
# to be clicked once before anything is delivered; until then the alarms exist
# but have no destination, which is the state prod was in with this unset.
alarm_notification_email = "ISPGZeroTrust@cms.hhs.gov"

# TLS cert rotation Lambda
# ACM ARN sourced from SSM Parameter Store /ztmf/prod/cert-rotation/acm-arn
enable_cert_rotation_lambda = true
cert_rotation_prefix        = "prod"
cert_rotation_domain        = "ztmf.cms.gov"

# Aurora PITR at the Aurora maximum. Set explicitly rather than relying on the
# default so the prod value is visible next to the environment it protects.
db_backup_retention_days = 35

# Monthly logical backup to S3. This is the environment the strategy exists for
# - prod holds the submitted data call responses that the training-session edit
# incident put at risk.
db_dump_enabled = true
