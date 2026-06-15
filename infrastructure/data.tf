data "aws_caller_identity" "current" {}

data "aws_region" "current" {}

data "aws_vpc" "ztmf" {
  filter {
    name = "tag:Name"
    // impl rides on dev's CMS-provisioned VPC; CMS Cloud has not issued a
    // separate ztmf-east-impl VPC. Resource-name suffix in locals keeps every
    // VPC-scoped resource (SGs, target groups, ALB) from colliding with dev.
    values = ["ztmf-east-${var.environment == "impl" ? "dev" : var.environment}"]
  }
}

data "aws_subnets" "private" {
  filter {
    name   = "vpc-id"
    values = [data.aws_vpc.ztmf.id]
  }

  filter {
    name   = "tag:use"
    values = ["private"]
  }
}

data "aws_subnet" "private" {
  for_each = toset(data.aws_subnets.private.ids)
  id       = each.value
}


# Commented out for now since they are not currently needed
# data "aws_subnets" "public" {
#   filter {
#     name   = "vpc-id"
#     values = [data.aws_vpc.ztmf.id]
#   }

#   filter {
#     name   = "tag:use"
#     values = ["private"]
#   }
# }

data "aws_ec2_managed_prefix_list" "cloudfront" {
  name = "com.amazonaws.global.cloudfront.origin-facing"
}

data "aws_ec2_managed_prefix_list" "shared_services" {
  name = "cmscloud-shared-services"
}

data "aws_secretsmanager_secret" "ztmf_va_trust_provider" {
  arn = aws_secretsmanager_secret.ztmf_va_trust_provider.arn
}

data "aws_secretsmanager_secret_version" "ztmf_va_trust_provider_current" {
  secret_id = data.aws_secretsmanager_secret.ztmf_va_trust_provider.id
}

data "aws_secretsmanager_secret" "ztmf_db_user" {
  arn = aws_secretsmanager_secret.ztmf_db_user.arn
}

data "aws_secretsmanager_secret_version" "ztmf_db_user_current" {
  secret_id = data.aws_secretsmanager_secret.ztmf_db_user.id
}

// Entra OIDC config + session signing key. Read only when entra_enabled so the
// first apply can create the secrets and let the operator seed them before
// anything consumes them (an unseeded secret has no version and would fail
// this read).
data "aws_secretsmanager_secret_version" "ztmf_entra_oidc_current" {
  count     = var.entra_enabled ? 1 : 0
  secret_id = aws_secretsmanager_secret.ztmf_entra_oidc.id
}

// The session signing key is consumed by ECS as a container secret (injected by
// ARN via local.entra_api_secrets), so Terraform never needs to read its plaintext
// value. No data-source read here, by design.

// Note: this used to be a `data "aws_secretsmanager_secrets" "rds"` lookup that
// found the RDS-managed master password secret by tag. The Secrets Manager API
// `tag-value` filter is prefix-matching, so once impl was provisioned the same
// filter started matching both dev's "cluster:ztmf" and impl's
// "cluster:ztmf-impl" tag values, and join("") concatenated the two ARNs into
// a corrupt DB_SECRET_ID that broke the ECS task definition. The cluster
// resource's master_user_secret[0].secret_arn is exposed directly, so no
// Secrets Manager lookup is needed (see locals.tf db_cred_secret).

# Account-singletons that impl reuses from dev's state.
# count = 1 only when this env doesn't manage them as resources, i.e. impl.

data "aws_ecr_repository" "ztmf_api" {
  count = local.manage_account_singletons ? 0 : 1
  name  = "ztmf/api"
}

data "aws_ecr_repository" "ztmf_ops" {
  count = local.manage_account_singletons ? 0 : 1
  name  = "ztmf/ops"
}

data "aws_secretsmanager_secret" "ztmf_smtp" {
  count = local.manage_account_singletons ? 0 : 1
  name  = "ztmf_smtp"
}

data "aws_secretsmanager_secret" "ztmf_smtp_ca_root" {
  count = local.manage_account_singletons ? 0 : 1
  name  = "ztmf_smtp_ca_root"
}

data "aws_secretsmanager_secret" "ztmf_smtp_intermediate" {
  count = local.manage_account_singletons ? 0 : 1
  name  = "ztmf_smtp_intermediate"
}

data "aws_secretsmanager_secret" "ztmf_slack_webhook" {
  count = local.manage_account_singletons ? 0 : 1
  name  = "ztmf_slack_webhook"
}

data "aws_security_group" "ztmf_vpc_endpoints" {
  count  = local.manage_vpc_endpoints ? 0 : 1
  name   = "ztmf_vpc_endpoints"
  vpc_id = data.aws_vpc.ztmf.id
}

# data "aws_network_interface" "s3" {
#   count      = length(data.aws_subnets.private.ids)
#   id         = flatten(aws_vpc_endpoint.ztmf["s3"][*].network_interface_ids)[count.index]
#   depends_on = [aws_vpc_endpoint.ztmf["s3"]]
# }

data "aws_ssm_parameter" "ztmf_api_tag" {
  // dev/prod use the historical ztmf_api_tag parameter name; impl uses a
  // namespaced parameter to avoid sharing the dev image-tag pointer.
  name = local.manage_account_singletons ? "ztmf_api_tag" : "ztmf-impl_api_tag"
}

data "aws_ssm_parameter" "ztmf_ops_tag" {
  name       = aws_ssm_parameter.ztmf_ops_tag.name
  depends_on = [aws_ssm_parameter.ztmf_ops_tag]
}

// ACM certificate ARN sourced from SSM Parameter Store, the same parameter
// the cert-rotation Lambda re-imports over. Single source of truth across
// CloudFront, ALB, and the rotation Lambda. Replaces the older
// data "aws_acm_certificate" lookup, which filtered on `domain = "dev.ztmf.cms.gov"`
// + `most_recent = true` and was non-deterministic when two ISSUED certs
// shared a CN; that ambiguity caused the 2026-04-30 production incident.
//
// Operator seeds the value once per AWS account with
//   aws ssm put-parameter --name /ztmf/<env>/cert-rotation/acm-arn \
//     --type String --value "arn:aws:acm:..."
data "aws_ssm_parameter" "ztmf_acm_arn" {
  name = "/ztmf/${var.environment}/cert-rotation/acm-arn"
}

locals {
  ztmf_acm_certificate_arn = nonsensitive(data.aws_ssm_parameter.ztmf_acm_arn.value)
}

// CMS cloud provided the following stack in each account for the preconfigured CMS cloud WAF
data "aws_cloudformation_stack" "web_acl" {
  name = "SamShieldAdvancedWaf"
}
