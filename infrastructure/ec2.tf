# this instance is only used as a bastion host to reach the database

resource "aws_instance" "bastion" {
  ami                         = "ami-0f403e3180720dd7e"
  iam_instance_profile        = aws_iam_instance_profile.ec2_bastion.name
  instance_type               = "t2.micro"
  associate_public_ip_address = false
  vpc_security_group_ids      = [aws_security_group.ztmf_bastion.id]
  subnet_id                   = data.aws_subnets.private.ids[0]
  tags = {
    Name = "ztmf-bastion"
  }
}

output "bastion_instance_id" {
  value = aws_instance.bastion.id
}

module "ec2_bastion" {
  name                = "ztmf_ec2_bastion"
  source              = "./modules/role"
  principal           = { Service = "ec2.amazonaws.com" }
  managed_policy_arns = ["arn:aws:iam::aws:policy/AmazonSSMManagedInstanceCore"]
}

resource "aws_iam_instance_profile" "ec2_bastion" {
  name = "ztmf_ec2_bastion"
  role = module.ec2_bastion.role_name
  path = "/delegatedadmin/adodeveloper/"
}

resource "aws_security_group" "ztmf_bastion" {
  name        = "ztmf_bastion"
  description = "bastion host"
  vpc_id      = data.aws_vpc.ztmf.id

  // only initiate connections to IPs in private subnets
  egress {
    description     = "HTTPS to private subnets" // access to session manager
    from_port       = 443
    to_port         = 443
    protocol        = "tcp"
    security_groups = [aws_security_group.ztmf_vpc_endpoints.id]
  }

  egress {
    description     = "PostgreSQL"
    from_port       = 5432
    to_port         = 5432
    protocol        = "tcp"
    security_groups = [aws_security_group.ztmf_db.id]
  }
}
