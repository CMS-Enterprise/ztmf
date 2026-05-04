# specific to ztmf dev account
environment            = "dev"
domain_name_prefix     = "dev."
ecs_service_task_count = 1
# job_code = "ZTMF_SCORING_USER"

# CFACTS / Snowflake sync configuration
cfacts_snowflake_view  = "BUS_ZEROTRUST.ENRICHMENT.VW_CFACTS_SYSTEMS_FOR_ZTMF"
snowflake_table_prefix = "ZTMF"

# TLS cert rotation Lambda
# ACM ARN sourced from SSM Parameter Store /ztmf/dev/cert-rotation/acm-arn
enable_cert_rotation_lambda = true
cert_rotation_prefix        = "dev"
cert_rotation_domain        = "dev.ztmf.cms.gov"
