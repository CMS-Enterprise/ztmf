// Everything below is dev's, read by name or tag so this root never owns or
// mutates a dev-root resource.
data "aws_vpc" "ztmf" {
  filter {
    name   = "tag:Name"
    values = ["ztmf-east-dev"]
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

data "aws_ecs_cluster" "ztmf" {
  cluster_name = "ztmf"
}

data "aws_lb" "ztmf_api" {
  name = "ztmf-api"
}

data "aws_lb_listener" "https" {
  load_balancer_arn = data.aws_lb.ztmf_api.arn
  port              = 443
}

data "aws_security_group" "alb" {
  name   = "ztmf"
  vpc_id = data.aws_vpc.ztmf.id
}

data "aws_ecr_repository" "api" {
  name = "ztmf/api"
}

data "aws_ecr_repository" "ui" {
  name = "ztmf/ui"
}

// Same Okta client the dev root's login rules use; the redirect URI registered
// with Okta is the dev host's /oauth2/idpresponse, which this rule shares.
data "aws_secretsmanager_secret_version" "okta" {
  secret_id = "ztmf_va_trust_provider"
}

data "aws_kms_alias" "ssm" {
  name = "alias/aws/ssm"
}

locals {
  oidc = jsondecode(data.aws_secretsmanager_secret_version.okta.secret_string)
}
