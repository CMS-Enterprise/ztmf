output "container_definitions" {
  description = "Container definitions JSON for aws_ecs_task_definition: the input unchanged when disabled, otherwise the init container followed by every container wrapped."
  value       = var.enabled ? jsonencode(concat([local.init_container], local.wrapped)) : var.container_definitions

  precondition {
    condition     = !var.enabled || toset(keys(var.entrypoints)) == toset([for c in local.containers : c.name])
    error_message = "entrypoints must name every container in container_definitions, and only those."
  }

  precondition {
    condition     = !contains([for c in local.containers : c.name], local.init_container_name)
    error_message = "crowdstrike-falcon-init-container is reserved for the sensor."
  }
}

output "volume_names" {
  description = "Task volumes to add (an empty bind-mount volume each). Empty when disabled."
  value       = var.enabled ? [local.volume_name] : []
}
