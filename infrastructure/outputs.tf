# Outputs for ZTMF Infrastructure

# NAT egress IP for Lambdas in this account. Source-system allowlists
# (Kion, etc.) need this address to permit outbound calls from the VPC.
output "lambda_static_ip" {
  description = "Static IP for Lambda outbound traffic; whitelist in upstream source systems (Kion, etc.)"
  value       = length(data.aws_eip.nat_gateway) > 0 ? data.aws_eip.nat_gateway[0].public_ip : "No NAT Gateway found"
  sensitive   = false
}

output "nat_gateway_id" {
  description = "ID of the NAT Gateway used by Lambdas in this account"
  value       = length(data.aws_nat_gateway.existing) > 0 ? data.aws_nat_gateway.existing[0].id : "No NAT Gateway found"
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

