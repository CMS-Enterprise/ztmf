resource "aws_db_subnet_group" "ztmf" {
  name       = local.ztmf_name
  subnet_ids = data.aws_subnets.private.ids
}

resource "aws_security_group" "ztmf_db" {
  name        = local.ztmf_db_sg_name
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
  cluster_identifier          = local.ztmf_name
  engine                      = "aurora-postgresql"
  engine_mode                 = "provisioned"
  engine_version              = "16.11"
  database_name               = "ztmf"
  db_subnet_group_name        = aws_db_subnet_group.ztmf.name
  master_username             = data.aws_secretsmanager_secret_version.ztmf_db_user_current.secret_string
  manage_master_user_password = true
  storage_encrypted           = true

  # Point-in-time recovery. Prod runs at the Aurora maximum of 35 days; dev
  # stays at 1 because it holds no data worth recovering at that horizon. See
  # var.db_backup_retention_days.
  #
  # In prod this is deliberately NOT lowered between data calls. Aurora meters
  # backup storage on change volume rather than volume size, so a quiet month
  # costs near nothing anyway, and the quiet months are exactly when an
  # unnoticed edit is most likely to be sitting inside the PITR window.
  #
  # 35 days is the Aurora ceiling, not a sufficient answer on its own. The
  # incident that prompted this surfaced 35 days after the edit, so PITR alone
  # would have covered it with zero margin. Late-discovery coverage comes from
  # the AWS Backup enrollment below and the monthly dumps in backup-dumps.tf.
  backup_retention_period = var.db_backup_retention_days

  # Pinned rather than left to the AWS-assigned random windows (prod was
  # 06:19-06:49 / sun:07:05-sun:07:35, dev 08:37-09:07 / tue:10:22-tue:10:52).
  # Pinning keeps the two envs identical and stops a silent out-of-band change
  # from going unnoticed. Both are UTC; the backup window must not overlap the
  # maintenance window.
  preferred_backup_window      = "06:00-06:30"
  preferred_maintenance_window = "sun:07:00-sun:07:30"

  # Snapshots inherit cluster tags, so a restored cluster keeps its provenance.
  copy_tags_to_snapshot = true

  # This cluster is the system of record for every data call response. An
  # accidental destroy is not recoverable from anything short of a snapshot
  # restore, so require an explicit console/CLI removal of the flag first.
  deletion_protection = true

  # Pinned to the value already in use so a manual parameter-group swap shows
  # up as drift instead of silently persisting.
  db_cluster_parameter_group_name = "default.aurora-postgresql16"

  # AWS_Backup enrolls the cluster in the CMS OIT `Daily15_Weekly90` plan,
  # which selects purely on this tag (StringEquals aws:ResourceTag/AWS_Backup
  # = d15_w90) and holds 15 daily plus weekly recovery points for 95 days. The
  # plans already run daily in this account against zero tagged resources, so
  # enrollment is one tag rather than a new backup stack.
  #
  # Do NOT use the d15_w90_m365_y730 plan: its yearly rule requests 730-day
  # retention against a vault capped at MaxRetentionDays 720, so those jobs
  # fail. Use d15_w90 or d15_w90_m365.
  #
  # Do NOT copy this tag onto aws_rds_cluster_instance below. AWS Backup
  # protects Aurora at the cluster level; a tagged serverless instance is
  # selected as a separate RDS resource and its jobs fail.
  tags = {
    Name        = local.ztmf_name
    Environment = var.environment
    AWS_Backup  = "d15_w90"
  }

  serverlessv2_scaling_configuration {
    max_capacity = 1.0
    min_capacity = 0.5
  }
  vpc_security_group_ids = [aws_security_group.ztmf_db.id]

  # auto_minor_version_upgrade is on for the instance, so AWS applies minor
  # patches on its own schedule and the live version drifts ahead of the
  # literal above (16.11 today). Without this, the next AWS-applied patch makes
  # every subsequent plan propose a downgrade back to the pinned value. Refresh
  # still records the real version in state, so the instance reference below
  # continues to resolve correctly. Remove this block temporarily to perform a
  # deliberate, reviewed version change.
  lifecycle {
    ignore_changes = [engine_version]
  }
}

resource "aws_rds_cluster_instance" "ztmf" {
  cluster_identifier   = aws_rds_cluster.ztmf.id
  ca_cert_identifier   = "rds-ca-ecc384-g1"
  db_subnet_group_name = aws_rds_cluster.ztmf.db_subnet_group_name
  instance_class       = "db.serverless"
  engine               = aws_rds_cluster.ztmf.engine
  engine_version       = aws_rds_cluster.ztmf.engine_version

  # Explicit rather than relying on the provider default, since it is the
  # reason engine_version is ignored above.
  auto_minor_version_upgrade = true

  tags = {
    Name        = local.ztmf_name
    Environment = var.environment
  }

  lifecycle {
    ignore_changes = [engine_version]
  }
}
