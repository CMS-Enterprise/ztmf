# ZTMF Certificate Rotation Lambda
#
# S3-triggered Lambda that validates and imports TLS certificates into ACM.

locals {
  cert_rotation_enabled = var.enable_cert_rotation_lambda
  # `cert_rotation_prefix` defaults to "" (not null), so `coalesce()` won't fall back.
  # Use environment as the default prefix unless explicitly overridden.
  cert_rotation_prefix = trim(
    trimspace(var.cert_rotation_prefix) != "" ? var.cert_rotation_prefix : var.environment,
    "/"
  )
  cert_rotation_bucket_name = (
    trimspace(var.cert_rotation_bucket_name) != ""
    ? trimspace(var.cert_rotation_bucket_name)
    : "ztmf-cert-rotation-${var.environment}"
  )
  cert_rotation_acm_certificate_arn = trimspace(var.cert_rotation_acm_certificate_arn)
  cert_rotation_domain              = trimspace(var.cert_rotation_domain)
  cert_rotation_env_prefixes_json = jsonencode({
    (local.cert_rotation_prefix) = {
      domain                = local.cert_rotation_domain
      acmCertificateArn     = local.cert_rotation_acm_certificate_arn
      backupSecretArn       = local.cert_rotation_enabled ? aws_secretsmanager_secret.cert_rotation_backup[0].arn : ""
      slackWebhookSecretArn = aws_secretsmanager_secret.ztmf_slack_webhook.arn
    }
  })
}

resource "aws_s3_bucket" "cert_rotation" {
  count  = local.cert_rotation_enabled ? 1 : 0
  bucket = local.cert_rotation_bucket_name

  tags = {
    Name        = "ZTMF Cert Rotation"
    Environment = var.environment
    Purpose     = "TLS cert uploads and archives"
  }
}

resource "aws_s3_bucket_public_access_block" "cert_rotation" {
  count  = local.cert_rotation_enabled ? 1 : 0
  bucket = aws_s3_bucket.cert_rotation[0].id

  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

resource "aws_s3_bucket_versioning" "cert_rotation" {
  count  = local.cert_rotation_enabled ? 1 : 0
  bucket = aws_s3_bucket.cert_rotation[0].id

  versioning_configuration {
    status = "Enabled"
  }
}

resource "aws_s3_bucket_server_side_encryption_configuration" "cert_rotation" {
  count  = local.cert_rotation_enabled ? 1 : 0
  bucket = aws_s3_bucket.cert_rotation[0].id

  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm = "AES256"
    }
  }
}

resource "aws_s3_bucket_lifecycle_configuration" "cert_rotation" {
  count  = local.cert_rotation_enabled ? 1 : 0
  bucket = aws_s3_bucket.cert_rotation[0].id

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

resource "aws_secretsmanager_secret" "cert_rotation_backup" {
  count = local.cert_rotation_enabled ? 1 : 0
  name  = "ztmf-cert-rotation-backup-${var.environment}"

  description = "Backups of TLS certificate bundles imported to ACM by cert-rotation Lambda"

  tags = {
    Name        = "ZTMF Cert Rotation Backup"
    Environment = var.environment
    Purpose     = "TLS cert rotation backup"
  }
}

# =============================================================================
# IAM
# =============================================================================

resource "aws_iam_role" "cert_rotation_lambda" {
  count = local.cert_rotation_enabled ? 1 : 0
  name  = "ztmf-cert-rotation-lambda-${var.environment}"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = "sts:AssumeRole"
        Effect = "Allow"
        Principal = {
          Service = "lambda.amazonaws.com"
        }
      }
    ]
  })

  tags = {
    Name        = "ZTMF Cert Rotation Lambda Role"
    Environment = var.environment
  }
}

resource "aws_iam_policy" "cert_rotation_lambda_logging" {
  count       = local.cert_rotation_enabled ? 1 : 0
  name        = "ztmf-cert-rotation-lambda-logging-${var.environment}"
  description = "IAM policy for logging from ZTMF cert-rotation Lambda"

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "logs:CreateLogGroup",
          "logs:CreateLogStream",
          "logs:PutLogEvents"
        ]
        Resource = ["arn:aws:logs:*:*:*"]
      }
    ]
  })
}

resource "aws_iam_policy" "cert_rotation_lambda_secrets" {
  count       = local.cert_rotation_enabled ? 1 : 0
  name        = "ztmf-cert-rotation-lambda-secrets-${var.environment}"
  description = "IAM policy for Secrets Manager access from ZTMF cert-rotation Lambda"

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "secretsmanager:GetSecretValue",
          "secretsmanager:DescribeSecret"
        ]
        Resource = [
          aws_secretsmanager_secret.ztmf_slack_webhook.arn
        ]
      },
      {
        Effect = "Allow"
        Action = [
          "secretsmanager:PutSecretValue",
          "secretsmanager:DescribeSecret"
        ]
        Resource = [
          aws_secretsmanager_secret.cert_rotation_backup[0].arn
        ]
      }
    ]
  })
}

resource "aws_iam_policy" "cert_rotation_lambda_acm" {
  count       = local.cert_rotation_enabled ? 1 : 0
  name        = "ztmf-cert-rotation-lambda-acm-${var.environment}"
  description = "IAM policy for ACM import from ZTMF cert-rotation Lambda"

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "acm:ImportCertificate"
        ]
        Resource = [
          local.cert_rotation_acm_certificate_arn
        ]
      }
    ]
  })
}

resource "aws_iam_policy" "cert_rotation_lambda_s3" {
  count       = local.cert_rotation_enabled ? 1 : 0
  name        = "ztmf-cert-rotation-lambda-s3-${var.environment}"
  description = "IAM policy for cert rotation S3 bucket access"

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid    = "S3ReadWriteRotationObjects"
        Effect = "Allow"
        Action = [
          "s3:GetObject",
          "s3:HeadObject",
          "s3:PutObject",
          "s3:DeleteObject"
        ]
        Resource = [
          "arn:aws:s3:::${local.cert_rotation_bucket_name}/${local.cert_rotation_prefix}/*",
          "arn:aws:s3:::${local.cert_rotation_bucket_name}/processed/${local.cert_rotation_prefix}/*"
        ]
      },
      {
        Sid    = "S3ListBucketForPrefix"
        Effect = "Allow"
        Action = [
          "s3:ListBucket"
        ]
        Resource = [
          "arn:aws:s3:::${local.cert_rotation_bucket_name}"
        ]
        Condition = {
          StringLike = {
            "s3:prefix" = [
              "${local.cert_rotation_prefix}/*",
              "processed/${local.cert_rotation_prefix}/*"
            ]
          }
        }
      }
    ]
  })
}

resource "aws_iam_role_policy_attachment" "cert_rotation_lambda_logging" {
  count      = local.cert_rotation_enabled ? 1 : 0
  role       = aws_iam_role.cert_rotation_lambda[0].name
  policy_arn = aws_iam_policy.cert_rotation_lambda_logging[0].arn
}

resource "aws_iam_role_policy_attachment" "cert_rotation_lambda_secrets" {
  count      = local.cert_rotation_enabled ? 1 : 0
  role       = aws_iam_role.cert_rotation_lambda[0].name
  policy_arn = aws_iam_policy.cert_rotation_lambda_secrets[0].arn
}

resource "aws_iam_role_policy_attachment" "cert_rotation_lambda_acm" {
  count      = local.cert_rotation_enabled ? 1 : 0
  role       = aws_iam_role.cert_rotation_lambda[0].name
  policy_arn = aws_iam_policy.cert_rotation_lambda_acm[0].arn
}

resource "aws_iam_role_policy_attachment" "cert_rotation_lambda_s3" {
  count      = local.cert_rotation_enabled ? 1 : 0
  role       = aws_iam_role.cert_rotation_lambda[0].name
  policy_arn = aws_iam_policy.cert_rotation_lambda_s3[0].arn
}

# =============================================================================
# Lambda + S3 notifications
# =============================================================================

resource "aws_cloudwatch_log_group" "cert_rotation_lambda" {
  count             = local.cert_rotation_enabled ? 1 : 0
  name              = "/aws/lambda/ztmf-cert-rotation-${var.environment}"
  retention_in_days = 30
}

resource "aws_lambda_function" "cert_rotation" {
  count         = local.cert_rotation_enabled ? 1 : 0
  function_name = "ztmf-cert-rotation-${var.environment}"
  role          = aws_iam_role.cert_rotation_lambda[0].arn
  handler       = "bootstrap"
  runtime       = "provided.al2"

  # Deployment package from S3 (uploaded by CI/CD)
  s3_bucket = aws_s3_bucket.lambda_deployments.bucket
  s3_key    = "cert-rotation-deployment-latest.zip"

  memory_size = 256
  timeout     = 60

  environment {
    variables = {
      CERT_BUCKET       = aws_s3_bucket.cert_rotation[0].bucket
      ENV_PREFIXES_JSON = local.cert_rotation_env_prefixes_json
      ARCHIVE_PREFIX    = "processed"
      DRY_RUN           = var.environment != "prod" ? "true" : "false"
    }
  }

  logging_config {
    log_format            = "JSON"
    application_log_level = "INFO"
    system_log_level      = "WARN"
    log_group             = aws_cloudwatch_log_group.cert_rotation_lambda[0].name
  }

  tags = {
    Name        = "ZTMF Cert Rotation Lambda"
    Environment = var.environment
    Purpose     = "TLS cert validation + ACM import"
  }

  depends_on = [
    aws_iam_role_policy_attachment.cert_rotation_lambda_logging,
    aws_iam_role_policy_attachment.cert_rotation_lambda_secrets,
    aws_iam_role_policy_attachment.cert_rotation_lambda_acm,
    aws_iam_role_policy_attachment.cert_rotation_lambda_s3,
    aws_cloudwatch_log_group.cert_rotation_lambda,
  ]
}

resource "aws_lambda_permission" "cert_rotation_allow_s3" {
  count         = local.cert_rotation_enabled ? 1 : 0
  statement_id  = "AllowExecutionFromS3"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.cert_rotation[0].function_name
  principal     = "s3.amazonaws.com"
  source_arn    = aws_s3_bucket.cert_rotation[0].arn
}

resource "aws_s3_bucket_notification" "cert_rotation_bucket" {
  count  = local.cert_rotation_enabled ? 1 : 0
  bucket = aws_s3_bucket.cert_rotation[0].id

  lambda_function {
    lambda_function_arn = aws_lambda_function.cert_rotation[0].arn
    events              = ["s3:ObjectCreated:*"]
    filter_prefix       = "${local.cert_rotation_prefix}/"
    filter_suffix       = ".pem"
  }

  depends_on = [aws_lambda_permission.cert_rotation_allow_s3]
}

# =============================================================================
# Variables (feature-scoped)
# =============================================================================

variable "enable_cert_rotation_lambda" {
  description = "Enable the S3-triggered TLS cert rotation Lambda resources"
  type        = bool
  default     = false
}

variable "cert_rotation_bucket_name" {
  description = "Optional override for the cert rotation S3 bucket name"
  type        = string
  default     = ""
}

variable "cert_rotation_prefix" {
  description = "S3 prefix under the cert bucket to watch (defaults to environment)"
  type        = string
  default     = ""
}

variable "cert_rotation_domain" {
  description = "Expected TLS domain name for the server certificate"
  type        = string
  default     = ""

  validation {
    condition     = var.enable_cert_rotation_lambda == false || trimspace(var.cert_rotation_domain) != ""
    error_message = "cert_rotation_domain must be set when enable_cert_rotation_lambda is true."
  }
}

variable "cert_rotation_acm_certificate_arn" {
  description = "ACM certificate ARN to re-import (overwrites the existing cert)"
  type        = string
  default     = ""

  validation {
    condition     = var.enable_cert_rotation_lambda == false || trimspace(var.cert_rotation_acm_certificate_arn) != ""
    error_message = "cert_rotation_acm_certificate_arn must be set when enable_cert_rotation_lambda is true."
  }
}

