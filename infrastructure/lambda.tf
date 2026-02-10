# ZTMF Data Sync Lambda Function
#
# This Lambda function synchronizes data from PostgreSQL to Snowflake
# - Runs in dry-run mode in dev environments
# - Performs real sync in production environments
# - Scheduled to run quarterly via EventBridge




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
      ENVIRONMENT            = var.environment
      DB_SECRET_ID           = local.db_cred_secret
      DB_ENDPOINT            = aws_rds_cluster.ztmf.endpoint
      DB_PORT                = "5432"
      DB_NAME                = "ztmf"
      SLACK_SECRET_ID        = aws_secretsmanager_secret.ztmf_slack_webhook.name
      SNOWFLAKE_TABLE_PREFIX = var.snowflake_table_prefix
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

  # Ensure IAM role and monitoring resources are ready
  depends_on = [
    aws_iam_role_policy_attachment.ztmf_sync_lambda_logging,
    aws_iam_role_policy_attachment.ztmf_sync_lambda_secrets,
    aws_iam_role_policy_attachment.ztmf_sync_lambda_vpc,
    aws_iam_role_policy_attachment.ztmf_sync_lambda_sqs,
    aws_cloudwatch_log_group.ztmf_sync_lambda,
  ]
}

# EventBridge rule for scheduled execution
resource "aws_cloudwatch_event_rule" "ztmf_sync_schedule" {
  name        = "ztmf-data-sync-schedule-${var.environment}"
  description = "Schedule for ZTMF data synchronization to Snowflake"

  # Different schedules per environment (6-field cron format: minutes hours day month day-of-week year)
  schedule_expression = var.environment == "prod" ? "cron(0 2 1 1,4,7,10 ? *)" : "cron(0 9 ? * MON *)"

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
    tables       = []    # Empty means sync all tables
    full_refresh = false # Use MERGE/upsert for better performance
    dry_run      = var.environment != "prod"
  })
}

# Lambda permission for EventBridge to invoke the function
resource "aws_lambda_permission" "eventbridge_invoke" {
  statement_id  = "AllowExecutionFromEventBridge"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.ztmf_sync.function_name
  principal     = "events.amazonaws.com"
  source_arn    = aws_cloudwatch_event_rule.ztmf_sync_schedule.arn
}