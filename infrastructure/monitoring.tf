# CloudWatch monitoring resources for ZTMF

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

# Dead Letter Queue for failed Lambda invocations
resource "aws_sqs_queue" "ztmf_sync_dlq" {
  name                      = "ztmf-data-sync-dlq-${var.environment}"
  message_retention_seconds = 1209600 # 14 days

  tags = {
    Name        = "ZTMF Data Sync DLQ"
    Environment = var.environment
  }
}