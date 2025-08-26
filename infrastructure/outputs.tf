# Outputs for ZTMF Infrastructure

# Static IP for Lambda function (for Snowflake whitelisting)
output "lambda_static_ip" {
  description = "Static IP address for Lambda function outbound traffic (whitelist in Snowflake)"
  value       = aws_eip.lambda_nat.public_ip
  sensitive   = false
}

# Lambda function details
output "lambda_function_name" {
  description = "Name of the ZTMF data sync Lambda function"
  value       = aws_lambda_function.ztmf_sync.function_name
  sensitive   = false
}

output "lambda_function_arn" {
  description = "ARN of the ZTMF data sync Lambda function"
  value       = aws_lambda_function.ztmf_sync.arn
  sensitive   = false
}