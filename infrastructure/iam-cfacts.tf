# IAM resources for ZTMF CFACTS Sync Lambda functions
# Shared role for both Snowflake and S3 CSV Lambdas

# IAM role for CFACTS Lambda execution
resource "aws_iam_role" "cfacts_sync_lambda" {
  name = "ztmf-cfacts-sync-lambda-${var.environment}"

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
    Name        = "ZTMF CFACTS Sync Lambda Role"
    Environment = var.environment
  }
}

# IAM policy for Lambda logging
resource "aws_iam_policy" "cfacts_sync_lambda_logging" {
  name        = "ztmf-cfacts-sync-lambda-logging-${var.environment}"
  description = "IAM policy for logging from ZTMF CFACTS Sync Lambdas"

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
resource "aws_iam_policy" "cfacts_sync_lambda_secrets" {
  name        = "ztmf-cfacts-sync-lambda-secrets-${var.environment}"
  description = "IAM policy for Secrets Manager access from ZTMF CFACTS Sync Lambdas"

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
resource "aws_iam_policy" "cfacts_sync_lambda_vpc" {
  name        = "ztmf-cfacts-sync-lambda-vpc-${var.environment}"
  description = "IAM policy for VPC access from ZTMF CFACTS Sync Lambdas"

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
resource "aws_iam_policy" "cfacts_sync_lambda_sqs" {
  name        = "ztmf-cfacts-sync-lambda-sqs-${var.environment}"
  description = "IAM policy for SQS DLQ access from ZTMF CFACTS Sync Lambdas"

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "sqs:SendMessage"
        ]
        Resource = aws_sqs_queue.cfacts_sync_dlq.arn
      }
    ]
  })
}

# IAM policy for S3 CFACTS sync bucket access
resource "aws_iam_policy" "cfacts_sync_lambda_s3" {
  name        = "ztmf-cfacts-sync-lambda-s3-${var.environment}"
  description = "IAM policy for S3 CFACTS sync bucket access"

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "s3:GetObject",
          "s3:PutObject",
          "s3:DeleteObject",
          "s3:ListBucket"
        ]
        Resource = [
          aws_s3_bucket.cfacts_sync.arn,
          "${aws_s3_bucket.cfacts_sync.arn}/*"
        ]
      }
    ]
  })
}

# Attach policies to role
resource "aws_iam_role_policy_attachment" "cfacts_sync_lambda_logging" {
  role       = aws_iam_role.cfacts_sync_lambda.name
  policy_arn = aws_iam_policy.cfacts_sync_lambda_logging.arn
}

resource "aws_iam_role_policy_attachment" "cfacts_sync_lambda_secrets" {
  role       = aws_iam_role.cfacts_sync_lambda.name
  policy_arn = aws_iam_policy.cfacts_sync_lambda_secrets.arn
}

resource "aws_iam_role_policy_attachment" "cfacts_sync_lambda_vpc" {
  role       = aws_iam_role.cfacts_sync_lambda.name
  policy_arn = aws_iam_policy.cfacts_sync_lambda_vpc.arn
}

resource "aws_iam_role_policy_attachment" "cfacts_sync_lambda_sqs" {
  role       = aws_iam_role.cfacts_sync_lambda.name
  policy_arn = aws_iam_policy.cfacts_sync_lambda_sqs.arn
}

resource "aws_iam_role_policy_attachment" "cfacts_sync_lambda_s3" {
  role       = aws_iam_role.cfacts_sync_lambda.name
  policy_arn = aws_iam_policy.cfacts_sync_lambda_s3.arn
}

# IAM policy for X-Ray tracing
resource "aws_iam_policy" "cfacts_sync_lambda_xray" {
  name        = "ztmf-cfacts-sync-lambda-xray-${var.environment}"
  description = "IAM policy for X-Ray tracing from ZTMF CFACTS Sync Lambdas"

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

resource "aws_iam_role_policy_attachment" "cfacts_sync_lambda_xray" {
  role       = aws_iam_role.cfacts_sync_lambda.name
  policy_arn = aws_iam_policy.cfacts_sync_lambda_xray.arn
}
