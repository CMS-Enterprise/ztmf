resource "aws_security_group" "ztmf_alb_public" {
  name        = "ztmf-alb-public"
  description = "Allow TLS inbound traffic"
  vpc_id      = data.aws_vpc.ztmf.id

  ingress {
    description = "HTTPS from everywhere"
    from_port   = 443
    to_port     = 443
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  // only initiate connections to IPs in private subnets
  egress {
    from_port   = 443
    to_port     = 443
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }
}

resource "aws_lb" "ztmf_rest_api" {
  name               = "ztmf-rest-api"
  internal           = false
  load_balancer_type = "application"
  security_groups    = [aws_security_group.ztmf_alb_public.id]
  subnets            = data.aws_subnets.public.ids

  enable_deletion_protection = false

  access_logs {
    bucket  = aws_s3_bucket.ztmf_logs.id
    prefix  = "rest-api-alb"
    enabled = true
  }
}

# TARGET GROUPS
resource "aws_lb_target_group" "ztmf_rest_api" {
  name        = "ztmf-rest-api"
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
resource "aws_lb_listener" "ztmf_rest_api" {
  load_balancer_arn = aws_lb.ztmf_rest_api.arn
  port              = "443"
  protocol          = "HTTPS"
  ssl_policy        = "ELBSecurityPolicy-TLS13-1-2-2021-06"
  certificate_arn   = data.aws_acm_certificate.ztmf.id

  default_action {
    type = "fixed-response"
    fixed_response {
      content_type = "application/json"
      message_body = "{\"error\": \"not found\"}"
      status_code  = "404"
    }
  }
}

# RULES
resource "aws_lb_listener_rule" "login" {
  listener_arn = aws_lb_listener.ztmf_rest_api.arn
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
      session_timeout            = 3600
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

  condition {
    http_header {
      http_header_name = "x-auth-token"
      values           = [data.aws_secretsmanager_secret_version.ztmf_x_auth_token_current.secret_string]
    }
  }
}

resource "aws_lb_listener_rule" "api" {
  listener_arn = aws_lb_listener.ztmf_rest_api.arn
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
        "/whoami"
      ]
    }
  }

  condition {
    http_header {
      http_header_name = "x-auth-token"
      values           = [data.aws_secretsmanager_secret_version.ztmf_x_auth_token_current.secret_string]
    }
  }
}
