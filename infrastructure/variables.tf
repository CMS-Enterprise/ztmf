variable "environment" {
  type = string
}

variable "domain_name_prefix" {
  type    = string
  default = ""
}

variable "ecs_service_task_count" {
  type    = number
  default = 1
}

variable "entra_enabled" {
  description = "Enable the Entra ID identity provider alongside Okta: adds the per-IdP ALB listener rules, flips /api/* off ALB OIDC to backend session validation, and injects the Entra + session env into the API task. Defaults to false so the secrets can be created and seeded (aws secretsmanager put-secret-value) before any auth wiring goes live. Flip to true only after ztmf_entra_oidc and ztmf_session_signing_key hold real values in the target account."
  type        = bool
  default     = false
}

variable "alarm_notification_email" {
  description = "Email subscribed to the ztmf-alarms SNS topic for login/OIDC alerting. Empty (default) creates the topic and alarms without a subscription so they can land before a destination is finalized; set per-environment in tfvars. For a Slack/PagerDuty endpoint, swap the subscription protocol in monitoring-login-auth.tf."
  type        = string
  default     = ""
}

variable "db_backup_retention_days" {
  description = "Aurora point-in-time-recovery retention window, in days. Prod runs at the Aurora maximum of 35 because the incident this addresses surfaced 35 days after the edit. Dev holds no data worth recovering at that horizon, so it runs shorter. Valid range is 1-35."
  type        = number
  default     = 35

  validation {
    condition     = var.db_backup_retention_days >= 1 && var.db_backup_retention_days <= 35
    error_message = "db_backup_retention_days must be between 1 and 35 (the Aurora maximum)."
  }
}

variable "db_dump_enabled" {
  description = "Enable the monthly EventBridge schedule that dumps the Aurora cluster to the ztmf-db-dumps bucket. The bucket, task definition and alarm are always created so the schedule can be flipped on without an infrastructure change; only the schedule itself is gated. Defaults to false so a new environment does not start writing backups before its ops image has been pushed."
  type        = bool
  default     = false
}

variable "kion_rotate_schedule_enabled" {
  description = "Enable the daily EventBridge schedule for ztmf-kion-key-rotate. Kion NAT allowlist is in place (CMS-Enterprise/ztmf-misc#174) and real rotation was validated end to end on 2026-04-22, so this defaults to true. Set to false only for temporary maintenance windows when rotation must be paused."
  type        = bool
  default     = true
}
