data "aws_caller_identity" "current" {}

data "aws_region" "current" {}

data "aws_vpc" "ztmf" {
  filter {
    name   = "tag:Name"
    values = ["ztmf-east-${var.environment}"]
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
#     values = ["public"]
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

data "aws_secretsmanager_secrets" "rds" {
  filter {
    name   = "tag-key"
    values = ["aws:rds:primaryDBClusterArn"]
  }

  filter {
    name   = "tag-value"
    values = ["arn:aws:rds:us-east-1:${local.account_id}:cluster:ztmf"]
  }
}

# data "aws_network_interface" "s3" {
#   count      = length(data.aws_subnets.private.ids)
#   id         = flatten(aws_vpc_endpoint.ztmf["s3"][*].network_interface_ids)[count.index]
#   depends_on = [aws_vpc_endpoint.ztmf["s3"]]
# }

data "aws_ssm_parameter" "ztmf_api_tag" {
  name = "ztmf_api_tag"
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
