resource "aws_lb_target_group" "env" {
  name        = "ztmf-${local.env_name}"
  port        = 443
  protocol    = "HTTPS"
  target_type = "ip"
  vpc_id      = data.aws_vpc.ztmf.id

  health_check {
    protocol            = "HTTPS"
    port                = 443
    path                = "/healthz"
    matcher             = "200"
    interval            = 15
    healthy_threshold   = 2
    unhealthy_threshold = 3
  }
}

// Login at the edge, then forward to this environment's target group: CMS
// login via the dev Okta client by default, or this environment's Cognito pool
// (cognito.tf). Only the PR prefix matches, so PR traffic can never reach dev's
// own /api/* rule.
resource "aws_lb_listener_rule" "env" {
  listener_arn = data.aws_lb_listener.https.arn
  priority     = local.listener_priority

  action {
    type = local.cognito ? "authenticate-cognito" : "authenticate-oidc"

    dynamic "authenticate_oidc" {
      for_each = local.cognito ? [] : [1]
      content {
        authorization_endpoint     = local.oidc["authorization_endpoint"]
        client_id                  = local.oidc["client_id"]
        client_secret              = local.oidc["client_secret"]
        issuer                     = local.oidc["issuer"]
        token_endpoint             = local.oidc["token_endpoint"]
        user_info_endpoint         = local.oidc["user_info_endpoint"]
        scope                      = "openid profile email groups"
        session_timeout            = 10800
        on_unauthenticated_request = "authenticate"
      }
    }

    // A demo session lasts the ALB's maximum, 7 days, and its cookie is named
    // for this environment: dev's Okta rule uses the default name on the same
    // host, so a shared name would let one session clobber the other.
    dynamic "authenticate_cognito" {
      for_each = local.cognito ? [1] : []
      content {
        user_pool_arn              = aws_cognito_user_pool.env[0].arn
        user_pool_client_id        = aws_cognito_user_pool_client.env[0].id
        user_pool_domain           = aws_cognito_user_pool_domain.env[0].domain
        scope                      = "openid email profile"
        session_cookie_name        = "AWSELBAuthSessionCookie-${local.env_name}"
        session_timeout            = 604800
        on_unauthenticated_request = "authenticate"
      }
    }
  }

  action {
    type             = "forward"
    target_group_arn = aws_lb_target_group.env.arn
  }

  condition {
    path_pattern {
      values = [local.path_prefix, "${local.path_prefix}/*"]
    }
  }
}
