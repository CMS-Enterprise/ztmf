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

// CMS login at the edge via the dev Okta client, then forward to this
// environment's target group. Only the PR prefix matches, so PR traffic can
// never reach dev's own /api/* rule.
resource "aws_lb_listener_rule" "env" {
  listener_arn = data.aws_lb_listener.https.arn
  priority     = local.listener_priority

  action {
    type = "authenticate-oidc"

    authenticate_oidc {
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
