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

variable "cfacts_snowflake_view" {
  description = "Fully qualified Snowflake view for CFACTS system sync"
  type        = string
}

variable "snowflake_table_prefix" {
  description = "Prefix for Snowflake table names in data sync (e.g. ZTMF -> ZTMF_DATACALLS)"
  type        = string
  default     = "ZTMF"
}

variable "kion_rotate_schedule_enabled" {
  description = "Enable the daily EventBridge schedule for ztmf-kion-key-rotate. Kion NAT allowlist is in place (CMS-Enterprise/ztmf-misc#174) and real rotation was validated end to end on 2026-04-22, so this defaults to true. Set to false only for temporary maintenance windows when rotation must be paused."
  type        = bool
  default     = true
}
