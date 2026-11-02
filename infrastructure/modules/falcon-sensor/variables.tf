variable "enabled" {
  description = "Add the sensor. When false, container_definitions is passed through untouched and no volume is returned, so the task definition does not change."
  type        = bool
}

variable "image" {
  description = "Falcon container sensor image the init container copies the sensor from."
  type        = string
}

variable "falconctl_opts_arn" {
  description = "ARN of the SSM SecureString holding FALCONCTL_OPTS (--cid, --provisioning-token, --tags). Injected through each container's secrets, never as plain environment."
  type        = string
}

variable "container_definitions" {
  description = "The task's container definitions as JSON, exactly as they would be passed to aws_ecs_task_definition (jsonencode([...]))."
  type        = string
}

variable "entrypoints" {
  description = "Original entrypoint and command for every container, keyed by container name. Set explicitly so the sensor never has to discover them from the image. entry_point is joined into one string after the wrapper; command is the container's command after wrapping (omit for none)."
  type = map(object({
    entry_point = list(string)
    command     = optional(list(string))
  }))
  default = {}
}

variable "init_memory" {
  description = "Hard memory limit (MiB) on the init container. It only runs before the other containers start, but counts against task memory while it does."
  type        = number
  default     = 512
}
