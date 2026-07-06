# specific to ztmf dev account
environment            = "dev"
domain_name_prefix     = "dev."
ecs_service_task_count = 1
# job_code = "ZTMF_SCORING_USER"

# Entra dual-IdP. Both secrets (ztmf_entra_oidc, ztmf_session_signing_key) are
# seeded and verified in the dev account, so flipping this to true adds the
# per-IdP ALB rules, moves /api/* off ALB OIDC to backend session validation,
# and injects the Entra + session env into the API task. Okta login is
# unchanged. Activates the dormant infra from #341 for the dev Entra pilot.
entra_enabled = true


# Login/OIDC alarm routing. Subscribes this address to the ztmf-alarms SNS topic
# (confirm the AWS subscription email once after first apply).
# INTERIM destination only — an individual address used to stand up alerting for
# the dev Entra pilot. Replace it with a shared team inbox or a Slack webhook
# before this routing is relied on beyond dev / before prod, so paging never
# depends on one person being reachable.
alarm_notification_email = "jono@aquia.us"

# TLS cert rotation Lambda
# ACM ARN sourced from SSM Parameter Store /ztmf/dev/cert-rotation/acm-arn
enable_cert_rotation_lambda = true
cert_rotation_prefix        = "dev"
cert_rotation_domain        = "dev.ztmf.cms.gov"
