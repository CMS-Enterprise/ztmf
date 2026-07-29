# impl runs in the dev AWS account alongside dev. Resource names suffix-rendered
# to "_impl" / "-impl" via locals.tf. Snowflake/Kion sync disabled at the
# schedule level until SDL/Kion coordination lands; cert-rotation Lambda is on
# and re-uses dev's existing multi-SAN ACM certificate (impl.ztmf.cms.gov SAN).

environment            = "impl"
domain_name_prefix     = "impl."
ecs_service_task_count = 1


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

# Backups: impl is a throwaway experiment environment sharing the dev account.
# Keep PITR at the 1-day minimum and leave the monthly dump schedule off (the
# bucket and task definition are still created, so it can be switched on for a
# single experiment without an infrastructure change).
db_backup_retention_days = 1
db_dump_enabled          = false
