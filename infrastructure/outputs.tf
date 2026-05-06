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

# CloudFront distribution domain - the CNAME target for the env's
# domain_name (dev.ztmf.cms.gov, impl.ztmf.cms.gov, ztmf.cms.gov).
# Hand this value to the DNS team to create the CNAME record.
output "cloudfront_distribution_domain" {
  description = "CloudFront distribution domain name; CNAME target for the env's public domain (dev.ztmf.cms.gov / impl.ztmf.cms.gov / ztmf.cms.gov)"
  value       = aws_cloudfront_distribution.ztmf.domain_name
  sensitive   = false
}

output "cloudfront_distribution_id" {
  description = "CloudFront distribution ID; populate the env's CLOUDFRONT_DISTRIBUTION_ID GitHub secret with this"
  value       = aws_cloudfront_distribution.ztmf.id
  sensitive   = false
}

# Internal ALB DNS - not normally needed for external DNS, but useful
# for debugging direct ALB access from inside the VPC.
output "alb_dns_name" {
  description = "Internal ALB DNS name (private; only resolvable from inside the VPC)"
  value       = aws_lb.ztmf_api.dns_name
  sensitive   = false
}

# Store test events as SSM parameters for team reference
resource "aws_ssm_parameter" "lambda_test_events" {
  for_each = var.environment == "dev" ? {
    "dry-run-single-table" = jsonencode({
      trigger_type = "manual"
      dry_run      = true
      tables       = ["users"]
      full_refresh = false
    })
    "dry-run-all-tables" = jsonencode({
      trigger_type = "manual"
      dry_run      = true
      tables       = []
      full_refresh = true
    })
    "real-test-single" = jsonencode({
      trigger_type = "manual"
      dry_run      = false
      tables       = ["users"]
      full_refresh = false
    })
    } : {
    "prod-dry-run-validation" = jsonencode({
      trigger_type = "manual"
      dry_run      = true
      tables       = ["users", "scores"]
      full_refresh = false
    })
    "prod-manual-full-sync" = jsonencode({
      trigger_type = "manual"
      dry_run      = false
      tables       = []
      full_refresh = true
    })
  }

  name  = "/ztmf/${var.environment}/lambda/test-events/${each.key}"
  type  = "String"
  value = each.value

  description = "Test event template for ZTMF Lambda data sync function"

  tags = {
    Name        = "ZTMF Lambda Test Event"
    Environment = var.environment
    TestEvent   = each.key
  }
}