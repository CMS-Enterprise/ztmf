# CloudWatch and DLQ resources for ZTMF cert-rotation Lambda.

# Dedicated DLQ for cert-rotation failures. A message here means a rotation
# attempt failed hard enough to exhaust Lambda's own retries; treat as urgent
# because the operator-uploaded bundle may not have reached ACM.
resource "aws_sqs_queue" "ztmf_cert_rotation_dlq" {
  count                     = local.cert_rotation_enabled ? 1 : 0
  name                      = "ztmf-cert-rotation-dlq-${var.environment}"
  message_retention_seconds = 1209600 # 14 days
  sqs_managed_sse_enabled   = true

  tags = {
    Name        = "ZTMF Cert Rotation DLQ"
    Environment = var.environment
  }
}

# Alarm on any Lambda runtime error. alarm_actions is left empty because repo
# convention is for the Lambda itself to emit Slack notifications, not
# CloudWatch. Wire to SNS when repo-wide alarm routing lands.
resource "aws_cloudwatch_metric_alarm" "ztmf_cert_rotation_errors" {
  count               = local.cert_rotation_enabled ? 1 : 0
  alarm_name          = "ztmf-cert-rotation-errors-${var.environment}"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = "1"
  metric_name         = "Errors"
  namespace           = "AWS/Lambda"
  period              = "60"
  statistic           = "Sum"
  threshold           = "0"
  alarm_description   = "ZTMF cert-rotation Lambda recorded an error"
  alarm_actions       = [] # TODO: wire to SNS topic when repo-wide alarm routing lands

  dimensions = {
    FunctionName = aws_lambda_function.cert_rotation[0].function_name
  }

  tags = {
    Name        = "ZTMF Cert Rotation Error Alarm"
    Environment = var.environment
  }
}

# Alarm on any DLQ depth.
resource "aws_cloudwatch_metric_alarm" "ztmf_cert_rotation_dlq_depth" {
  count               = local.cert_rotation_enabled ? 1 : 0
  alarm_name          = "ztmf-cert-rotation-dlq-depth-${var.environment}"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = "1"
  metric_name         = "ApproximateNumberOfMessagesVisible"
  namespace           = "AWS/SQS"
  period              = "60"
  statistic           = "Maximum"
  threshold           = "0"
  alarm_description   = "ZTMF cert-rotation DLQ has unprocessed messages"
  alarm_actions       = []

  dimensions = {
    QueueName = aws_sqs_queue.ztmf_cert_rotation_dlq[0].name
  }

  tags = {
    Name        = "ZTMF Cert Rotation DLQ Depth Alarm"
    Environment = var.environment
  }
}
