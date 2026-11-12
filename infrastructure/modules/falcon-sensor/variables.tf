variable "enabled" {
  description = "Add the sensor. When false, container_definitions is passed through untouched and no volume is returned, so the task definition does not change."
  type        = bool
}

variable "image" {
  description = "Falcon container sensor image the init container copies the sensor from, pinned by digest so a re-pushed tag cannot change the sensor files without a change here. The default is the 7.40.0-7903 release. The one pinned version for every task lives here; callers pass null to take it. Bump deliberately: the sensor is copied from it on every task start. To bump, read the new tag's digest from the registry or from a task's imageDigest and replace the whole value."
  type        = string
  default     = "artifactory.cloud.cms.gov/crowdstrike-remote/falcon-container/release/falcon-container@sha256:685810bb17b9f6c50ebdb7c56ce1108207180b647c104644c7a17ff88ac40705"
  nullable    = false
}

variable "falconctl_opts_arn" {
  description = "ARN of the SSM SecureString holding FALCONCTL_OPTS (--cid, --provisioning-token, --tags). Injected through each container's secrets, never as plain environment."
  type        = string
}

variable "container_definitions" {
  description = "The task's container definitions as JSON, exactly as they would be passed to aws_ecs_task_definition (jsonencode([...]))."
  type        = string

  validation {
    condition     = !contains([for c in jsondecode(var.container_definitions) : c.name], "crowdstrike-falcon-init-container")
    error_message = "crowdstrike-falcon-init-container is reserved for the sensor."
  }
}

variable "entrypoints" {
  description = "Original entrypoint and command for every container, keyed by container name. Set explicitly so the sensor never has to discover them from the image. entry_point is the single executable the wrapper starts; its arguments go in command (omit for none)."
  type = map(object({
    entry_point = list(string)
    command     = optional(list(string))
  }))
  default = {}

  validation {
    condition     = alltrue([for e in values(var.entrypoints) : length(e.entry_point) == 1])
    error_message = "Each entry_point must be exactly one executable; put its arguments in command."
  }
}

variable "init_memory" {
  description = "Hard memory limit (MiB) on the init container. It exits before the other containers start."
  type        = number
  default     = 512
}

variable "init_log_configuration" {
  description = "logConfiguration for the init container, so a failed sensor copy leaves logs. Null for none."
  type = object({
    logDriver = string
    options   = optional(map(string), {})
  })
  default = null
}
