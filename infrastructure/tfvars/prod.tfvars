# specific to ztmf prod account
environment        = "prod"
domain_name_prefix = ""
ecs_service_task_count = 1
# job_code = "ZTMF_SCORING_USER"

# CFACTS / Snowflake sync configuration
cfacts_snowflake_view  = "BUS_ZEROTRUST.PRIVATE.VW_CFACTS_SYSTEMS_FOR_ZTMF"
snowflake_table_prefix = "ZTMF"
