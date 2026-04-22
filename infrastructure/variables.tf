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
  description = "Enable the daily EventBridge schedule for ztmf-kion-key-rotate. Defaults to false because the Lambda's NAT egress IPs are not yet on the Kion tenant allowlist (see CMS-Enterprise/ztmf-misc#174). Flip to true after Kion confirms the allowlist change so scheduled rotations begin."
  type        = bool
  default     = false
}
