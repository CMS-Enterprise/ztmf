data "aws_caller_identity" "current" {}

data "aws_region" "current" {}

data "aws_vpc" "ztmf" {
  filter {
    name   = "tag:Name"
    values = ["ztmf-east-${local.vpc_environment}"]
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
    values = ["arn:aws:rds:us-east-1:${local.account_id}:cluster:${local.name_prefix}"]
  }
}

# data "aws_network_interface" "s3" {
#   count      = length(data.aws_subnets.private.ids)
#   id         = flatten(aws_vpc_endpoint.ztmf["s3"][*].network_interface_ids)[count.index]
#   depends_on = [aws_vpc_endpoint.ztmf["s3"]]
# }

data "aws_ssm_parameter" "ztmf_api_tag" {
  name = "${local.name_prefix}_api_tag"
}

// this resource needed to be created manually by importing a Digitcert certificate
// the cert's primary domain is dev.ztmf.cms.gov with other environments as SANs
// impl.ztmf.cms.gov must be added as a SAN before deploying the impl environment
data "aws_acm_certificate" "ztmf" {
  domain      = "dev.ztmf.cms.gov"
  statuses    = ["ISSUED"]
  most_recent = true
}

// CMS cloud provided the following stack in each account for the preconfigured CMS cloud WAF
data "aws_cloudformation_stack" "web_acl" {
  name = "SamShieldAdvancedWaf"
}
