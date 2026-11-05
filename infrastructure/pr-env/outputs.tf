output "url" {
  description = "Environment URL behind CMS login; post this on the PR."
  value       = local.url
}

output "env_name" {
  value = local.env_name
}

output "service_name" {
  value = aws_ecs_service.env.name
}

output "log_group" {
  value = aws_cloudwatch_log_group.env.name
}
