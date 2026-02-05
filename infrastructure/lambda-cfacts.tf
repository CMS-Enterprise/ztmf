# ZTMF CFACTS Sync Lambda Functions
#
# Two Lambda functions for syncing CFACTS data INTO PostgreSQL cfacts_systems table:
# 1. Snowflake Lambda - Queries configured Snowflake view (CFACTS_SNOWFLAKE_VIEW env var), writes to PG
# 2. S3 CSV Lambda - Processes CSV uploads from S3 incoming/, writes to PG, archives to processed/


# =============================================================================
# Snowflake Lambda
# =============================================================================

resource "aws_lambda_function" "cfacts_snowflake_sync" {
  function_name = "ztmf-cfacts-snowflake-sync-${var.environment}"
  role          = aws_iam_role.cfacts_sync_lambda.arn
  handler       = "bootstrap"
  runtime       = "provided.al2"

  # Deployment package from S3 (uploaded by CI/CD)
  s3_bucket = aws_s3_bucket.lambda_deployments.bucket
  s3_key    = "cfacts-snowflake-deployment-latest.zip"

  # Lambda configuration
  memory_size = 1024 # 1GB for Snowflake + PostgreSQL operations
  timeout     = 900  # 15 minutes max

  # VPC configuration to access RDS
  vpc_config {
    subnet_ids         = data.aws_subnets.private.ids
    security_group_ids = [aws_security_group.ztmf_sync_lambda.id]
  }

  # Environment variables
  environment {
    variables = {
      ENVIRONMENT          = var.environment
      DB_SECRET_ID         = local.db_cred_secret
      DB_ENDPOINT          = aws_rds_cluster.ztmf.endpoint
      DB_PORT              = "5432"
      DB_NAME              = "ztmf"
      SLACK_SECRET_ID      = aws_secretsmanager_secret.ztmf_slack_webhook.name
      CFACTS_SNOWFLAKE_VIEW = var.cfacts_snowflake_view
    }
  }

  # Advanced logging configuration
  logging_config {
    log_format            = "JSON"
    application_log_level = "INFO"
    system_log_level      = "WARN"
    log_group             = aws_cloudwatch_log_group.cfacts_snowflake_lambda.name
  }

  # Dead letter queue for failed invocations
  dead_letter_config {
    target_arn = aws_sqs_queue.cfacts_sync_dlq.arn
  }

  # Enable X-Ray tracing
  tracing_config {
    mode = "Active"
  }

  tags = {
    Name        = "ZTMF CFACTS Snowflake Sync Lambda"
    Environment = var.environment
    Purpose     = "Snowflake to PostgreSQL CFACTS data synchronization"
  }

  depends_on = [
    aws_iam_role_policy_attachment.cfacts_sync_lambda_logging,
    aws_iam_role_policy_attachment.cfacts_sync_lambda_secrets,
    aws_iam_role_policy_attachment.cfacts_sync_lambda_vpc,
    aws_iam_role_policy_attachment.cfacts_sync_lambda_sqs,
    aws_cloudwatch_log_group.cfacts_snowflake_lambda,
  ]
}

# EventBridge rule - daily at 3AM UTC (initially disabled)
resource "aws_cloudwatch_event_rule" "cfacts_snowflake_schedule" {
  name        = "ztmf-cfacts-snowflake-schedule-${var.environment}"
  description = "Daily schedule for CFACTS Snowflake sync to PostgreSQL"
  state       = "DISABLED"

  schedule_expression = "cron(0 3 * * ? *)"

  tags = {
    Name        = "ZTMF CFACTS Snowflake Sync Schedule"
    Environment = var.environment
  }
}

# EventBridge target
resource "aws_cloudwatch_event_target" "cfacts_snowflake_lambda" {
  rule      = aws_cloudwatch_event_rule.cfacts_snowflake_schedule.name
  target_id = "CfactsSnowflakeSyncLambdaTarget"
  arn       = aws_lambda_function.cfacts_snowflake_sync.arn

  input = jsonencode({
    trigger_type = "scheduled"
    dry_run      = var.environment != "prod"
  })
}

# Lambda permission for EventBridge
resource "aws_lambda_permission" "cfacts_snowflake_eventbridge" {
  statement_id  = "AllowExecutionFromEventBridge"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.cfacts_snowflake_sync.function_name
  principal     = "events.amazonaws.com"
  source_arn    = aws_cloudwatch_event_rule.cfacts_snowflake_schedule.arn
}


# =============================================================================
# S3 CSV Lambda
# =============================================================================

resource "aws_lambda_function" "cfacts_s3_sync" {
  function_name = "ztmf-cfacts-s3-sync-${var.environment}"
  role          = aws_iam_role.cfacts_sync_lambda.arn
  handler       = "bootstrap"
  runtime       = "provided.al2"

  # Deployment package from S3 (uploaded by CI/CD)
  s3_bucket = aws_s3_bucket.lambda_deployments.bucket
  s3_key    = "cfacts-s3-deployment-latest.zip"

  # Lambda configuration - smaller than Snowflake (no external DB connection)
  memory_size = 512
  timeout     = 300 # 5 minutes

  # VPC configuration to access RDS
  vpc_config {
    subnet_ids         = data.aws_subnets.private.ids
    security_group_ids = [aws_security_group.ztmf_sync_lambda.id]
  }

  # Environment variables
  environment {
    variables = {
      ENVIRONMENT     = var.environment
      DB_SECRET_ID    = local.db_cred_secret
      DB_ENDPOINT     = aws_rds_cluster.ztmf.endpoint
      DB_PORT         = "5432"
      DB_NAME         = "ztmf"
      SLACK_SECRET_ID = aws_secretsmanager_secret.ztmf_slack_webhook.name
    }
  }

  # Advanced logging configuration
  logging_config {
    log_format            = "JSON"
    application_log_level = "INFO"
    system_log_level      = "WARN"
    log_group             = aws_cloudwatch_log_group.cfacts_s3_lambda.name
  }

  # Dead letter queue for failed invocations
  dead_letter_config {
    target_arn = aws_sqs_queue.cfacts_sync_dlq.arn
  }

  # Enable X-Ray tracing
  tracing_config {
    mode = "Active"
  }

  tags = {
    Name        = "ZTMF CFACTS S3 CSV Sync Lambda"
    Environment = var.environment
    Purpose     = "S3 CSV to PostgreSQL CFACTS data synchronization"
  }

  depends_on = [
    aws_iam_role_policy_attachment.cfacts_sync_lambda_logging,
    aws_iam_role_policy_attachment.cfacts_sync_lambda_secrets,
    aws_iam_role_policy_attachment.cfacts_sync_lambda_vpc,
    aws_iam_role_policy_attachment.cfacts_sync_lambda_sqs,
    aws_iam_role_policy_attachment.cfacts_sync_lambda_s3,
    aws_cloudwatch_log_group.cfacts_s3_lambda,
  ]
}

# S3 bucket notification for CSV uploads
resource "aws_s3_bucket_notification" "cfacts_csv_upload" {
  bucket = aws_s3_bucket.cfacts_sync.id

  lambda_function {
    lambda_function_arn = aws_lambda_function.cfacts_s3_sync.arn
    events              = ["s3:ObjectCreated:*"]
    filter_prefix       = "incoming/"
    filter_suffix       = ".csv"
  }

  depends_on = [aws_lambda_permission.cfacts_s3_bucket]
}

# Lambda permission for S3 to invoke the function
resource "aws_lambda_permission" "cfacts_s3_bucket" {
  statement_id  = "AllowExecutionFromS3"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.cfacts_s3_sync.function_name
  principal     = "s3.amazonaws.com"
  source_arn    = aws_s3_bucket.cfacts_sync.arn
}
