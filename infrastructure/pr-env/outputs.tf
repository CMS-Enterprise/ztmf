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

output "auth_provider" {
  value = var.auth_provider
}

output "cognito_users" {
  description = "Test users who can sign in, in cognito mode; empty otherwise."
  value       = [for u in aws_cognito_user.demo : u.username]
}

output "cognito_password_parameter" {
  description = "SSM SecureString holding the test users' shared password, in cognito mode."
  value       = one(aws_ssm_parameter.cognito_password[*].name)
}
