# ZTMF Kion API Key Rotation Lambda
#
# Daily Lambda that reads the current Kion App API key from ztmf_kion_${env},
# exchanges it for a fresh key via the Kion rotation endpoint, and writes the
# new value back into the same secret. Idempotent: the Go code skips rotation
# if the secret was updated in the last ROTATE_AFTER_DAYS days.

resource "aws_lambda_function" "ztmf_kion_key_rotate" {
  function_name = "ztmf-kion-key-rotate-${var.environment}"
  role          = aws_iam_role.ztmf_kion_key_rotate.arn
  handler       = "bootstrap"
  runtime       = "provided.al2023"
  architectures = ["arm64"]

  s3_bucket = aws_s3_bucket.lambda_deployments.bucket
  s3_key    = "kion-key-rotate-deployment-latest.zip"

  # Small workload: one secret read, one HTTP call, one secret write.
  memory_size = 256
  timeout     = 120

  vpc_config {
    subnet_ids         = data.aws_subnets.private.ids
    security_group_ids = [aws_security_group.ztmf_sync_lambda.id]
  }

  environment {
    variables = {
      ENVIRONMENT       = var.environment
      KION_SECRET_ID    = "ztmf_kion_${var.environment}"
      SLACK_SECRET_ID   = aws_secretsmanager_secret.ztmf_slack_webhook.name
      ROTATE_AFTER_DAYS = "4"
    }
  }

  logging_config {
    log_format            = "JSON"
    application_log_level = "INFO"
    system_log_level      = "WARN"
    log_group             = aws_cloudwatch_log_group.ztmf_kion_key_rotate.name
  }

  dead_letter_config {
    target_arn = aws_sqs_queue.ztmf_kion_key_rotate_dlq.arn
  }

  tracing_config {
    mode = "Active"
  }

  tags = {
    Name        = "ZTMF Kion Key Rotate Lambda"
    Environment = var.environment
    Purpose     = "Daily rotation of Kion App API key in AWS Secrets Manager"
  }

  depends_on = [
    aws_iam_role_policy_attachment.ztmf_kion_key_rotate_logging,
    aws_iam_role_policy_attachment.ztmf_kion_key_rotate_secrets,
    aws_iam_role_policy_attachment.ztmf_kion_key_rotate_vpc,
    aws_iam_role_policy_attachment.ztmf_kion_key_rotate_sqs,
    aws_iam_role_policy_attachment.ztmf_kion_key_rotate_xray,
    aws_iam_role_policy_attachment.ztmf_kion_key_rotate_metrics,
    aws_cloudwatch_log_group.ztmf_kion_key_rotate,
  ]
}

# Daily schedule at 06:00 UTC. The Lambda's idempotency check means firing
# every day is safe; it only calls Kion once every ROTATE_AFTER_DAYS days.
#
# Gated by var.kion_rotate_schedule_enabled so the rule is created in a
# DISABLED state until the Lambda's NAT egress IPs are allowlisted on the
# Kion tenant (tracked in CMS-Enterprise/ztmf-misc#174). With the schedule
# disabled the Lambda, secret, DLQ, and alarms all exist and can be
# exercised via manual `aws lambda invoke`; flipping the variable to true
# and re-applying terraform is the only step needed to go live.
resource "aws_cloudwatch_event_rule" "ztmf_kion_key_rotate_schedule" {
  name                = "ztmf-kion-key-rotate-schedule-${var.environment}"
  description         = "Daily trigger for the ZTMF Kion key rotation Lambda"
  state               = var.kion_rotate_schedule_enabled ? "ENABLED" : "DISABLED"
  schedule_expression = "cron(0 6 * * ? *)"

  tags = {
    Name        = "ZTMF Kion Key Rotate Schedule"
    Environment = var.environment
  }
}

resource "aws_cloudwatch_event_target" "ztmf_kion_key_rotate_target" {
  rule      = aws_cloudwatch_event_rule.ztmf_kion_key_rotate_schedule.name
  target_id = "ZtmfKionKeyRotateLambdaTarget"
  arn       = aws_lambda_function.ztmf_kion_key_rotate.arn

  # dev runs in dry-run mode so scheduled invocations exercise the wiring
  # without consuming a fresh Kion key every day.
  input = jsonencode({
    trigger_type = "scheduled"
    dry_run      = var.environment != "prod"
    force        = false
  })
}

resource "aws_lambda_permission" "ztmf_kion_key_rotate_eventbridge" {
  statement_id  = "AllowExecutionFromEventBridge"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.ztmf_kion_key_rotate.function_name
  principal     = "events.amazonaws.com"
  source_arn    = aws_cloudwatch_event_rule.ztmf_kion_key_rotate_schedule.arn
}
