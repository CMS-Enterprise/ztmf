resource "aws_cloudfront_origin_access_control" "cloudfront_s3_oac" {
  name                              = "ZTMF CloudFront S3 OAC"
  description                       = "ZTMF CloudFront S3 OAC"
  origin_access_control_origin_type = "s3"
  signing_behavior                  = "always"
  signing_protocol                  = "sigv4"
}

resource "aws_cloudfront_response_headers_policy" "hsts_policy" {
  name    = "ZTMF-HSTS-Policy"
  comment = "HSTS policy for ZTMF"

  security_headers_config {
    strict_transport_security {
      access_control_max_age_sec = 31536000 // 1 year
      include_subdomains         = true
      preload                    = true
      override                   = true
    }

    frame_options {
      frame_option = "DENY"
      override     = true
    }

    content_security_policy {
      content_security_policy = "default-src 'self'; script-src 'self' https://${local.domain_name}; style-src 'self' https://${local.domain_name} 'unsafe-inline'; img-src 'self' https://${local.domain_name} data:; font-src 'self' https://${local.domain_name} data:; connect-src 'self' https://${local.domain_name}; media-src 'self' https://${local.domain_name}; object-src 'none'; base-uri 'self';"
      override                = true
    }

    referrer_policy {
      referrer_policy = "strict-origin-when-cross-origin"
      override        = true
    }

    content_type_options {
      override = true
    }
  }

  custom_headers_config {
    items {
      header   = "Permissions-Policy"
      override = true
      value    = "microphone=(), geolocation=(), accelerometer=(), ambient-light-sensor=(), autoplay=(), camera=(), magnetometer=(), midi=(), serial=(), usb=(), bluetooth=(), display-capture=()"
    }
  }
}

resource "aws_cloudfront_distribution" "ztmf" {
  aliases             = [local.domain_name]
  enabled             = true
  is_ipv6_enabled     = false
  comment             = "ZTMF Scoring"
  default_root_object = "index.html"
  // CMS provides a pre configured web acl, but it cant be tagged thus it can
  // only be found by looking to the stack outputs
  web_acl_id = data.aws_cloudformation_stack.web_acl.outputs["SamQuickACLEnforcingV2"]
  origin {
    origin_id                = "ztmf_web_assets"
    domain_name              = aws_s3_bucket.ztmf_web_assets.bucket_regional_domain_name
    origin_access_control_id = aws_cloudfront_origin_access_control.cloudfront_s3_oac.id
  }

  origin {
    origin_id   = "ztmf_rest_api"
    domain_name = aws_lb.ztmf_rest_api.dns_name
    custom_origin_config {
      http_port              = 80
      https_port             = 443
      origin_protocol_policy = "https-only"
      origin_ssl_protocols   = ["TLSv1.2"]
    }
    custom_header {
      name  = "x-auth-token"
      value = data.aws_secretsmanager_secret_version.ztmf_x_auth_token_current.secret_string
    }
  }

  default_cache_behavior {
    allowed_methods            = ["HEAD", "DELETE", "POST", "GET", "OPTIONS", "PUT", "PATCH"]
    cached_methods             = ["HEAD", "GET", "OPTIONS"]
    target_origin_id           = "ztmf_web_assets"
    response_headers_policy_id = aws_cloudfront_response_headers_policy.hsts_policy.id

    forwarded_values {
      query_string = false

      cookies {
        forward = "none"
      }
    }

    viewer_protocol_policy = "redirect-to-https"
    min_ttl                = 0
    default_ttl            = 3600
    max_ttl                = 86400
  }

  ordered_cache_behavior {
    path_pattern               = "/api/*"
    allowed_methods            = ["HEAD", "DELETE", "POST", "GET", "OPTIONS", "PUT", "PATCH"]
    cached_methods             = ["HEAD", "GET", "OPTIONS"]
    target_origin_id           = "ztmf_rest_api"
    response_headers_policy_id = aws_cloudfront_response_headers_policy.hsts_policy.id

    forwarded_values {
      query_string = true
      headers      = ["*"]
      cookies {
        forward = "all"
      }
    }

    min_ttl                = 0
    default_ttl            = 0
    max_ttl                = 0
    compress               = true
    viewer_protocol_policy = "redirect-to-https"
  }

  ordered_cache_behavior {
    path_pattern               = "/oauth2/*"
    allowed_methods            = ["HEAD", "DELETE", "POST", "GET", "OPTIONS", "PUT", "PATCH"]
    cached_methods             = ["HEAD", "GET", "OPTIONS"]
    target_origin_id           = "ztmf_rest_api"
    response_headers_policy_id = aws_cloudfront_response_headers_policy.hsts_policy.id

    forwarded_values {
      query_string = true
      headers      = ["*"]
      cookies {
        forward = "all"
      }
    }

    min_ttl                = 0
    default_ttl            = 0
    max_ttl                = 0
    compress               = true
    viewer_protocol_policy = "redirect-to-https"
  }

  ordered_cache_behavior {
    path_pattern               = "/login"
    allowed_methods            = ["HEAD", "DELETE", "POST", "GET", "OPTIONS", "PUT", "PATCH"]
    cached_methods             = ["HEAD", "GET", "OPTIONS"]
    target_origin_id           = "ztmf_rest_api"
    response_headers_policy_id = aws_cloudfront_response_headers_policy.hsts_policy.id

    forwarded_values {
      query_string = true
      headers      = ["*"]
      cookies {
        forward = "all"
      }
    }

    min_ttl                = 0
    default_ttl            = 0
    max_ttl                = 0
    compress               = true
    viewer_protocol_policy = "redirect-to-https"
  }

  restrictions {
    geo_restriction {
      restriction_type = "whitelist"
      locations        = ["US"]
    }
  }

  viewer_certificate {
    # cloudfront_default_certificate = true
    acm_certificate_arn = data.aws_acm_certificate.ztmf.id
    ssl_support_method  = "sni-only"
  }
}
