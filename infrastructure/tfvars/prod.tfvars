# specific to ztmf prod account
environment            = "prod"
domain_name_prefix     = ""
ecs_service_task_count = 1
# job_code = "ZTMF_SCORING_USER"

# Entra dual-IdP. Keep false until validated on dev and both secrets are
# seeded in the prod account (scripts/bootstrap-entra-secrets.sh), then flip to
# true to enable the second identity provider in production.
entra_enabled = false

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
