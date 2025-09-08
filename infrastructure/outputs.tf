# Outputs for ZTMF Infrastructure

# Static IP for Lambda function (for Snowflake whitelisting)
output "lambda_static_ip" {
  description = "Static IP address for Lambda function outbound traffic (whitelist in Snowflake)"
  value       = length(data.aws_eip.nat_gateway) > 0 ? data.aws_eip.nat_gateway[0].public_ip : "No NAT Gateway found"
  sensitive   = false
}

# NAT Gateway details (for reference)
output "nat_gateway_id" {
  description = "ID of the existing NAT Gateway used by Lambda"
  value       = length(data.aws_nat_gateway.existing) > 0 ? data.aws_nat_gateway.existing[0].id : "No NAT Gateway found"
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