resource "aws_security_group" "ztmf_alb" {
  name        = "ztmf"
  description = "Allow TLS inbound traffic"
  vpc_id      = data.aws_vpc.ztmf.id

  ingress {
    description = "HTTPS from VPC CIDR"
    from_port   = 443
    to_port     = 443
    protocol    = "tcp"
    cidr_blocks = [data.aws_vpc.ztmf.cidr_block]
  }

  // only initiate connections to IPs in private subnets
  egress {
    from_port   = 443
    to_port     = 443
    protocol    = "tcp"
    cidr_blocks = [for subnet in data.aws_subnet.private : subnet.cidr_block]
  }
}

resource "aws_lb" "ztmf" {
  name               = "ztmf-alb"
  internal           = true
  load_balancer_type = "application"
  security_groups    = [aws_security_group.ztmf_alb.id]
  subnets            = data.aws_subnets.private.ids

  enable_deletion_protection = false

  # access_logs {
  #   bucket  = aws_s3_bucket.lb_logs.id
  #   prefix  = "test-lb"
  #   enabled = true
  # }
}

# TARGET GROUPS

resource "aws_lb_target_group" "ztmf_api" {
  name        = "ztmf-api"
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

# default rule will forward to s3 bucket
resource "aws_lb_target_group" "s3" {
  name        = "s3"
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

resource "aws_lb_target_group_attachment" "s3" {
  for_each         = toset(flatten(data.aws_network_interface.s3.*.private_ips))
  target_group_arn = aws_lb_target_group.s3.arn
  target_id        = each.value
  port             = 443
}

# LISTENER

resource "aws_lb_listener" "ztmf_alb_https" {
  load_balancer_arn = aws_lb.ztmf.arn
  port              = "443"
  protocol          = "HTTPS"
  ssl_policy        = "ELBSecurityPolicy-TLS13-1-2-2021-06"
  certificate_arn   = data.aws_acm_certificate.ztmf.id

  default_action {
    type             = "forward"
    target_group_arn = aws_lb_target_group.s3.arn
    # fixed_response {
    #   content_type = "application/json"
    #   message_body = "{\"status\": \"ok\"}"
    #   status_code  = "200"
    # }
  }
}

resource "aws_lb_listener_rule" "graphql" {
  listener_arn = aws_lb_listener.ztmf_alb_https.arn
  priority     = 1

  action {
    type             = "forward"
    target_group_arn = aws_lb_target_group.ztmf_api.arn
  }

  condition {
    path_pattern {
      values = ["/graphql*"]
    }
  }
}

# redirect trailing slashes to include /index.html to play nice with S3
resource "aws_lb_listener_rule" "index" {
  listener_arn = aws_lb_listener.ztmf_alb_https.arn
  priority     = 2

  action {
    type = "redirect"
    redirect {
      path        = "/#{path}index.html"
      status_code = "HTTP_301"
    }
  }


  condition {
    path_pattern {
      values = ["*/"]
    }
  }
}

# RULES
