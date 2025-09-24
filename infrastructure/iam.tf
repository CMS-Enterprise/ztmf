# IAM resources for ZTMF Lambda functions

# IAM role for Lambda execution
resource "aws_iam_role" "ztmf_sync_lambda" {
  name = "ztmf-data-sync-lambda-${var.environment}"

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
    Name        = "ZTMF Data Sync Lambda Role"
    Environment = var.environment
  }
}

# IAM policy for Lambda logging
resource "aws_iam_policy" "ztmf_sync_lambda_logging" {
  name        = "ztmf-data-sync-lambda-logging-${var.environment}"
  description = "IAM policy for logging from ZTMF Data Sync Lambda"

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

# IAM policy for Secrets Manager access
resource "aws_iam_policy" "ztmf_sync_lambda_secrets" {
  name        = "ztmf-data-sync-lambda-secrets-${var.environment}"
  description = "IAM policy for Secrets Manager access from ZTMF Data Sync Lambda"

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
          local.db_cred_secret,
          "arn:aws:secretsmanager:${data.aws_region.current.id}:*:secret:ztmf_snowflake_${var.environment}*",
          aws_secretsmanager_secret.ztmf_slack_webhook.arn
        ]
      }
    ]
  })
}

# IAM policy for VPC access (required for RDS access)
resource "aws_iam_policy" "ztmf_sync_lambda_vpc" {
  name        = "ztmf-data-sync-lambda-vpc-${var.environment}"
  description = "IAM policy for VPC access from ZTMF Data Sync Lambda"

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

# IAM policy for SQS Dead Letter Queue access
resource "aws_iam_policy" "ztmf_sync_lambda_sqs" {
  name        = "ztmf-data-sync-lambda-sqs-${var.environment}"
  description = "IAM policy for SQS Dead Letter Queue access from ZTMF Data Sync Lambda"

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "sqs:SendMessage"
        ]
        Resource = aws_sqs_queue.ztmf_sync_dlq.arn
      }
    ]
  })
}

# Attach logging policy to Lambda role
resource "aws_iam_role_policy_attachment" "ztmf_sync_lambda_logging" {
  role       = aws_iam_role.ztmf_sync_lambda.name
  policy_arn = aws_iam_policy.ztmf_sync_lambda_logging.arn
}

# Attach secrets policy to Lambda role
resource "aws_iam_role_policy_attachment" "ztmf_sync_lambda_secrets" {
  role       = aws_iam_role.ztmf_sync_lambda.name
  policy_arn = aws_iam_policy.ztmf_sync_lambda_secrets.arn
}

# Attach VPC policy to Lambda role
resource "aws_iam_role_policy_attachment" "ztmf_sync_lambda_vpc" {
  role       = aws_iam_role.ztmf_sync_lambda.name
  policy_arn = aws_iam_policy.ztmf_sync_lambda_vpc.arn
}

# Attach SQS policy to Lambda role
resource "aws_iam_role_policy_attachment" "ztmf_sync_lambda_sqs" {
  role       = aws_iam_role.ztmf_sync_lambda.name
  policy_arn = aws_iam_policy.ztmf_sync_lambda_sqs.arn
}