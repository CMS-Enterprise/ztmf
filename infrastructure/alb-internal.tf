resource "aws_security_group" "ztmf_alb" {
  name        = local.ztmf_alb_sg_name
  description = "Allow TLS inbound traffic"
  vpc_id      = data.aws_vpc.ztmf.id

  ingress {
    description     = "HTTPS from VPC CIDR"
    from_port       = 443
    to_port         = 443
    protocol        = "tcp"
    prefix_list_ids = [data.aws_ec2_managed_prefix_list.cloudfront.id]
  }

  // allow 443 egress to support OIDC integration with external vendor
  egress {
    from_port   = 443
    to_port     = 443
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }
}

resource "aws_lb" "ztmf_api" {
  name                       = local.ztmf_api_name
  internal                   = true
  load_balancer_type         = "application"
  security_groups            = [aws_security_group.ztmf_alb.id]
  subnets                    = data.aws_subnets.private.ids
  enable_deletion_protection = true

  # Per-request access logs record actions_executed + error_reason for failed
  # authenticate-oidc, the one signal missing when an Entra/Okta login breaks.
  # Bucket name is referenced literally (not aws_s3_bucket.ztmf_logs[0].id)
  # because that resource is count-gated on manage_account_singletons; the
  # literal name is what the bucket policy in s3.tf already grants the ELB log
  # delivery account on the rest-api-alb/* prefix. SSE-S3 on the bucket
  # satisfies ALB's SSE-S3-only requirement; no bucket/policy change needed.
  access_logs {
    bucket  = "ztmf-logs-${local.account_id}-use1"
    prefix  = "rest-api-alb"
    enabled = true
  }
}



# S3 related rules are for support the application behind Verified Access
# default rule will forward to s3 bucket
# resource "aws_lb_target_group" "s3" {
#   name        = "s3"
#   port        = 443
#   protocol    = "HTTPS"
#   target_type = "ip"
#   vpc_id      = data.aws_vpc.ztmf.id

#   health_check {
#     protocol            = "HTTPS"
#     port                = 443
#     matcher             = "200-499"
#     healthy_threshold   = 2
#     unhealthy_threshold = 2
#   }
# }

# resource "aws_lb_target_group_attachment" "s3" {
#   for_each         = toset(flatten(data.aws_network_interface.s3[*].private_ips))
#   target_group_arn = aws_lb_target_group.s3.arn
#   target_id        = each.value
#   port             = 443
# }

# TARGET GROUPS
resource "aws_lb_target_group" "ztmf_rest_api" {
  name        = local.ztmf_rest_api_tg
  port        = 443
  protocol    = "HTTPS"
  target_type = "ip"
  vpc_id      = data.aws_vpc.ztmf.id

  health_check {
    protocol            = "HTTPS"
    port                = 443
    matcher             = "200-499"
    healthy_threshold   = 2
    unhealthy_threshold = 2
  }
}

# LISTENER
resource "aws_lb_listener" "ztmf_api_https" {
  load_balancer_arn = aws_lb.ztmf_api.arn
  port              = "443"
  protocol          = "HTTPS"
  ssl_policy        = "ELBSecurityPolicy-TLS13-1-2-2021-06"
  certificate_arn   = local.ztmf_acm_certificate_arn

  default_action {
    type             = "forward"
    target_group_arn = aws_lb_target_group.ztmf_rest_api.arn
  }
}


# RULES
#
# Two rule sets, selected by var.entra_enabled:
#
#   entra_enabled = false (default, single-IdP Okta):
#     priority 1  /login*  authenticate-oidc(okta) -> redirect "/"   (authz)
#     priority 2  /api/*   authenticate-oidc(okta) -> forward         (ztmf_api)
#
#   entra_enabled = true (dual-IdP):
#     priority 1  /api/v1/auth/lookup*  forward, no OIDC              (lookup)
#     priority 2  /login/entra*  authenticate-oidc(entra) -> forward  (entra_login)
#     priority 3  /login*        authenticate-oidc(okta)  -> forward  (okta_login)
#     priority 4  /api/*         forward, no OIDC                      (api_forward)
#
# When the flag flips the old two rules are destroyed and the four new ones are
# created in the same apply, so the priorities never collide. The session cookie
# the backend mints on the /login* forward is what gates /api/* once ALB stops
# doing it. Do not flip until ztmf_session_signing_key and ztmf_entra_oidc are
# seeded in the target account (see scripts/bootstrap-entra-secrets.sh).

# Existing un-indexed addresses move under count, so toggling the flag is the
# only thing that creates/destroys these (no churn on unrelated applies).
moved {
  from = aws_lb_listener_rule.authz
  to   = aws_lb_listener_rule.authz[0]
}

moved {
  from = aws_lb_listener_rule.ztmf_api
  to   = aws_lb_listener_rule.ztmf_api[0]
}

resource "aws_lb_listener_rule" "authz" {
  count        = var.entra_enabled ? 0 : 1
  listener_arn = aws_lb_listener.ztmf_api_https.arn
  priority     = 1

  action {
    type = "authenticate-oidc"

    authenticate_oidc {
      authorization_endpoint     = local.oidc_options["authorization_endpoint"]
      client_id                  = local.oidc_options["client_id"]
      client_secret              = local.oidc_options["client_secret"]
      issuer                     = local.oidc_options["issuer"]
      token_endpoint             = local.oidc_options["token_endpoint"]
      user_info_endpoint         = local.oidc_options["user_info_endpoint"]
      scope                      = "openid profile email groups"
      session_timeout            = 10800
      on_unauthenticated_request = "authenticate"
    }
  }

  action {
    type = "redirect"
    redirect {
      status_code = "HTTP_302"
      path        = "/"
    }
  }

  condition {
    path_pattern {
      values = [
        "/login*",
      ]
    }
  }
}

resource "aws_lb_listener_rule" "ztmf_api" {
  count        = var.entra_enabled ? 0 : 1
  listener_arn = aws_lb_listener.ztmf_api_https.arn
  priority     = 2

  action {
    type = "authenticate-oidc"

    authenticate_oidc {
      authorization_endpoint     = local.oidc_options["authorization_endpoint"]
      client_id                  = local.oidc_options["client_id"]
      client_secret              = local.oidc_options["client_secret"]
      issuer                     = local.oidc_options["issuer"]
      token_endpoint             = local.oidc_options["token_endpoint"]
      user_info_endpoint         = local.oidc_options["user_info_endpoint"]
      scope                      = "openid profile email groups"
      session_timeout            = 3600
      on_unauthenticated_request = "deny"
    }
  }

  action {
    type             = "forward"
    target_group_arn = aws_lb_target_group.ztmf_rest_api.arn
  }

  condition {
    path_pattern {
      values = [
        "/api/*",
      ]
    }
  }
}

# ---- Dual-IdP rule set (entra_enabled = true) ----------------------------

# Priority 1: the unauthenticated pre-auth lookup. Must sort ahead of the
# /api/* rule so it is not swallowed by it. Plain forward, no OIDC, so the
# browser can resolve which IdP owns an email before any session exists.
resource "aws_lb_listener_rule" "lookup" {
  count        = var.entra_enabled ? 1 : 0
  listener_arn = aws_lb_listener.ztmf_api_https.arn
  priority     = 1

  action {
    type             = "forward"
    target_group_arn = aws_lb_target_group.ztmf_rest_api.arn
  }

  condition {
    path_pattern {
      values = ["/api/v1/auth/lookup*"]
    }
  }
}

# Priority 2: Entra login. Distinct session cookie name so it cannot
# collide with the Okta rule's default ALB session cookie. Forwards to the
# backend, which reads the forwarded IdP token and mints the app session.
resource "aws_lb_listener_rule" "entra_login" {
  count        = var.entra_enabled ? 1 : 0
  listener_arn = aws_lb_listener.ztmf_api_https.arn
  priority     = 2

  action {
    type = "authenticate-oidc"

    authenticate_oidc {
      authorization_endpoint     = local.entra_oidc_options["authorization_endpoint"]
      client_id                  = local.entra_oidc_options["client_id"]
      client_secret              = local.entra_oidc_options["client_secret"]
      issuer                     = local.entra_oidc_options["issuer"]
      token_endpoint             = local.entra_oidc_options["token_endpoint"]
      user_info_endpoint         = local.entra_oidc_options["user_info_endpoint"]
      scope                      = "openid profile email"
      session_cookie_name        = "AWSELBAuthSessionCookie-Entra"
      session_timeout            = 10800
      on_unauthenticated_request = "authenticate"
    }
  }

  action {
    type             = "forward"
    target_group_arn = aws_lb_target_group.ztmf_rest_api.arn
  }

  condition {
    path_pattern {
      values = ["/login/entra*"]
    }
  }
}

# Priority 3: Okta login, unchanged provider config, but the success action is
# now a forward to the backend (so it can mint the app session) rather than a
# redirect. The Okta app registration, client, and default cookie are untouched.
resource "aws_lb_listener_rule" "okta_login" {
  count        = var.entra_enabled ? 1 : 0
  listener_arn = aws_lb_listener.ztmf_api_https.arn
  priority     = 3

  action {
    type = "authenticate-oidc"

    authenticate_oidc {
      authorization_endpoint     = local.oidc_options["authorization_endpoint"]
      client_id                  = local.oidc_options["client_id"]
      client_secret              = local.oidc_options["client_secret"]
      issuer                     = local.oidc_options["issuer"]
      token_endpoint             = local.oidc_options["token_endpoint"]
      user_info_endpoint         = local.oidc_options["user_info_endpoint"]
      scope                      = "openid profile email groups"
      session_timeout            = 10800
      on_unauthenticated_request = "authenticate"
    }
  }

  action {
    type             = "forward"
    target_group_arn = aws_lb_target_group.ztmf_rest_api.arn
  }

  condition {
    path_pattern {
      values = ["/login*"]
    }
  }
}

# Priority 4: API traffic. ALB no longer runs OIDC here; the backend validates
# the app session cookie (or, in non-prod, a bearer token). Plain forward.
resource "aws_lb_listener_rule" "api_forward" {
  count        = var.entra_enabled ? 1 : 0
  listener_arn = aws_lb_listener.ztmf_api_https.arn
  priority     = 4

  action {
    type             = "forward"
    target_group_arn = aws_lb_target_group.ztmf_rest_api.arn
  }

  condition {
    path_pattern {
      values = ["/api/*"]
    }
  }
}
