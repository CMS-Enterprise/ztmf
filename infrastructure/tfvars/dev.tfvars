# specific to ztmf dev account
environment            = "dev"
domain_name_prefix     = "dev."
ecs_service_task_count = 1
# job_code = "ZTMF_SCORING_USER"

# HHS Entra dual-IdP. Keep false until both secrets are seeded in the dev
# account (scripts/bootstrap-entra-secrets.sh), then flip to true to add the
# per-IdP ALB rules, move /api/* off ALB OIDC to backend session validation,
# and inject the Entra + session env into the API task.
entra_enabled = false


# TLS cert rotation Lambda
# ACM ARN sourced from SSM Parameter Store /ztmf/dev/cert-rotation/acm-arn
enable_cert_rotation_lambda = true
cert_rotation_prefix        = "dev"
cert_rotation_domain        = "dev.ztmf.cms.gov"
