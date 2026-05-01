# ZTMF Certificate Rotation Lambda
#
# S3-triggered Lambda that validates and imports TLS certificates into ACM.
# Notifies via shared Slack webhook. IAM lives in iam-cert-rotation.tf; DLQ
# and alarms live in monitoring-cert-rotation.tf.

# ACM certificate ARN sourced from SSM Parameter Store rather than tfvars.
# Operator sets the per-account value once via:
#   aws ssm put-parameter --name /ztmf/<env>/cert-rotation/acm-arn \
#     --type String --value "arn:aws:acm:..."
# Keeps account IDs and ARNs out of the repo and lets each AWS account own
# its own value without committing tfvars per environment.
data "aws_ssm_parameter" "cert_rotation_acm_arn" {
  count = var.enable_cert_rotation_lambda ? 1 : 0
  name  = "/ztmf/${var.environment}/cert-rotation/acm-arn"
}

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
  # nonsensitive() unwraps the SSM data source's sensitive marker so the ARN
  # can flow into Lambda env vars and IAM policies. The value is an ACM ARN,
  # not secret material; CloudFront and ALB already publish the same ARN.
  cert_rotation_acm_certificate_arn = local.cert_rotation_enabled ? nonsensitive(data.aws_ssm_parameter.cert_rotation_acm_arn[0].value) : ""
  cert_rotation_domain              = trimspace(var.cert_rotation_domain)
  cert_rotation_env_prefixes_json = jsonencode({
    (local.cert_rotation_prefix) = {
      domain            = local.cert_rotation_domain
      acmCertificateArn = local.cert_rotation_acm_certificate_arn
      backupSecretArn   = local.cert_rotation_enabled ? aws_secretsmanager_secret.cert_rotation_backup[0].arn : ""
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

  # Source prefix is versioned, so DeleteObject writes a delete marker and
  # retains the previous object as a noncurrent version. Without this rule,
  # every rotated key.pem would linger indefinitely as a readable noncurrent
  # version. Expire noncurrent versions after one day; the Lambda's retry
  # window is measured in minutes, so one day is a wide operational margin
  # and a tight confidentiality bound.
  rule {
    id     = "expire-noncurrent-source-versions"
    status = "Enabled"

    filter {
      prefix = "${local.cert_rotation_prefix}/"
    }

    noncurrent_version_expiration {
      noncurrent_days = 1
    }

    abort_incomplete_multipart_upload {
      days_after_initiation = 1
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
# Lambda + S3 notifications
# =============================================================================

resource "aws_cloudwatch_log_group" "cert_rotation_lambda" {
  count             = local.cert_rotation_enabled ? 1 : 0
  name              = "/aws/lambda/ztmf-cert-rotation-${var.environment}"
  retention_in_days = 30

  tags = {
    Name        = "ZTMF Cert Rotation Lambda Logs"
    Environment = var.environment
    Function    = "ztmf-cert-rotation"
  }
}

resource "aws_lambda_function" "cert_rotation" {
  count         = local.cert_rotation_enabled ? 1 : 0
  function_name = "ztmf-cert-rotation-${var.environment}"
  role          = aws_iam_role.cert_rotation_lambda[0].arn
  handler       = "bootstrap"
  runtime       = "provided.al2"

  # Deployment package from S3 (uploaded by CI/CD).
  s3_bucket = aws_s3_bucket.lambda_deployments.bucket
  s3_key    = "cert-rotation-deployment-latest.zip"

  memory_size = 256
  timeout     = 60

  vpc_config {
    subnet_ids         = data.aws_subnets.private.ids
    security_group_ids = [aws_security_group.ztmf_sync_lambda.id]
  }

  environment {
    variables = {
      ENVIRONMENT       = var.environment
      CERT_BUCKET       = aws_s3_bucket.cert_rotation[0].bucket
      ENV_PREFIXES_JSON = local.cert_rotation_env_prefixes_json
      ARCHIVE_PREFIX    = "processed"
      DRY_RUN           = var.environment != "prod" ? "true" : "false"
      SLACK_SECRET_ID   = aws_secretsmanager_secret.ztmf_slack_webhook.name
    }
  }

  logging_config {
    log_format            = "JSON"
    application_log_level = "INFO"
    system_log_level      = "WARN"
    log_group             = aws_cloudwatch_log_group.cert_rotation_lambda[0].name
  }

  dead_letter_config {
    target_arn = aws_sqs_queue.ztmf_cert_rotation_dlq[0].arn
  }

  tracing_config {
    mode = "Active"
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
    aws_iam_role_policy_attachment.cert_rotation_lambda_vpc,
    aws_iam_role_policy_attachment.cert_rotation_lambda_sqs,
    aws_iam_role_policy_attachment.cert_rotation_lambda_xray,
    aws_cloudwatch_log_group.cert_rotation_lambda,
  ]

  lifecycle {
    # Restores the ARN-shape guard that previously lived on the deleted
    # cert_rotation_acm_certificate_arn variable. The value now comes from
    # SSM Parameter Store, so a typo there (wrong region, secrets ARN
    # pasted by mistake, stale ARN from a deleted cert) would otherwise
    # surface only at runtime as ACM ImportCertificate AccessDenied.
    precondition {
      condition     = can(regex("^arn:aws:acm:[a-z0-9-]+:[0-9]{12}:certificate/[0-9a-f-]+$", local.cert_rotation_acm_certificate_arn))
      error_message = "SSM /ztmf/${var.environment}/cert-rotation/acm-arn must hold a valid ACM certificate ARN of the form arn:aws:acm:<region>:<account>:certificate/<id>."
    }
  }
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

  validation {
    # Prevent collision with the hard-coded archive prefix. If the watched
    # prefix were "processed", the Lambda's own archive writes would match
    # the S3 notification filter and retrigger the Lambda on its own output.
    # Case-insensitive so a typo like "Processed" is still rejected at plan.
    condition     = lower(trim(var.cert_rotation_prefix, "/")) != "processed"
    error_message = "cert_rotation_prefix must not be \"processed\" (case-insensitive); that value is reserved for the archive destination."
  }
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

# ACM certificate ARN is no longer a tfvars variable; sourced from
# SSM Parameter Store /ztmf/<env>/cert-rotation/acm-arn (see data block
# at top of file).
