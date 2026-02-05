# CloudWatch monitoring resources for ZTMF CFACTS Sync Lambdas

# CloudWatch Log Group for Snowflake Lambda
resource "aws_cloudwatch_log_group" "cfacts_snowflake_lambda" {
  name              = "/aws/lambda/ztmf-cfacts-snowflake-sync-${var.environment}"
  retention_in_days = 14

  tags = {
    Name        = "ZTMF CFACTS Snowflake Sync Lambda Logs"
    Environment = var.environment
    Function    = "ztmf-cfacts-snowflake-sync"
  }
}

# CloudWatch Log Group for S3 Lambda
resource "aws_cloudwatch_log_group" "cfacts_s3_lambda" {
  name              = "/aws/lambda/ztmf-cfacts-s3-sync-${var.environment}"
  retention_in_days = 14

  tags = {
    Name        = "ZTMF CFACTS S3 Sync Lambda Logs"
    Environment = var.environment
    Function    = "ztmf-cfacts-s3-sync"
  }
}

# CloudWatch Metric Alarm for Snowflake Lambda errors
resource "aws_cloudwatch_metric_alarm" "cfacts_snowflake_errors" {
  alarm_name          = "ztmf-cfacts-snowflake-sync-errors-${var.environment}"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = "1"
  metric_name         = "Errors"
  namespace           = "AWS/Lambda"
  period              = "60"
  statistic           = "Sum"
  threshold           = "0"
  alarm_description   = "ZTMF CFACTS Snowflake sync Lambda function errors"
  alarm_actions       = [] # TODO: Add SNS topic for notifications

  dimensions = {
    FunctionName = aws_lambda_function.cfacts_snowflake_sync.function_name
  }

  tags = {
    Name        = "ZTMF CFACTS Snowflake Sync Error Alarm"
    Environment = var.environment
  }
}

# CloudWatch Metric Alarm for S3 Lambda errors
resource "aws_cloudwatch_metric_alarm" "cfacts_s3_errors" {
  alarm_name          = "ztmf-cfacts-s3-sync-errors-${var.environment}"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = "1"
  metric_name         = "Errors"
  namespace           = "AWS/Lambda"
  period              = "60"
  statistic           = "Sum"
  threshold           = "0"
  alarm_description   = "ZTMF CFACTS S3 sync Lambda function errors"
  alarm_actions       = [] # TODO: Add SNS topic for notifications

  dimensions = {
    FunctionName = aws_lambda_function.cfacts_s3_sync.function_name
  }

  tags = {
    Name        = "ZTMF CFACTS S3 Sync Error Alarm"
    Environment = var.environment
  }
}

# Shared Dead Letter Queue for CFACTS sync Lambda failures
resource "aws_sqs_queue" "cfacts_sync_dlq" {
  name                      = "ztmf-cfacts-sync-dlq-${var.environment}"
  message_retention_seconds = 1209600 # 14 days

  tags = {
    Name        = "ZTMF CFACTS Sync DLQ"
    Environment = var.environment
  }
}
