# CloudWatch and DLQ resources for ZTMF Kion key rotation Lambda.

# Dedicated DLQ for Kion rotation failures. Using a separate queue from the
# cfacts DLQ keeps the paging signal unambiguous: a message here means the
# Kion API key is at risk of expiring.
resource "aws_sqs_queue" "ztmf_kion_key_rotate_dlq" {
  name                      = "ztmf-kion-key-rotate-dlq-${var.environment}"
  message_retention_seconds = 1209600 # 14 days
  sqs_managed_sse_enabled   = true

  tags = {
    Name        = "ZTMF Kion Key Rotate DLQ"
    Environment = var.environment
  }
}

# CloudWatch log group for the rotation Lambda.
resource "aws_cloudwatch_log_group" "ztmf_kion_key_rotate" {
  name              = "/aws/lambda/ztmf-kion-key-rotate-${var.environment}"
  retention_in_days = 14

  tags = {
    Name        = "ZTMF Kion Key Rotate Lambda Logs"
    Environment = var.environment
    Function    = "ztmf-kion-key-rotate"
  }
}

# Alarm on any Lambda runtime error. Matches the shape used by cfacts alarms;
# alarm_actions is left empty because repo convention is for the Lambda itself
# to emit Slack notifications, not CloudWatch.
resource "aws_cloudwatch_metric_alarm" "ztmf_kion_key_rotate_errors" {
  alarm_name          = "ztmf-kion-key-rotate-errors-${var.environment}"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = "1"
  metric_name         = "Errors"
  namespace           = "AWS/Lambda"
  period              = "60"
  statistic           = "Sum"
  threshold           = "0"
  alarm_description   = "ZTMF Kion key rotation Lambda recorded an error"
  alarm_actions       = [] # TODO: wire to SNS topic when repo-wide alarm routing lands

  dimensions = {
    FunctionName = aws_lambda_function.ztmf_kion_key_rotate.function_name
  }

  tags = {
    Name        = "ZTMF Kion Key Rotate Error Alarm"
    Environment = var.environment
  }
}

# Alarm on any DLQ depth. A message here means a rotation attempt failed hard
# enough to exhaust in-Lambda retries; treat as urgent because Kion keys expire
# every 7 days.
resource "aws_cloudwatch_metric_alarm" "ztmf_kion_key_rotate_dlq_depth" {
  alarm_name          = "ztmf-kion-key-rotate-dlq-depth-${var.environment}"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = "1"
  metric_name         = "ApproximateNumberOfMessagesVisible"
  namespace           = "AWS/SQS"
  period              = "60"
  statistic           = "Maximum"
  threshold           = "0"
  alarm_description   = "ZTMF Kion key rotation DLQ has unprocessed messages"
  alarm_actions       = []

  dimensions = {
    QueueName = aws_sqs_queue.ztmf_kion_key_rotate_dlq.name
  }

  tags = {
    Name        = "ZTMF Kion Key Rotate DLQ Depth Alarm"
    Environment = var.environment
  }
}

# Alarm on staleness: the Lambda emits DaysSinceRotation at the end of every
# run. If the most recent value is >= 6 we are one day from Kion invalidating
# the key and downstream integrations breaking.
resource "aws_cloudwatch_metric_alarm" "ztmf_kion_days_since_rotation" {
  alarm_name          = "ztmf-kion-days-since-rotation-${var.environment}"
  comparison_operator = "GreaterThanOrEqualToThreshold"
  evaluation_periods  = "1"
  metric_name         = "DaysSinceRotation"
  namespace           = "ZTMF/Kion"
  period              = "86400" # 1 day
  statistic           = "Maximum"
  threshold           = "6"
  treat_missing_data  = "breaching"
  alarm_description   = "ZTMF Kion key has not been rotated in 6+ days; Kion keys expire in 7 days"
  alarm_actions       = []

  dimensions = {
    Environment = var.environment
  }

  tags = {
    Name        = "ZTMF Kion Days Since Rotation Alarm"
    Environment = var.environment
  }
}
