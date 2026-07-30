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


# TLS cert rotation Lambda
# ACM ARN sourced from SSM Parameter Store /ztmf/prod/cert-rotation/acm-arn
enable_cert_rotation_lambda = true
cert_rotation_prefix        = "prod"
cert_rotation_domain        = "ztmf.cms.gov"
