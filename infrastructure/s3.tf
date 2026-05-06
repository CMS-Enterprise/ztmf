resource "aws_s3_bucket" "ztmf_web_assets" {
  bucket = local.domain_name
}

resource "aws_s3_bucket_policy" "ztmf_web_assets_access_from_vpc" {
  bucket = aws_s3_bucket.ztmf_web_assets.id
  policy = data.aws_iam_policy_document.allow_s3_access_from_cloudfront.json
}

data "aws_iam_policy_document" "allow_s3_access_from_cloudfront" {
  statement {
    principals {
      type        = "Service"
      identifiers = ["cloudfront.amazonaws.com"]
    }

    actions = [
      "s3:GetObject",
    ]

    resources = [
      "${aws_s3_bucket.ztmf_web_assets.arn}/*",
    ]

    condition {
      test     = "StringEquals"
      variable = "AWS:SourceArn"
      values   = [aws_cloudfront_distribution.ztmf.arn]
    }
  }

  statement {
    sid    = "AllowSSLRequestsOnly"
    effect = "Deny"

    principals {
      type        = "*"
      identifiers = ["*"]
    }

    actions = [
      "s3:*"
    ]

    resources = [
      aws_s3_bucket.ztmf_web_assets.arn,
      "${aws_s3_bucket.ztmf_web_assets.arn}/*",
    ]


    condition {
      test     = "Bool"
      variable = "aws:SecureTransport"
      values   = ["false"]
    }
  }
}

// Account-shared ALB access log bucket. Account ID-keyed name; impl skips
// creation since dev's bucket already exists in the same account.
resource "aws_s3_bucket" "ztmf_logs" {
  count  = local.manage_account_singletons ? 1 : 0
  bucket = "ztmf-logs-${local.account_id}-use1"
}

resource "aws_s3_bucket_server_side_encryption_configuration" "ztmf_logs" {
  count  = local.manage_account_singletons ? 1 : 0
  bucket = aws_s3_bucket.ztmf_logs[0].id
  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm = "AES256"
    }
  }
}

resource "aws_s3_bucket_policy" "ztmf_logs_access" {
  count  = local.manage_account_singletons ? 1 : 0
  bucket = aws_s3_bucket.ztmf_logs[0].id
  policy = data.aws_iam_policy_document.ztmf_logs_access.json
}

data "aws_iam_policy_document" "ztmf_logs_access" {
  statement {
    principals {
      type = "AWS"
      # this is the ID of the AWS-managed account for the load balancer
      # as found here https://docs.aws.amazon.com/elasticloadbalancing/latest/application/enable-access-logging.html#verify-bucket-permissions
      identifiers = ["arn:aws:iam::127311923021:root"]
    }

    actions = [
      "s3:PutObject",
    ]

    resources = [
      "arn:aws:s3:::ztmf-logs-${local.account_id}-use1/rest-api-alb/*"
    ]
  }

  statement {
    sid    = "AllowSSLRequestsOnly"
    effect = "Deny"

    principals {
      type        = "*"
      identifiers = ["*"]
    }

    actions = [
      "s3:*"
    ]

    resources = [
      "arn:aws:s3:::ztmf-logs-${local.account_id}-use1",
      "arn:aws:s3:::ztmf-logs-${local.account_id}-use1/*",
    ]


    condition {
      test     = "Bool"
      variable = "aws:SecureTransport"
      values   = ["false"]
    }
  }
}

# S3 bucket for Lambda deployment packages
resource "aws_s3_bucket" "lambda_deployments" {
  bucket = "ztmf-lambda-deployments-${var.environment}"

  tags = {
    Name        = "ZTMF Lambda Deployments"
    Environment = var.environment
    Purpose     = "Lambda deployment packages"
  }
}

# Block all public access to the Lambda deployment bucket
resource "aws_s3_bucket_public_access_block" "lambda_deployments" {
  bucket = aws_s3_bucket.lambda_deployments.id

  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

# S3 bucket versioning for Lambda deployments
resource "aws_s3_bucket_versioning" "lambda_deployments" {
  bucket = aws_s3_bucket.lambda_deployments.id
  versioning_configuration {
    status = "Enabled"
  }
}

# S3 bucket server-side encryption
resource "aws_s3_bucket_server_side_encryption_configuration" "lambda_deployments" {
  bucket = aws_s3_bucket.lambda_deployments.id

  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm = "AES256"
    }
  }
}

# S3 bucket for CFACTS CSV sync (incoming/ and processed/)
resource "aws_s3_bucket" "cfacts_sync" {
  bucket = "ztmf-cfacts-sync-${var.environment}"

  tags = {
    Name        = "ZTMF CFACTS Sync"
    Environment = var.environment
    Purpose     = "CFACTS CSV file uploads and archives"
  }
}

# Block all public access to CFACTS sync bucket
resource "aws_s3_bucket_public_access_block" "cfacts_sync" {
  bucket = aws_s3_bucket.cfacts_sync.id

  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

# Enable versioning for CFACTS sync bucket
resource "aws_s3_bucket_versioning" "cfacts_sync" {
  bucket = aws_s3_bucket.cfacts_sync.id
  versioning_configuration {
    status = "Enabled"
  }
}

# Server-side encryption for CFACTS sync bucket
resource "aws_s3_bucket_server_side_encryption_configuration" "cfacts_sync" {
  bucket = aws_s3_bucket.cfacts_sync.id

  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm = "AES256"
    }
  }
}

# Lifecycle rule: expire processed files after 90 days
resource "aws_s3_bucket_lifecycle_configuration" "cfacts_sync" {
  bucket = aws_s3_bucket.cfacts_sync.id

  rule {
    id     = "expire-processed-files"
    status = "Enabled"

    filter {
      prefix = "processed/"
    }

    expiration {
      days = 90
    }
  }
}
