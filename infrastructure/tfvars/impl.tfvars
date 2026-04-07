# ZTMF impl environment — lives in the dev AWS account alongside dev
# Shares dev VPC (ztmf-east-dev) but has its own RDS, ECS, ALB, CloudFront, etc.
environment            = "impl"
domain_name_prefix     = "impl."
ecs_service_task_count = 1
aurora_min_capacity    = 0.5
aurora_max_capacity    = 0.5
vpc_environment        = "dev" # share dev's VPC since there is no ztmf-east-impl

# CFACTS / Snowflake sync configuration
cfacts_snowflake_view  = "BUS_ZEROTRUST.PRIVATE.VW_CFACTS_SYSTEMS_FOR_ZTMF"
snowflake_table_prefix = "ZTMF"
