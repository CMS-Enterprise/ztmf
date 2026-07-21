# CloudWatch alerting for the score-rollover copy that runs when a new data
# call is created (backend copyPreviousScores). The copy is best-effort and
# never fails the create, so its only runtime signal is a distinctive log token,
# ROLLOVER_ANOMALY, emitted on any zero, partial, or errored rollover. This turns
# that token into a metric and alarm so an empty new cycle is detected rather
# than shipped silently. See ztmf#411.

# Count of ROLLOVER_ANOMALY lines in the API log group.
resource "aws_cloudwatch_log_metric_filter" "ztmf_api_rollover_anomaly" {
  name           = "ztmf-api-rollover-anomaly-${var.environment}"
  log_group_name = aws_cloudwatch_log_group.ztmf_api.name
  pattern        = "ROLLOVER_ANOMALY"

  metric_transformation {
    name          = "RolloverAnomaly"
    namespace     = "ZTMF/API"
    value         = "1"
    default_value = "0"
    unit          = "Count"
  }
}

# Alarm on any anomaly. The score-rollover copy is plain Go that only
# log.Printf's - it has no self-emitter - so, like the login/OIDC alarms, it
# routes to the shared ztmf_alarms SNS topic (defined in monitoring-login-auth.tf)
# rather than leaving alarm_actions empty (that convention is only for the kion /
# cert-rotation Lambdas, which emit Slack themselves).
resource "aws_cloudwatch_metric_alarm" "ztmf_api_rollover_anomaly" {
  alarm_name          = "ztmf-api-rollover-anomaly-${var.environment}"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = "1"
  metric_name         = "RolloverAnomaly"
  namespace           = "ZTMF/API"
  period              = "300"
  statistic           = "Sum"
  threshold           = "0"
  treat_missing_data  = "notBreaching"
  alarm_description   = "ZTMF data-call score rollover copied zero/partial rows or errored (ROLLOVER_ANOMALY)"
  alarm_actions       = [aws_sns_topic.ztmf_alarms.arn]
  ok_actions          = [aws_sns_topic.ztmf_alarms.arn]

  tags = {
    Name        = "ZTMF API Rollover Anomaly Alarm"
    Environment = var.environment
  }
}
