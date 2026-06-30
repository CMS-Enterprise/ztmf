# SNS topic + CloudWatch alarms for the login / OIDC path.
#
# This is the repo's first *wired* alarm route. Existing app alarms
# (monitoring-kion.tf, monitoring-cert-rotation.tf) leave alarm_actions empty
# because those Lambdas emit Slack themselves; the ALB login path has no such
# emitter, so a broken Entra/Okta handshake was silent until now. The topic is
# same-account and unencrypted, so same-account CloudWatch alarms publish under
# the default access policy (an explicit topic policy is only needed
# cross-account, per the SNS access-policy docs).
#
# SCOPE NOTE: these alarms catch ALB<->IdP *handshake* breakage (token/userinfo
# exchange and rejected callbacks). They do NOT catch the failure we hit on
# 2026-06-30 (AADSTS50105 user-not-assigned), because that is rejected at Entra
# before any callback and produces no ELBAuth* datapoint. Catching that class
# requires an end-to-end browser canary with a seeded test user, or exporting
# Entra sign-in logs and alerting on AADSTS spikes (tracked separately).

resource "aws_sns_topic" "ztmf_alarms" {
  name = "ztmf-alarms-${var.environment}"

  tags = {
    Name        = "ZTMF Alarms"
    Environment = var.environment
  }
}

# Optional subscription. Count-gated so the topic and alarms can land before a
# destination is finalized; set var.alarm_notification_email per env in tfvars.
# Email subscriptions require a one-time manual confirmation click.
resource "aws_sns_topic_subscription" "ztmf_alarms_email" {
  count     = var.alarm_notification_email != "" ? 1 : 0
  topic_arn = aws_sns_topic.ztmf_alarms.arn
  protocol  = "email"
  endpoint  = var.alarm_notification_email
}

# ELBAuthError: the ALB could not COMPLETE the OIDC exchange with the IdP
# (token/userinfo endpoint returned non-2XX or timed out) - bad client secret,
# IdP unreachable, or egress/DNS. >0 in a 5-min window is actionable.
resource "aws_cloudwatch_metric_alarm" "ztmf_login_elb_auth_error" {
  alarm_name          = "ztmf-login-elb-auth-error-${var.environment}"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = "1"
  metric_name         = "ELBAuthError"
  namespace           = "AWS/ApplicationELB"
  period              = "300"
  statistic           = "Sum"
  threshold           = "0"
  treat_missing_data  = "notBreaching"
  alarm_description   = "ALB authenticate-oidc could not complete the IdP token/userinfo exchange (Okta or Entra). Check IdP client secret, endpoints, and ALB egress."
  alarm_actions       = [aws_sns_topic.ztmf_alarms.arn]
  ok_actions          = [aws_sns_topic.ztmf_alarms.arn]

  dimensions = {
    LoadBalancer = aws_lb.ztmf_api.arn_suffix
  }

  tags = {
    Name        = "ZTMF Login ELB Auth Error Alarm"
    Environment = var.environment
  }
}

# ELBAuthFailure: the IdP response was REJECTED by the ALB (invalid id_token,
# issuer/audience mismatch, or missing/invalid authorization code - i.e. a
# redirect-URI or consent problem at the IdP). >0 in a 5-min window is
# actionable and tells Batch 2 exactly what to look for in the IdP console.
resource "aws_cloudwatch_metric_alarm" "ztmf_login_elb_auth_failure" {
  alarm_name          = "ztmf-login-elb-auth-failure-${var.environment}"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = "1"
  metric_name         = "ELBAuthFailure"
  namespace           = "AWS/ApplicationELB"
  period              = "300"
  statistic           = "Sum"
  threshold           = "0"
  treat_missing_data  = "notBreaching"
  alarm_description   = "ALB authenticate-oidc rejected the IdP response (invalid id_token, issuer/audience mismatch, or missing/invalid code). Check IdP app config: redirect URI, consent, issuer/audience."
  alarm_actions       = [aws_sns_topic.ztmf_alarms.arn]
  ok_actions          = [aws_sns_topic.ztmf_alarms.arn]

  dimensions = {
    LoadBalancer = aws_lb.ztmf_api.arn_suffix
  }

  tags = {
    Name        = "ZTMF Login ELB Auth Failure Alarm"
    Environment = var.environment
  }
}
