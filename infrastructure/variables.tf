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
