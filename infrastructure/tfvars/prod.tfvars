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

# Aurora PITR at the Aurora maximum. Set explicitly rather than relying on the
# default so the prod value is visible next to the environment it protects.
db_backup_retention_days = 35

# Monthly logical backup to S3. This is the environment the strategy exists for
# - prod holds the submitted data call responses that the training-session edit
# incident put at risk.
db_dump_enabled = true
