# specific to ztmf prod account
environment            = "prod"
domain_name_prefix     = ""
ecs_service_task_count = 1
# job_code = "ZTMF_SCORING_USER"

# Entra dual-IdP. Keep false until validated on dev and both secrets are
# seeded in the prod account (scripts/bootstrap-entra-secrets.sh), then flip to
# true to enable the second identity provider in production.
entra_enabled = false


# TLS cert rotation Lambda
# ACM ARN sourced from SSM Parameter Store /ztmf/prod/cert-rotation/acm-arn
enable_cert_rotation_lambda = true
cert_rotation_prefix        = "prod"
cert_rotation_domain        = "ztmf.cms.gov"
