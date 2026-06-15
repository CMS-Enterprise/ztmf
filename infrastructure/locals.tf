locals {
  // adding reference here to make other references shorter to type "local.account_id" :)
  account_id = data.aws_caller_identity.current.account_id

  // Aurora's manage_master_user_password=true generates a password and stores
  // it in a Secrets Manager secret tagged with the cluster ARN. The cluster
  // resource exposes that secret's ARN directly via master_user_secret, which
  // is the authoritative reference and avoids a Secrets Manager tag lookup.
  // The earlier data.aws_secretsmanager_secrets filter returned both dev's
  // and impl's secrets once both clusters lived in the same account because
  // tag-value matching is prefix-based ("cluster:ztmf" is a prefix of
  // "cluster:ztmf-impl").
  db_cred_secret = aws_rds_cluster.ztmf.master_user_secret[0].secret_arn

  domain_name = "${var.domain_name_prefix}ztmf.cms.gov"

  // simplify referencing of json object fields for aws_verifiedaccess_trust_provider.ztmf_idmokta.oidc_options
  // technically only one of the fields was a true secret (client_secret), but since we have the space here
  //  we can use it to simplify code instead of placing all the other fields in TF vars
  oidc_options = jsondecode(data.aws_secretsmanager_secret_version.ztmf_va_trust_provider_current.secret_string)

  // Entra OIDC config, decoded only when entra_enabled (the data source is
  // gated on the same flag). Empty map otherwise so references compile while
  // the feature is off.
  entra_oidc_options = var.entra_enabled ? jsondecode(data.aws_secretsmanager_secret_version.ztmf_entra_oidc_current[0].secret_string) : {}

  // Extra API task environment when the dual-IdP feature is on. Both issuers
  // must be set together: once issuer validation is active, an Okta token whose
  // issuer is not configured would be rejected, so the Okta issuer (already in
  // the existing OIDC secret) is wired alongside the Entra values. These are
  // non-secret OIDC discovery values; the session signing key is injected
  // separately as a container secret.
  entra_api_env = var.entra_enabled ? [
    { name = "AUTH_OKTA_ISSUER", value = local.oidc_options["issuer"] },
    { name = "AUTH_ENTRA_ISSUER", value = local.entra_oidc_options["issuer"] },
    { name = "AUTH_ENTRA_JWKS_URL", value = local.entra_oidc_options["jwks_uri"] },
    { name = "AUTH_ENTRA_TENANT_ID", value = local.entra_oidc_options["tenant_id"] },
    # Pin the accepted audience to the ZTMF Entra app's client id so a validly
    # signed token minted for a different app in the same tenant is rejected.
    { name = "AUTH_ENTRA_AUDIENCE", value = local.entra_oidc_options["client_id"] },
    { name = "AUTH_COOKIE_DOMAIN", value = local.domain_name },
  ] : []

  entra_api_secrets = var.entra_enabled ? [
    { name = "AUTH_SESSION_SIGNING_SECRET", valueFrom = aws_secretsmanager_secret.ztmf_session_signing_key.arn },
  ] : []

  // impl shares the dev AWS account and dev VPC; per-env suffix renames every
  // VPC- and account-scoped resource that would otherwise collide with dev's.
  // dev/prod render an empty suffix so existing AWS object names (and the
  // terraform state addresses that bind to them) stay literal.
  name_suffix    = var.environment == "impl" ? "-impl" : ""
  underscore_sfx = var.environment == "impl" ? "_impl" : ""

  ztmf_name          = "ztmf${local.name_suffix}"
  ztmf_api_name      = "ztmf-api${local.name_suffix}"
  ztmf_rest_api_tg   = "ztmf-rest-api${local.name_suffix}"
  ztmf_db_sg_name    = "ztmf_db${local.underscore_sfx}"
  ztmf_api_log_group = "ztmf_api${local.underscore_sfx}"
  ztmf_ops_log_group = "ztmf_ops${local.underscore_sfx}"
  ztmf_vpce_sg_name  = "ztmf_vpc_endpoints${local.underscore_sfx}"
  ztmf_alb_sg_name   = "ztmf${local.name_suffix}"
  ztmf_api_task_sg   = "ztmf-api-task${local.name_suffix}"
  ztmf_ops_task_sg   = "ztmf_ops_task${local.underscore_sfx}"

  // True account-level singletons (ECR repos, OIDC provider, shared SMTP/TLS
  // secrets, account-shared S3 log bucket). Created in dev/prod, reused by
  // impl via data sources so two states never own the same physical resource.
  manage_account_singletons = var.environment != "impl"

  // dev's VPC already owns the 9 interface endpoints listed in vpc.tf.
  // impl reuses them rather than racing dev's state for ownership.
  manage_vpc_endpoints = var.environment != "impl"

  // Reference shims: pick the resource for dev/prod, the data source for impl.
  // Keeps every other file's reference site identical regardless of env.
  ecr_api_repo_url = local.manage_account_singletons ? aws_ecr_repository.ztmf_api[0].repository_url : data.aws_ecr_repository.ztmf_api[0].repository_url
  ecr_ops_repo_url = local.manage_account_singletons ? aws_ecr_repository.ztmf_ops[0].repository_url : data.aws_ecr_repository.ztmf_ops[0].repository_url

  smtp_secret_arn       = local.manage_account_singletons ? aws_secretsmanager_secret.ztmf_smtp[0].arn : data.aws_secretsmanager_secret.ztmf_smtp[0].arn
  smtp_ca_root_arn      = local.manage_account_singletons ? aws_secretsmanager_secret.ztmf_smtp_ca_root[0].arn : data.aws_secretsmanager_secret.ztmf_smtp_ca_root[0].arn
  smtp_intermediate_arn = local.manage_account_singletons ? aws_secretsmanager_secret.ztmf_smtp_intermediate[0].arn : data.aws_secretsmanager_secret.ztmf_smtp_intermediate[0].arn

  slack_webhook_arn  = local.manage_account_singletons ? aws_secretsmanager_secret.ztmf_slack_webhook[0].arn : data.aws_secretsmanager_secret.ztmf_slack_webhook[0].arn
  slack_webhook_name = local.manage_account_singletons ? aws_secretsmanager_secret.ztmf_slack_webhook[0].name : data.aws_secretsmanager_secret.ztmf_slack_webhook[0].name
}
