# IAM resources for the ZTMF cert-rotation Lambda.
#
# Least-privilege: the role can only write its own log group, read the shared
# Slack webhook secret, put to the per-env backup secret, import a specific
# ACM certificate, read/write the watched S3 bucket paths, send to its own
# DLQ, attach to the shared Lambda security group in the VPC, and emit X-Ray
# segments.

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
  description = "CloudWatch Logs access for the ZTMF cert-rotation Lambda"

  # Scoped to this Lambda's own log group. Two resource entries because
  # CreateLogGroup acts on the group ARN and PutLogEvents/CreateLogStream
  # act on the :* stream-pattern form.
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
        Resource = [
          "arn:aws:logs:${data.aws_region.current.id}:${local.account_id}:log-group:/aws/lambda/ztmf-cert-rotation-${var.environment}",
          "arn:aws:logs:${data.aws_region.current.id}:${local.account_id}:log-group:/aws/lambda/ztmf-cert-rotation-${var.environment}:*"
        ]
      }
    ]
  })
}

resource "aws_iam_policy" "cert_rotation_lambda_secrets" {
  count       = local.cert_rotation_enabled ? 1 : 0
  name        = "ztmf-cert-rotation-lambda-secrets-${var.environment}"
  description = "Secrets Manager access for the shared Slack webhook (read) and the cert-rotation backup secret (write)"

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid    = "SlackWebhookRead"
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
        # Write-only. The Lambda never reads the backup secret: it produces
        # the payload from the S3 bundle it just validated. Granting only
        # PutSecretValue keeps the role from being able to read back the
        # private key material it stored.
        Sid    = "CertRotationBackupWrite"
        Effect = "Allow"
        Action = [
          "secretsmanager:PutSecretValue"
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
  description = "ACM ImportCertificate permission scoped to the configured certificate ARN"

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
  description = "Read, write, delete, and list objects under the watched prefix and its processed archive"

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
        # CopyObject performs an internal source-existence check that
        # requires s3:ListBucket on the source bucket; without this grant
        # the archive copy fails with AccessDenied even though the role
        # already has GetObject on the source key. Scope the list to the
        # same two prefixes the Lambda reads and writes so the role
        # cannot enumerate the rest of the bucket.
        Sid    = "S3ListBucketForCopySourceCheck"
        Effect = "Allow"
        Action = ["s3:ListBucket"]
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

resource "aws_iam_policy" "cert_rotation_lambda_vpc" {
  count       = local.cert_rotation_enabled ? 1 : 0
  name        = "ztmf-cert-rotation-lambda-vpc-${var.environment}"
  description = "VPC ENI management for the ZTMF cert-rotation Lambda"

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "ec2:CreateNetworkInterface",
          "ec2:DescribeNetworkInterfaces",
          "ec2:DeleteNetworkInterface",
          "ec2:AttachNetworkInterface",
          "ec2:DetachNetworkInterface"
        ]
        Resource = "*"
      }
    ]
  })
}

resource "aws_iam_policy" "cert_rotation_lambda_sqs" {
  count       = local.cert_rotation_enabled ? 1 : 0
  name        = "ztmf-cert-rotation-lambda-sqs-${var.environment}"
  description = "SQS SendMessage access to the cert-rotation DLQ"

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "sqs:SendMessage"
        ]
        Resource = aws_sqs_queue.ztmf_cert_rotation_dlq[0].arn
      }
    ]
  })
}

resource "aws_iam_policy" "cert_rotation_lambda_xray" {
  count       = local.cert_rotation_enabled ? 1 : 0
  name        = "ztmf-cert-rotation-lambda-xray-${var.environment}"
  description = "X-Ray tracing for the ZTMF cert-rotation Lambda"

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "xray:PutTraceSegments",
          "xray:PutTelemetryRecords"
        ]
        Resource = "*"
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

resource "aws_iam_role_policy_attachment" "cert_rotation_lambda_vpc" {
  count      = local.cert_rotation_enabled ? 1 : 0
  role       = aws_iam_role.cert_rotation_lambda[0].name
  policy_arn = aws_iam_policy.cert_rotation_lambda_vpc[0].arn
}

resource "aws_iam_role_policy_attachment" "cert_rotation_lambda_sqs" {
  count      = local.cert_rotation_enabled ? 1 : 0
  role       = aws_iam_role.cert_rotation_lambda[0].name
  policy_arn = aws_iam_policy.cert_rotation_lambda_sqs[0].arn
}

resource "aws_iam_role_policy_attachment" "cert_rotation_lambda_xray" {
  count      = local.cert_rotation_enabled ? 1 : 0
  role       = aws_iam_role.cert_rotation_lambda[0].name
  policy_arn = aws_iam_policy.cert_rotation_lambda_xray[0].arn
}
