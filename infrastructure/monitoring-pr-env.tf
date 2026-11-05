# Fleet alarm for per-PR environments (ztmf-misc#344, epic ztmf-misc#321).
#
# Teardown on close is the primary cleanup and the nightly reaper
# (.github/workflows/pr-env-reaper.yml) is the backstop. This alarm is the
# layer that does not depend on any GitHub workflow running: it counts running
# tasks in the cluster outside the dev API service, which is exactly the
# pr-env fleet (one task per environment), and pages when it stays above the
# concurrent-environment cap, through the ztmf-alarms topic and its shared
# inbox. It turns the cap from a policy into a number something enforces.
#
# A Budgets alert on the ztmf-pr-* cost-allocation tags was in the ticket's
# scope but is not possible from this linked account: tag activation is
# payer-account only ("Linked account doesn't have access to cost allocation
# tags"), and a budget cannot filter on an inactive tag.

# Container Insights publishes RunningTaskCount per service every minute, so a
# 60-second period sums one sample per service. Ten consecutive minutes above
# the cap absorbs the brief second task of a rolling redeploy on push.
resource "aws_cloudwatch_metric_alarm" "ztmf_pr_env_task_count" {
  count               = var.pr_env_enabled ? 1 : 0
  alarm_name          = "ztmf-pr-env-task-count-${var.environment}"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = 10
  datapoints_to_alarm = 10
  threshold           = var.pr_env_max_environments
  treat_missing_data  = "notBreaching"
  alarm_description   = "More than ${var.pr_env_max_environments} per-PR environment tasks have run for 10 minutes. Check the pr-env reaper's tracking issue in ztmf-misc and the pr/* keys in the dev state bucket."
  alarm_actions       = [aws_sns_topic.ztmf_alarms.arn]
  ok_actions          = [aws_sns_topic.ztmf_alarms.arn]

  metric_query {
    id          = "fleet"
    label       = "Running pr-env tasks"
    period      = 60
    return_data = true
    expression  = "SELECT SUM(RunningTaskCount) FROM SCHEMA(\"ECS/ContainerInsights\", ClusterName, ServiceName) WHERE ClusterName = '${aws_ecs_cluster.ztmf.name}' AND ServiceName != '${aws_ecs_service.ztmf_api.name}'"
  }

  tags = {
    Name        = "ZTMF PR Environment Task Count Alarm"
    Environment = var.environment
  }
}
