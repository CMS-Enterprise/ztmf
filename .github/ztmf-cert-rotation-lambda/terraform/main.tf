provider "aws" {
  region = var.aws_region
}

data "aws_caller_identity" "current" {}

locals {
  env_prefix = trim(var.env_prefix, "/")

  env_prefixes_json = jsonencode({
    (local.env_prefix) = {
      domain                = var.domain
      acmCertificateArn     = var.acm_certificate_arn
      backupSecretArn       = var.backup_secret_arn
      slackWebhookSecretArn = local.slack_webhook_secret_arn_effective
    }
  })
}

resource "aws_secretsmanager_secret" "slack_webhook_placeholder" {
  count = var.slack_webhook_secret_arn == null ? 1 : 0
  name  = "${var.name}-slack-webhook"
}

resource "aws_secretsmanager_secret_version" "slack_webhook_placeholder" {
  count         = var.slack_webhook_secret_arn == null ? 1 : 0
  secret_id     = aws_secretsmanager_secret.slack_webhook_placeholder[0].id
  secret_string = "DISABLED"
}

locals {
  slack_webhook_secret_arn_effective = (
    var.slack_webhook_secret_arn != null
    ? var.slack_webhook_secret_arn
    : aws_secretsmanager_secret.slack_webhook_placeholder[0].arn
  )
}

resource "aws_cloudwatch_log_group" "lambda" {
  name              = "/aws/lambda/${var.name}"
  retention_in_days = 30
}

data "aws_iam_policy_document" "assume_role" {
  statement {
    effect  = "Allow"
    actions = ["sts:AssumeRole"]
    principals {
      type        = "Service"
      identifiers = ["lambda.amazonaws.com"]
    }
  }
}

resource "aws_iam_role" "lambda" {
  name               = "${var.name}-role"
  assume_role_policy = data.aws_iam_policy_document.assume_role.json
}

data "aws_iam_policy_document" "lambda" {
  statement {
    sid     = "Logs"
    effect  = "Allow"
    actions = ["logs:CreateLogStream", "logs:PutLogEvents"]
    resources = [
      "${aws_cloudwatch_log_group.lambda.arn}:*"
    ]
  }

  statement {
    sid     = "S3ReadWriteRotationObjects"
    effect  = "Allow"
    actions = [
      "s3:GetObject",
      "s3:HeadObject",
      "s3:DeleteObject",
      "s3:CopyObject"
    ]
    resources = [
      "arn:aws:s3:::${var.cert_bucket_name}/${local.env_prefix}/*",
      "arn:aws:s3:::${var.cert_bucket_name}/processed/${local.env_prefix}/*"
    ]
  }

  statement {
    sid     = "S3ListBucketForPrefix"
    effect  = "Allow"
    actions = ["s3:ListBucket"]
    resources = [
      "arn:aws:s3:::${var.cert_bucket_name}"
    ]
    condition {
      test     = "StringLike"
      variable = "s3:prefix"
      values = [
        "${local.env_prefix}/*",
        "processed/${local.env_prefix}/*"
      ]
    }
  }

  statement {
    sid     = "ACMImportFixedArn"
    effect  = "Allow"
    actions = ["acm:ImportCertificate"]
    resources = [
      var.acm_certificate_arn
    ]
  }

  statement {
    sid     = "SecretsBackupWrite"
    effect  = "Allow"
    actions = ["secretsmanager:PutSecretValue"]
    resources = [
      var.backup_secret_arn
    ]
  }

  statement {
    sid     = "SlackWebhookRead"
    effect  = "Allow"
    actions = ["secretsmanager:GetSecretValue"]
    resources = [local.slack_webhook_secret_arn_effective]
  }
}

resource "aws_iam_policy" "lambda" {
  name   = "${var.name}-policy"
  policy = data.aws_iam_policy_document.lambda.json
}

resource "aws_iam_role_policy_attachment" "lambda" {
  role       = aws_iam_role.lambda.name
  policy_arn = aws_iam_policy.lambda.arn
}

resource "aws_lambda_function" "rotation" {
  function_name = var.name
  role          = aws_iam_role.lambda.arn

  runtime       = var.lambda_handler_runtime
  handler       = "bootstrap"
  architectures = var.lambda_architectures
  timeout       = 60
  memory_size   = 256

  filename         = var.lambda_zip_path
  source_code_hash = filebase64sha256(var.lambda_zip_path)

  environment {
    variables = {
      CERT_BUCKET       = var.cert_bucket_name
      ENV_PREFIXES_JSON = local.env_prefixes_json
      ARCHIVE_PREFIX    = "processed"
      DRY_RUN           = var.dry_run ? "true" : "false"
    }
  }

  depends_on = [aws_cloudwatch_log_group.lambda]
}

resource "aws_lambda_permission" "allow_s3" {
  statement_id  = "AllowExecutionFromS3"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.rotation.function_name
  principal     = "s3.amazonaws.com"
  source_arn    = "arn:aws:s3:::${var.cert_bucket_name}"
}

resource "aws_s3_bucket_notification" "bucket" {
  bucket = var.cert_bucket_name

  lambda_function {
    lambda_function_arn = aws_lambda_function.rotation.arn
    events              = ["s3:ObjectCreated:*"]
    filter_prefix       = "${local.env_prefix}/"
    filter_suffix       = ".pem"
  }

  depends_on = [aws_lambda_permission.allow_s3]
}

