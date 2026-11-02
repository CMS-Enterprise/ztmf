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

# Login/OIDC alarm routing. Subscribes this address to the ztmf-alarms SNS topic.
# A shared inbox rather than an individual, so paging never depends on one
# person being reachable. AWS sends a subscription confirmation email that has
# to be clicked once before anything is delivered; until then the alarms exist
# but have no destination, which is the state prod was in with this unset.
alarm_notification_email = "ISPGZeroTrust@cms.hhs.gov"

# TLS 1.3 only at the CloudFront edge, after a week in dev. A client
# that can only do TLS 1.2 gets a handshake failure.
cloudfront_minimum_protocol_version = "TLSv1.3_2025"

# Post-quantum FIPS policy on the internal ALB listener. Its backend policy
# offers hybrid ML-KEM to the API targets; dev shows hop=alb on X25519MLKEM768.
alb_ssl_policy = "ELBSecurityPolicy-TLS13-1-2-Res-FIPS-PQ-2025-09"

# TLS cert rotation Lambda
# ACM ARN sourced from SSM Parameter Store /ztmf/prod/cert-rotation/acm-arn
enable_cert_rotation_lambda = true
cert_rotation_prefix        = "prod"
cert_rotation_domain        = "ztmf.cms.gov"
