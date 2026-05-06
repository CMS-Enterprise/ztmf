# impl runs in the dev AWS account alongside dev. Resource names suffix-rendered
# to "_impl" / "-impl" via locals.tf. Snowflake/Kion sync disabled at the
# schedule level until SDL/Kion coordination lands; cert-rotation Lambda is on
# and re-uses dev's existing multi-SAN ACM certificate (impl.ztmf.cms.gov SAN).

environment            = "impl"
domain_name_prefix     = "impl."
ecs_service_task_count = 1

# CFACTS / Snowflake sync configuration
# Impl has no Snowflake account; left blank, schedule disabled via locals.
cfacts_snowflake_view  = ""
snowflake_table_prefix = "ZTMF"

# Kion rotation: impl has no ztmf_kion_impl secret yet; keep schedule off
kion_rotate_schedule_enabled = false

# TLS cert rotation Lambda
# ACM ARN sourced from SSM Parameter Store /ztmf/impl/cert-rotation/acm-arn,
# which is seeded with the dev account's existing multi-SAN cert ARN.
#
# IMPORTANT: the cert rotation Lambda runs with DRY_RUN=true for any env
# other than prod (see lambda-cert-rotation.tf). Cert bundle uploads to
# s3://ztmf-cert-rotation-impl/impl/ are validated and archived but are
# NEVER imported into ACM. impl reuses dev's already-imported multi-SAN
# cert; rotations are exercised in prod only.
enable_cert_rotation_lambda = true
cert_rotation_prefix        = "impl"
cert_rotation_domain        = "impl.ztmf.cms.gov"
