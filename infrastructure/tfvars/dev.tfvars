# specific to ztmf dev account
environment            = "dev"
domain_name_prefix     = "dev."
ecs_service_task_count = 1
# job_code = "ZTMF_SCORING_USER"

# CFACTS / Snowflake sync configuration
cfacts_snowflake_view  = "BUS_ZEROTRUST.ENRICHMENT.VW_CFACTS_SYSTEMS_FOR_ZTMF"
snowflake_table_prefix = "ZTMF"

# TLS cert rotation Lambda (disabled by default)
enable_cert_rotation_lambda   = false
cert_rotation_prefix          = "dev"
cert_rotation_domain          = "dev.ztmf.cms.gov"
cert_rotation_acm_certificate_arn = ""
