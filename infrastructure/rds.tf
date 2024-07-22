resource "aws_db_subnet_group" "ztmf" {
  name       = "ztmf"
  subnet_ids = data.aws_subnets.private.ids
}

resource "aws_security_group" "ztmf_db" {
  name        = "ztmf_db"
  description = "Allow postgresql inbound traffic"
  vpc_id      = data.aws_vpc.ztmf.id

  ingress {
    description = "PostgreSQL from VPC private subnets"
    from_port   = 5432
    to_port     = 5432
    protocol    = "tcp"
    cidr_blocks = [for subnet in data.aws_subnet.private : subnet.cidr_block]
  }
}

resource "aws_rds_cluster" "ztmf" {
  cluster_identifier          = "ztmf"
  engine                      = "aurora-postgresql"
  engine_mode                 = "provisioned"
  engine_version              = "16.1"
  database_name               = "ztmf"
  db_subnet_group_name        = aws_db_subnet_group.ztmf.name
  master_username             = data.aws_secretsmanager_secret_version.ztmf_db_user_current.secret_string
  manage_master_user_password = true
  storage_encrypted           = true

  serverlessv2_scaling_configuration {
    max_capacity = 1.0
    min_capacity = 0.5
  }
  vpc_security_group_ids = [aws_security_group.ztmf_db.id]
}

resource "aws_rds_cluster_instance" "ztmf" {
  cluster_identifier   = aws_rds_cluster.ztmf.id
  ca_cert_identifier   = "rds-ca-ecc384-g1"
  db_subnet_group_name = aws_rds_cluster.ztmf.db_subnet_group_name
  instance_class       = "db.serverless"
  engine               = aws_rds_cluster.ztmf.engine
  engine_version       = aws_rds_cluster.ztmf.engine_version
}
