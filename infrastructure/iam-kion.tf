# IAM resources for the ZTMF Kion key rotation Lambda.
#
# Least-privilege: the role can only read and write its own environment's Kion
# secret, read the shared Slack webhook, emit logs, attach to the private VPC,
# send to its own DLQ, trace via X-Ray, and publish one custom CloudWatch
# metric. No wildcards on Secrets Manager.

resource "aws_iam_role" "ztmf_kion_key_rotate" {
  name = "ztmf-kion-key-rotate-${var.environment}"

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
    Name        = "ZTMF Kion Key Rotate Lambda Role"
    Environment = var.environment
  }
}

resource "aws_iam_policy" "ztmf_kion_key_rotate_logging" {
  name        = "ztmf-kion-key-rotate-logging-${var.environment}"
  description = "CloudWatch Logs access for the Kion key rotation Lambda"

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
          "arn:aws:logs:${data.aws_region.current.id}:${local.account_id}:log-group:/aws/lambda/ztmf-kion-key-rotate-${var.environment}",
          "arn:aws:logs:${data.aws_region.current.id}:${local.account_id}:log-group:/aws/lambda/ztmf-kion-key-rotate-${var.environment}:*"
        ]
      }
    ]
  })
}

resource "aws_iam_policy" "ztmf_kion_key_rotate_secrets" {
  name        = "ztmf-kion-key-rotate-secrets-${var.environment}"
  description = "Secrets Manager access scoped to the Kion secret (read+write) and the shared Slack webhook (read)"

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid    = "KionSecretReadWrite"
        Effect = "Allow"
        Action = [
          "secretsmanager:GetSecretValue",
          "secretsmanager:DescribeSecret",
          "secretsmanager:PutSecretValue"
        ]
        # Scope to this environment's Kion secret only. The trailing "-*"
        # matches the AWS-generated 6-character suffix Secrets Manager
        # appends to every secret ARN while excluding adjacent names like
        # "ztmf_kion_dev_backup" that the previous trailing "*" would have
        # allowed. Account segment is pinned to the current account ID;
        # the previous wildcard could have granted cross-account access
        # if resource-based policies were ever added elsewhere.
        Resource = [
          "arn:aws:secretsmanager:${data.aws_region.current.id}:${local.account_id}:secret:ztmf_kion_${var.environment}-*"
        ]
      },
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
      }
    ]
  })
}

resource "aws_iam_policy" "ztmf_kion_key_rotate_vpc" {
  name        = "ztmf-kion-key-rotate-vpc-${var.environment}"
  description = "VPC ENI management for the Kion key rotation Lambda"

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

resource "aws_iam_policy" "ztmf_kion_key_rotate_sqs" {
  name        = "ztmf-kion-key-rotate-sqs-${var.environment}"
  description = "SQS SendMessage access to the Kion rotation DLQ"

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "sqs:SendMessage"
        ]
        Resource = aws_sqs_queue.ztmf_kion_key_rotate_dlq.arn
      }
    ]
  })
}

resource "aws_iam_policy" "ztmf_kion_key_rotate_xray" {
  name        = "ztmf-kion-key-rotate-xray-${var.environment}"
  description = "X-Ray tracing for the Kion key rotation Lambda"

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

resource "aws_iam_policy" "ztmf_kion_key_rotate_metrics" {
  name        = "ztmf-kion-key-rotate-metrics-${var.environment}"
  description = "CloudWatch PutMetricData for the ZTMF/Kion namespace"

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect   = "Allow"
        Action   = ["cloudwatch:PutMetricData"]
        Resource = "*"
        Condition = {
          StringEquals = {
            "cloudwatch:namespace" = "ZTMF/Kion"
          }
        }
      }
    ]
  })
}

resource "aws_iam_role_policy_attachment" "ztmf_kion_key_rotate_logging" {
  role       = aws_iam_role.ztmf_kion_key_rotate.name
  policy_arn = aws_iam_policy.ztmf_kion_key_rotate_logging.arn
}

resource "aws_iam_role_policy_attachment" "ztmf_kion_key_rotate_secrets" {
  role       = aws_iam_role.ztmf_kion_key_rotate.name
  policy_arn = aws_iam_policy.ztmf_kion_key_rotate_secrets.arn
}

resource "aws_iam_role_policy_attachment" "ztmf_kion_key_rotate_vpc" {
  role       = aws_iam_role.ztmf_kion_key_rotate.name
  policy_arn = aws_iam_policy.ztmf_kion_key_rotate_vpc.arn
}

resource "aws_iam_role_policy_attachment" "ztmf_kion_key_rotate_sqs" {
  role       = aws_iam_role.ztmf_kion_key_rotate.name
  policy_arn = aws_iam_policy.ztmf_kion_key_rotate_sqs.arn
}

resource "aws_iam_role_policy_attachment" "ztmf_kion_key_rotate_xray" {
  role       = aws_iam_role.ztmf_kion_key_rotate.name
  policy_arn = aws_iam_policy.ztmf_kion_key_rotate_xray.arn
}

resource "aws_iam_role_policy_attachment" "ztmf_kion_key_rotate_metrics" {
  role       = aws_iam_role.ztmf_kion_key_rotate.name
  policy_arn = aws_iam_policy.ztmf_kion_key_rotate_metrics.arn
}
