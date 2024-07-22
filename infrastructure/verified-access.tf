resource "aws_verifiedaccess_trust_provider" "ztmf_idmokta" {
  description              = "ZTMF IDM/OKTA"
  trust_provider_type      = "user"
  user_trust_provider_type = "oidc"
  policy_reference_name    = "ztmf_idm_okta"
  oidc_options {
    authorization_endpoint = local.oidc_options["authorization_endpoint"]
    client_id              = local.oidc_options["client_id"]
    issuer                 = local.oidc_options["issuer"]
    scope                  = "openid profile email groups"
    token_endpoint         = local.oidc_options["token_endpoint"]
    user_info_endpoint     = local.oidc_options["user_info_endpoint"]
    client_secret          = local.oidc_options["client_secret"]
  }
  tags = {
    "Name" = "ztmf_idm_okta"
  }
}

resource "aws_cloudwatch_log_group" "ztmf_va_cwl" {
  name = "ztmf_verified_access"
}

resource "aws_verifiedaccess_instance" "ztmf_va" {
  description  = "Verified Access instance for ZTMF Scoring Tool"
  fips_enabled = false // true for prod
  tags = {
    Name = "ztmf"
  }
}

resource "aws_verifiedaccess_instance_logging_configuration" "ztmf_va" {
  access_logs {
    include_trust_context = true
    log_version           = "ocsf-1.0.0-rc.2"
    cloudwatch_logs {
      enabled   = true
      log_group = aws_cloudwatch_log_group.ztmf_va_cwl.id
    }
  }
  verifiedaccess_instance_id = aws_verifiedaccess_instance.ztmf_va.id
}

resource "aws_verifiedaccess_instance_trust_provider_attachment" "ztmf_va" {
  verifiedaccess_instance_id       = aws_verifiedaccess_instance.ztmf_va.id
  verifiedaccess_trust_provider_id = aws_verifiedaccess_trust_provider.ztmf_idmokta.id
}

resource "aws_verifiedaccess_group" "ztmf_va_users" {
  verifiedaccess_instance_id = aws_verifiedaccess_instance.ztmf_va.id
  policy_document            = <<-EOT
  permit(principal,action,resource)
  when {
    context.http_request.http_method != "INVALID_METHOD"
  };
  EOT
}

resource "aws_security_group" "ztmf_va_ep_sg" {
  name        = "ztmf-verified-access-endpoint"
  description = "Allow TLS inbound traffic"
  vpc_id      = data.aws_vpc.ztmf.id

  ingress {
    from_port   = 443
    to_port     = 443
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  egress {
    from_port   = 443
    to_port     = 443
    protocol    = "tcp"
    cidr_blocks = [data.aws_vpc.ztmf.cidr_block]
  }
}

resource "aws_verifiedaccess_endpoint" "ztmf_va_ep" {
  application_domain     = local.domain_name
  attachment_type        = "vpc"
  description            = "ztmf"
  domain_certificate_arn = data.aws_acm_certificate.ztmf.id
  endpoint_domain_prefix = "devztmfcmsgov"
  endpoint_type          = "load-balancer"
  load_balancer_options {
    load_balancer_arn = aws_lb.ztmf.arn
    port              = 443
    protocol          = "https"
    subnet_ids        = data.aws_subnets.public.ids
  }
  security_group_ids       = [aws_security_group.ztmf_va_ep_sg.id]
  verified_access_group_id = aws_verifiedaccess_group.ztmf_va_users.id
  policy_document          = <<-EOT
  permit(principal,action,resource)
  when {
    context has "ztmf_idm_okta" &&
    context.ztmf_idm_okta has "email" &&
    context.ztmf_idm_okta.email like "*@cms.hhs.gov" &&
    context.ztmf_idm_okta has "email_verified" &&
    context.ztmf_idm_okta.email_verified == true &&
    context.ztmf_idm_okta has "groups" &&
    context.ztmf_idm_okta.groups.contains("${(var.environment == "dev" ? "ZTMF_SCORING_USER_DEV" : "ZTMF_SCORING_USER")}")
  };
  EOT
}

output "aws_verifiedaccess_endpoint_domain" {
  value = aws_verifiedaccess_endpoint.ztmf_va_ep.endpoint_domain
}
