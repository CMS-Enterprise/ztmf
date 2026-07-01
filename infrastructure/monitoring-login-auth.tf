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

# Client-event telemetry. The pre-auth login page beacons a login-lookup outage
# (GET /api/v1/auth/lookup failing: timeout/network/5xx/4xx/malformed) to the
# unauthenticated POST /api/v1/client-events, which writes one structured line to
# ztmf_api. This metric filter turns those lines into an outage count.
#
# BEST-EFFORT signal: the endpoint is unauthenticated, so this metric is
# spoofable (fake beacons inflate it) and suppressible (exhausting the in-app
# rate limiter masks a real outage). The internal ALB access logs remain the
# unforgeable source of record; this is a convenience layer on top. No dimensions
# by design - default_value and dimensions are mutually exclusive in the provider,
# and a per-reason/per-IP dimension would add custom-metric cost and
# re-identification risk. Total count is the signal we want.
resource "aws_cloudwatch_log_metric_filter" "ztmf_login_lookup_unavailable" {
  name           = "ztmf-login-lookup-unavailable-${var.environment}"
  log_group_name = aws_cloudwatch_log_group.ztmf_api.name
  # Exact-phrase match against the handler's log line (controller/clientevents.go
  # writes "client_event: lookup_unavailable reason=..."). The two are coupled: if
  # that log format changes, update this pattern or the metric goes silently quiet
  # (and, under treat_missing_data=notBreaching below, the alarm never fires).
  pattern = "\"client_event: lookup_unavailable\""

  metric_transformation {
    name          = "LookupUnavailable"
    namespace     = "ZTMF/Login"
    value         = "1"
    default_value = "0" # continuous zeros keep the alarm out of INSUFFICIENT_DATA
    unit          = "Count"
  }
}

# Alarm on a sustained burst of lookup outages, tuned so a lone transient timeout
# does not page: Sum >= 5 in a single 5-min window. Mirrors the ELBAuth alarm
# shape above and routes to the same ztmf-alarms topic. Reads the custom
# (dimensionless) metric the filter above publishes.
resource "aws_cloudwatch_metric_alarm" "ztmf_login_lookup_unavailable" {
  alarm_name          = "ztmf-login-lookup-unavailable-${var.environment}"
  comparison_operator = "GreaterThanOrEqualToThreshold"
  evaluation_periods  = "1"
  metric_name         = aws_cloudwatch_log_metric_filter.ztmf_login_lookup_unavailable.metric_transformation[0].name
  namespace           = aws_cloudwatch_log_metric_filter.ztmf_login_lookup_unavailable.metric_transformation[0].namespace
  period              = "300"
  statistic           = "Sum"
  threshold           = "5"
  treat_missing_data  = "notBreaching"
  alarm_description   = "Client-side login-lookup outages beaconed to /api/v1/client-events exceeded the threshold in a 5-min window. Best-effort/spoofable signal - the internal ALB access logs are the source of record. Check ztmf_api for 'client_event: lookup_unavailable' and the /api/v1/auth/lookup path health."
  alarm_actions       = [aws_sns_topic.ztmf_alarms.arn]
  ok_actions          = [aws_sns_topic.ztmf_alarms.arn]

  tags = {
    Name        = "ZTMF Login Lookup Unavailable Alarm"
    Environment = var.environment
  }
}
