output "lambda_function_name" {
  value       = aws_lambda_function.rotation.function_name
  description = "Deployed Lambda function name."
}

output "lambda_function_arn" {
  value       = aws_lambda_function.rotation.arn
  description = "Deployed Lambda function ARN."
}

