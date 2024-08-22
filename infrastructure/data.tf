data "aws_caller_identity" "current" {}

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

data "aws_subnets" "public" {
  filter {
    name   = "vpc-id"
    values = [data.aws_vpc.ztmf.id]
  }

  filter {
    name   = "tag:use"
    values = ["public"]
  }
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

data "aws_secretsmanager_secret" "ztmf_x_auth_token" {
  arn = aws_secretsmanager_secret.ztmf_x_auth_token.arn
}

data "aws_secretsmanager_secret_version" "ztmf_x_auth_token_current" {
  secret_id = data.aws_secretsmanager_secret.ztmf_x_auth_token.id
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

data "aws_network_interface" "s3" {
  count      = length(data.aws_subnets.private.ids)
  id         = flatten(aws_vpc_endpoint.ztmf["s3"][*].network_interface_ids)[count.index]
  depends_on = [aws_vpc_endpoint.ztmf["s3"]]
}

data "aws_ssm_parameter" "ztmf_api_tag" {
  name = "ztmf_api_tag"
}

// this resource needed to be created manually by importing a Digitcert certificate
data "aws_acm_certificate" "ztmf" {
  domain   = local.domain_name
  statuses = ["ISSUED"]
}
