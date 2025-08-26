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

# Elastic IP for NAT Gateway (provides static outbound IP for Snowflake whitelisting)
resource "aws_eip" "lambda_nat" {
  domain = "vpc"

  tags = {
    Name        = "ZTMF Lambda NAT Gateway EIP"
    Environment = var.environment
    Purpose     = "Static IP for Snowflake whitelisting"
  }
}

# Get public subnets for NAT Gateway placement
data "aws_subnets" "public" {
  filter {
    name   = "vpc-id"
    values = [data.aws_vpc.ztmf.id]
  }

  filter {
    name   = "tag:use"
    values = ["public"]
  }
}

# NAT Gateway for Lambda outbound connectivity
resource "aws_nat_gateway" "lambda" {
  allocation_id = aws_eip.lambda_nat.id
  subnet_id     = data.aws_subnets.public.ids[0]  # Use first public subnet

  tags = {
    Name        = "ZTMF Lambda NAT Gateway"
    Environment = var.environment
    Purpose     = "Outbound connectivity for Lambda Snowflake sync"
  }

  depends_on = [aws_eip.lambda_nat]
}

# Route table for Lambda private subnets
resource "aws_route_table" "lambda_private" {
  vpc_id = data.aws_vpc.ztmf.id

  route {
    cidr_block     = "0.0.0.0/0"
    nat_gateway_id = aws_nat_gateway.lambda.id
  }

  tags = {
    Name        = "ZTMF Lambda Private Route Table"
    Environment = var.environment
    Purpose     = "Route Lambda traffic through NAT Gateway"
  }
}

# Associate route table with private subnets where Lambda runs
resource "aws_route_table_association" "lambda_private" {
  for_each = toset(data.aws_subnets.private.ids)

  subnet_id      = each.value
  route_table_id = aws_route_table.lambda_private.id
}
