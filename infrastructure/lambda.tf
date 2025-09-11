# ZTMF Data Sync Lambda Function
#
# This Lambda function synchronizes data from PostgreSQL to Snowflake
# - Runs in dry-run mode in dev environments
# - Performs real sync in production environments
# - Scheduled to run quarterly via EventBridge

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

# Create a minimal placeholder Lambda deployment package
data "archive_file" "lambda_placeholder" {
  type        = "zip"
  output_path = "/tmp/lambda-placeholder.zip"

  source {
    content  = <<EOF
#!/bin/bash
echo "Placeholder Lambda function - will be replaced by CI/CD pipeline"
echo "This is just to allow Terraform to create the Lambda resource"
exit 0
EOF
    filename = "bootstrap"
  }
}

# Placeholder S3 object for Lambda deployment package
# This will be replaced by CI/CD pipeline
resource "aws_s3_object" "lambda_deployment_placeholder" {
  bucket = aws_s3_bucket.lambda_deployments.bucket
  key    = "lambda-deployment-latest.zip"
  source = data.archive_file.lambda_placeholder.output_path

  # Ensure bucket is ready
  depends_on = [
    aws_s3_bucket.lambda_deployments,
    aws_s3_bucket_server_side_encryption_configuration.lambda_deployments
  ]

  tags = {
    Name        = "ZTMF Lambda Deployment Placeholder"
    Environment = var.environment
    Purpose     = "Initial deployment package replaced by CI-CD"
  }
}

# CloudWatch Log Group for Lambda function
resource "aws_cloudwatch_log_group" "ztmf_sync_lambda" {
  name              = "/aws/lambda/ztmf-data-sync-${var.environment}"
  retention_in_days = 14

  tags = {
    Name        = "ZTMF Data Sync Lambda Logs"
    Environment = var.environment
    Function    = "ztmf-data-sync"
  }
}

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
          "arn:aws:secretsmanager:${data.aws_region.current.id}:*:secret:ztmf_snowflake_${var.environment}*"
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

# Security group for Lambda function
resource "aws_security_group" "ztmf_sync_lambda" {
  name        = "ztmf-data-sync-lambda-${var.environment}"
  description = "Security group for ZTMF Data Sync Lambda function"
  vpc_id      = data.aws_vpc.ztmf.id

  egress {
    description = "PostgreSQL to RDS"
    from_port   = 5432
    to_port     = 5432
    protocol    = "tcp"
    cidr_blocks = [for subnet in data.aws_subnet.private : subnet.cidr_block]
  }

  egress {
    description = "HTTPS outbound for Snowflake connectivity"
    from_port   = 443
    to_port     = 443
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  egress {
    description = "HTTP outbound for Snowflake OCSP certificate validation"
    from_port   = 80
    to_port     = 80
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  egress {
    description = "HTTP outbound for Snowflake OCSP cache server"
    from_port   = 80
    to_port     = 80
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  egress {
    description = "DNS resolution"
    from_port   = 53
    to_port     = 53
    protocol    = "udp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = {
    Name        = "ZTMF Data Sync Lambda SG"
    Environment = var.environment
  }
}

# Lambda function
resource "aws_lambda_function" "ztmf_sync" {
  function_name = "ztmf-data-sync-${var.environment}"
  role          = aws_iam_role.ztmf_sync_lambda.arn
  handler       = "bootstrap"
  runtime       = "provided.al2"

  # Deployment package from S3 (will be uploaded by CI/CD)
  s3_bucket = aws_s3_bucket.lambda_deployments.bucket
  s3_key    = "lambda-deployment-latest.zip"

  # Lambda configuration
  memory_size = 1024 # 1GB memory for database operations
  timeout     = 900  # 15 minutes (max for Lambda)

  # VPC configuration to access RDS
  vpc_config {
    subnet_ids         = data.aws_subnets.private.ids
    security_group_ids = [aws_security_group.ztmf_sync_lambda.id]
  }

  # Environment variables
  environment {
    variables = {
      ENVIRONMENT  = var.environment
      DB_SECRET_ID = local.db_cred_secret
      DB_ENDPOINT  = aws_rds_cluster.ztmf.endpoint
      DB_PORT      = "5432"
      DB_NAME      = "ztmf"
    }
  }

  # Advanced logging configuration
  logging_config {
    log_format            = "JSON"
    application_log_level = "INFO"
    system_log_level      = "WARN"
    log_group             = aws_cloudwatch_log_group.ztmf_sync_lambda.name
  }

  # Dead letter queue for failed invocations
  dead_letter_config {
    target_arn = aws_sqs_queue.ztmf_sync_dlq.arn
  }

  # Enable X-Ray tracing
  tracing_config {
    mode = "Active"
  }

  tags = {
    Name        = "ZTMF Data Sync Lambda"
    Environment = var.environment
    Purpose     = "PostgreSQL to Snowflake data synchronization"
  }

  # Ensure IAM role, log group, and deployment package are ready
  depends_on = [
    aws_iam_role_policy_attachment.ztmf_sync_lambda_logging,
    aws_iam_role_policy_attachment.ztmf_sync_lambda_secrets,
    aws_iam_role_policy_attachment.ztmf_sync_lambda_vpc,
    aws_iam_role_policy_attachment.ztmf_sync_lambda_sqs,
    aws_cloudwatch_log_group.ztmf_sync_lambda,
    aws_s3_object.lambda_deployment_placeholder
  ]
}

# Dead Letter Queue for failed Lambda invocations
resource "aws_sqs_queue" "ztmf_sync_dlq" {
  name                      = "ztmf-data-sync-dlq-${var.environment}"
  message_retention_seconds = 1209600 # 14 days

  tags = {
    Name        = "ZTMF Data Sync DLQ"
    Environment = var.environment
  }
}

# Lambda permission for EventBridge to invoke the function
resource "aws_lambda_permission" "eventbridge_invoke" {
  statement_id  = "AllowExecutionFromEventBridge"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.ztmf_sync.function_name
  principal     = "events.amazonaws.com"
  source_arn    = aws_cloudwatch_event_rule.ztmf_sync_schedule.arn
}

# EventBridge rule for scheduled execution
resource "aws_cloudwatch_event_rule" "ztmf_sync_schedule" {
  name        = "ztmf-data-sync-schedule-${var.environment}"
  description = "Schedule for ZTMF data synchronization to Snowflake"

  # Different schedules per environment
  schedule_expression = var.environment == "prod" ? "cron(0 2 1 */3 * ? *)" : "cron(0 9 ? * MON *)"

  tags = {
    Name        = "ZTMF Data Sync Schedule"
    Environment = var.environment
  }
}

# EventBridge target - Lambda function
resource "aws_cloudwatch_event_target" "ztmf_sync_lambda" {
  rule      = aws_cloudwatch_event_rule.ztmf_sync_schedule.name
  target_id = "ZTMFDataSyncLambdaTarget"
  arn       = aws_lambda_function.ztmf_sync.arn

  # Input for Lambda function (JSON event)
  input = jsonencode({
    trigger_type = "scheduled"
    tables       = [] # Empty means sync all tables
    full_refresh = true
    dry_run      = var.environment != "prod"
  })
}

# CloudWatch Metric Alarm for Lambda errors
resource "aws_cloudwatch_metric_alarm" "ztmf_sync_errors" {
  alarm_name          = "ztmf-data-sync-errors-${var.environment}"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = "1"
  metric_name         = "Errors"
  namespace           = "AWS/Lambda"
  period              = "60"
  statistic           = "Sum"
  threshold           = "0"
  alarm_description   = "This metric monitors ZTMF data sync Lambda function errors"
  alarm_actions       = [] # TODO: Add SNS topic for notifications

  dimensions = {
    FunctionName = aws_lambda_function.ztmf_sync.function_name
  }

  tags = {
    Name        = "ZTMF Data Sync Error Alarm"
    Environment = var.environment
  }
}

# CloudWatch Metric Alarm for Lambda duration
resource "aws_cloudwatch_metric_alarm" "ztmf_sync_duration" {
  alarm_name          = "ztmf-data-sync-duration-${var.environment}"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = "1"
  metric_name         = "Duration"
  namespace           = "AWS/Lambda"
  period              = "60"
  statistic           = "Average"
  threshold           = "600000" # 10 minutes in milliseconds
  alarm_description   = "This metric monitors ZTMF data sync Lambda function duration"
  alarm_actions       = [] # TODO: Add SNS topic for notifications

  dimensions = {
    FunctionName = aws_lambda_function.ztmf_sync.function_name
  }

  tags = {
    Name        = "ZTMF Data Sync Duration Alarm"
    Environment = var.environment
  }
}

# SSM Parameter to store Lambda function name for CI/CD reference
resource "aws_ssm_parameter" "ztmf_sync_lambda_name" {
  name  = "/ztmf/${var.environment}/lambda/data-sync/function-name"
  type  = "String"
  value = aws_lambda_function.ztmf_sync.function_name

  description = "ZTMF Data Sync Lambda function name for CI/CD reference"

  tags = {
    Name        = "ZTMF Data Sync Lambda Name"
    Environment = var.environment
  }
}

# SSM Parameter to store S3 bucket for Lambda deployments
resource "aws_ssm_parameter" "ztmf_sync_lambda_bucket" {
  name  = "/ztmf/${var.environment}/lambda/data-sync/deployment-bucket"
  type  = "String"
  value = aws_s3_bucket.lambda_deployments.bucket

  description = "ZTMF Data Sync Lambda deployment S3 bucket for CI/CD reference"

  tags = {
    Name        = "ZTMF Data Sync Lambda Bucket"
    Environment = var.environment
  }
}