# CMS already provides a VPC, we just need some endpoints in it

resource "aws_security_group" "ztmf_vpc_endpoints" {
  name        = "ztmf_vpc_endpoints"
  description = "Allow HTTP(S) traffic from private subnets"
  vpc_id      = data.aws_vpc.ztmf.id

  ingress {
    description = "HTTPS from private subnets"
    from_port   = 443
    to_port     = 443
    protocol    = "tcp"
    cidr_blocks = [for subnet in data.aws_subnet.private : subnet.cidr_block]
  }

  # ingress {
  #   description = "HTTP from private subnets"
  #   from_port   = 80
  #   to_port     = 80
  #   protocol    = "tcp"
  #   cidr_blocks = [for subnet in data.aws_subnet.private : subnet.cidr_block]
  # }
}

resource "aws_vpc_endpoint" "ztmf" {
  for_each            = toset(["ec2", "logs", "ecr.api", "ecr.dkr", "secretsmanager", "ssm", "ec2messages", "ssmmessages", "s3"])
  vpc_id              = data.aws_vpc.ztmf.id
  service_name        = "com.amazonaws.us-east-1.${each.value}"
  vpc_endpoint_type   = "Interface"
  subnet_ids          = data.aws_subnets.private.ids
  security_group_ids  = [aws_security_group.ztmf_vpc_endpoints.id]
  private_dns_enabled = true
  dns_options { private_dns_only_for_inbound_resolver_endpoint = false }
}

# Find existing NAT Gateway provided by CMS Cloud
data "aws_nat_gateways" "existing" {
  vpc_id = data.aws_vpc.ztmf.id
}

data "aws_nat_gateway" "existing" {
  count = length(data.aws_nat_gateways.existing.ids) > 0 ? 1 : 0
  id    = data.aws_nat_gateways.existing.ids[0]
}

# Find the Elastic IP associated with the existing NAT Gateway
data "aws_eip" "nat_gateway" {
  count = length(data.aws_nat_gateways.existing.ids) > 0 ? 1 : 0
  id    = data.aws_nat_gateway.existing[0].allocation_id
}
