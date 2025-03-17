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
