variable "environment" {
  type = string
}

variable "domain_name_prefix" {
  type    = string
  default = ""
}

variable "vpc_environment" {
  description = "Which environment's VPC to use (e.g. impl shares dev's VPC). Empty string means use own environment."
  type        = string
  default     = ""
}

variable "ecs_service_task_count" {
  type    = number
  default = 1
}

variable "aurora_min_capacity" {
  description = "Aurora Serverless v2 minimum ACU (0.5 is the AWS minimum)"
  type        = number
  default     = 0.5
}

variable "aurora_max_capacity" {
  description = "Aurora Serverless v2 maximum ACU"
  type        = number
  default     = 1.0
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
