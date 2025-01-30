resource "aws_security_group" "ztmf_alb" {
  name        = "ztmf"
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
  name                       = "ztmf-api"
  internal                   = true
  load_balancer_type         = "application"
  security_groups            = [aws_security_group.ztmf_alb.id]
  subnets                    = data.aws_subnets.private.ids
  enable_deletion_protection = true
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
resource "aws_lb_listener" "ztmf_api_https" {
  load_balancer_arn = aws_lb.ztmf_api.arn
  port              = "443"
  protocol          = "HTTPS"
  ssl_policy        = "ELBSecurityPolicy-TLS13-1-2-2021-06"
  certificate_arn   = data.aws_acm_certificate.ztmf.id

  default_action {
    type             = "forward"
    target_group_arn = aws_lb_target_group.ztmf_rest_api.arn
  }
}


# RULES
resource "aws_lb_listener_rule" "authz" {
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
