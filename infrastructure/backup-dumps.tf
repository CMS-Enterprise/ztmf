# Monthly logical backup of the Aurora cluster to S3.
#
# Third leg of the backup strategy, alongside 35-day PITR and the AWS Backup
# `d15_w90` enrollment in rds.tf. Each leg covers a different discovery lag:
#
#   lag           PITR @ 35d   d15_w90 chain   monthly dump
#   <= 35 days    yes          yes             yes
#   36-95 days    no           yes             yes
#   96d - 1 year  no           no              yes
#   > 1 year      no           no              yes
#
# The incident that prompted this surfaced 35 days after the edit, which is
# exactly the Aurora PITR ceiling. Designing to the worst case already observed
# leaves no margin, which is what the two longer-lived legs are for.
#
# Snapshot export to Parquet was considered and rejected: it needs a bucket, an
# IAM role, a customer-managed KMS key with specific grants, and orchestration,
# and produces an artifact you cannot restore from. Aurora cloning is not a
# backup either - a clone holds today's data and costs ~$43/month if left
# running. A pg_dump is strictly better for the same money.

resource "aws_s3_bucket" "ztmf_db_dumps" {
  bucket = "ztmf-db-dumps-${var.environment}"

  # Must be set at creation; enabling Object Lock later requires AWS support.
  object_lock_enabled = true

  tags = {
    Name        = "ZTMF Database Dumps"
    Environment = var.environment
    Purpose     = "Long-horizon logical backups of the Aurora cluster"
  }
}

resource "aws_s3_bucket_public_access_block" "ztmf_db_dumps" {
  bucket = aws_s3_bucket.ztmf_db_dumps.id

  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

# Required by Object Lock, and independently useful: a same-key overwrite
# cannot destroy the prior dump.
resource "aws_s3_bucket_versioning" "ztmf_db_dumps" {
  bucket = aws_s3_bucket.ztmf_db_dumps.id
  versioning_configuration {
    status = "Enabled"
  }
}

resource "aws_s3_bucket_server_side_encryption_configuration" "ztmf_db_dumps" {
  bucket = aws_s3_bucket.ztmf_db_dumps.id

  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm = "AES256"
    }
    bucket_key_enabled = true
  }
}

# GOVERNANCE rather than COMPLIANCE. GOVERNANCE stops routine and accidental
# deletion while leaving a documented break-glass path for a principal holding
# s3:BypassGovernanceRetention. COMPLIANCE cannot be overridden by anyone
# including the account root for the full retention period, which is a large
# commitment to make on behalf of an account we do not solely control.
resource "aws_s3_bucket_object_lock_configuration" "ztmf_db_dumps" {
  bucket = aws_s3_bucket.ztmf_db_dumps.id

  rule {
    default_retention {
      mode = "GOVERNANCE"
      days = 365
    }
  }

  depends_on = [aws_s3_bucket_versioning.ztmf_db_dumps]
}

# No expiration rule: these are the only copies with no expiry, which is the
# entire reason they exist. Storage class steps down instead. Glacier IR keeps
# millisecond retrieval, so a dump stays as easy to diff at two years as at two
# months - important because the recovery procedure is "read it", not "restore
# the cluster".
resource "aws_s3_bucket_lifecycle_configuration" "ztmf_db_dumps" {
  bucket = aws_s3_bucket.ztmf_db_dumps.id

  rule {
    id     = "dumps-to-glacier-ir"
    status = "Enabled"

    filter {
      prefix = "pgdump/"
    }

    transition {
      days          = 90
      storage_class = "GLACIER_IR"
    }

    noncurrent_version_transition {
      noncurrent_days = 90
      storage_class   = "GLACIER_IR"
    }
  }

  rule {
    id     = "abort-incomplete-uploads"
    status = "Enabled"

    filter {}

    abort_incomplete_multipart_upload {
      days_after_initiation = 7
    }
  }

  depends_on = [aws_s3_bucket_versioning.ztmf_db_dumps]
}

# Dedicated task role rather than reusing module.ops_task. The interactive ops
# role exists so an operator can reach the database; this one additionally
# writes backups, and the two should not grow into each other.
module "db_dump_task" {
  name      = "ztmf_db_dump_task${local.underscore_sfx}"
  source    = "./modules/role"
  principal = { Service = "ecs-tasks.amazonaws.com" }
}

resource "aws_iam_role_policy" "ztmf_db_dump_task" {
  name = "dbDumpTaskPermissions"
  role = module.db_dump_task.role_id
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = [
          "secretsmanager:GetSecretValue",
          "secretsmanager:DescribeSecret",
        ]
        Effect   = "Allow"
        Resource = [local.db_cred_secret]
      },
      {
        # Write-only. The task never needs to read a dump back, and no
        # s3:DeleteObject means a compromised task cannot erase history even
        # before Object Lock is considered. AbortMultipartUpload only lets it
        # clean up its own failed upload; the lifecycle rule sweeps whatever it
        # misses after 7 days.
        Action = [
          "s3:PutObject",
          "s3:AbortMultipartUpload",
        ]
        Effect   = "Allow"
        Resource = ["${aws_s3_bucket.ztmf_db_dumps.arn}/*"]
      },
      {
        # PutMetricData cannot be scoped to a resource; the namespace
        # condition is the supported way to constrain it.
        Action   = ["cloudwatch:PutMetricData"]
        Effect   = "Allow"
        Resource = ["*"]
        Condition = {
          StringEquals = {
            "cloudwatch:namespace" = "ZTMF/Backup"
          }
        }
      },
    ]
  })
}

resource "aws_cloudwatch_log_group" "ztmf_db_dump" {
  name              = "ztmf_db_dump${local.underscore_sfx}"
  retention_in_days = 90

  tags = {
    Name        = "ZTMF DB Dump"
    Environment = var.environment
  }
}

# Same image as the interactive ops task, different entrypoint. Memory is
# raised over the 512 MB interactive task to give pg_dump's compression buffers
# headroom; the dump itself lands in the task's ephemeral storage before upload.
resource "aws_ecs_task_definition" "ztmf_db_dump" {
  execution_role_arn       = module.ops_task_execution.role_arn
  task_role_arn            = module.db_dump_task.role_arn
  family                   = "db-dump${local.name_suffix}"
  requires_compatibilities = ["FARGATE"]
  network_mode             = "awsvpc"
  cpu                      = 512
  memory                   = 1024
  container_definitions = jsonencode([
    {
      name      = "ztmfdbdump"
      image     = "${local.ecr_ops_repo_url}:${data.aws_ssm_parameter.ztmf_ops_tag.insecure_value}"
      essential = true
      command   = ["/usr/local/bin/dump-db.sh"]

      environment = [
        { name = "ENVIRONMENT", value = var.environment },
        { name = "AWS_REGION", value = "us-east-1" },
        { name = "DB_NAME", value = "ztmf" },
        { name = "DB_ENDPOINT", value = aws_rds_cluster.ztmf.endpoint },
        { name = "DB_PORT", value = "5432" },
        { name = "DB_SECRET_ID", value = local.db_cred_secret },
        { name = "DUMP_BUCKET", value = aws_s3_bucket.ztmf_db_dumps.bucket },
      ]

      logConfiguration = {
        logDriver = "awslogs"
        options = {
          "awslogs-group"         = aws_cloudwatch_log_group.ztmf_db_dump.name
          "awslogs-region"        = "us-east-1"
          "awslogs-stream-prefix" = "dump"
        }
      }
    }
  ])
}

resource "aws_security_group" "ztmf_db_dump_task" {
  name        = "ztmf_db_dump_task${local.underscore_sfx}"
  description = "scheduled db dump task: Aurora, VPC endpoints, CloudWatch via NAT"
  vpc_id      = data.aws_vpc.ztmf.id

  egress {
    description = "HTTPS to VPC endpoints (ECR, Secrets Manager, S3, CloudWatch Logs)"
    from_port   = 443
    to_port     = 443
    protocol    = "tcp"
    security_groups = [
      local.manage_vpc_endpoints
      ? aws_security_group.ztmf_vpc_endpoints[0].id
      : data.aws_security_group.ztmf_vpc_endpoints[0].id
    ]
  }

  egress {
    description     = "PostgreSQL to Aurora"
    from_port       = 5432
    to_port         = 5432
    protocol        = "tcp"
    security_groups = [aws_security_group.ztmf_db.id]
  }

  # The VPC has interface endpoints for ecr, logs, s3, secretsmanager, ssm and
  # friends, but none for `monitoring`, so the PutMetricData calls egress
  # through the CMS-provided NAT gateway rather than adding an endpoint. This
  # rule is also what carries any AWS API call whose endpoint resolves
  # publicly, which in practice includes some of the S3 traffic.
  egress {
    description = "HTTPS to CloudWatch and other AWS APIs via the CMS-provided NAT gateway"
    from_port   = 443
    to_port     = 443
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }
}

# EventBridge Scheduler rather than an EventBridge rule: it is the current AWS
# recommendation for scheduled invocations, supports a native ECS RunTask
# target without a rule/target pair, and matches the direction of the pending
# migration of the existing Lambda schedules.
module "db_dump_scheduler" {
  name      = "ztmf_db_dump_scheduler${local.underscore_sfx}"
  source    = "./modules/role"
  principal = { Service = "scheduler.amazonaws.com" }
}

resource "aws_iam_role_policy" "ztmf_db_dump_scheduler" {
  name = "dbDumpSchedulerPermissions"
  role = module.db_dump_scheduler.role_id
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = ["ecs:RunTask"]
        Effect = "Allow"
        # RunTask is authorized against the revision-less family ARN so a new
        # task definition revision does not require a policy update.
        Resource = ["${aws_ecs_task_definition.ztmf_db_dump.arn_without_revision}:*"]
        Condition = {
          ArnLike = {
            "ecs:cluster" = aws_ecs_cluster.ztmf.arn
          }
        }
      },
      {
        Action = ["iam:PassRole"]
        Effect = "Allow"
        Resource = [
          module.db_dump_task.role_arn,
          module.ops_task_execution.role_arn,
        ]
        Condition = {
          StringLike = {
            "iam:PassedToService" = "ecs-tasks.amazonaws.com"
          }
        }
      },
    ]
  })
}

# 06:30 UTC on the 1st. Deliberately outside the 06:00-06:30 Aurora backup
# window so the two do not contend, and well clear of the Sunday maintenance
# window.
resource "aws_scheduler_schedule" "ztmf_db_dump" {
  name        = "ztmf-db-dump-${var.environment}"
  description = "Monthly pg_dump of the ZTMF Aurora cluster to S3"
  state       = var.db_dump_enabled ? "ENABLED" : "DISABLED"

  schedule_expression          = "cron(30 6 1 * ? *)"
  schedule_expression_timezone = "UTC"

  flexible_time_window {
    mode = "OFF"
  }

  target {
    arn      = aws_ecs_cluster.ztmf.arn
    role_arn = module.db_dump_scheduler.role_arn

    ecs_parameters {
      task_definition_arn = aws_ecs_task_definition.ztmf_db_dump.arn
      launch_type         = "FARGATE"
      task_count          = 1

      network_configuration {
        assign_public_ip = false
        subnets          = data.aws_subnets.private.ids
        security_groups  = [aws_security_group.ztmf_db_dump_task.id]
      }
    }

    # Covers failure to *invoke* RunTask (throttling, a transient ECS API
    # error) only. Once ECS accepts the task the delivery is successful as far
    # as Scheduler is concerned, so a container that starts and then fails is
    # not retried here - that case is caught by DaysSinceLastDump staying high
    # into the next day.
    retry_policy {
      maximum_event_age_in_seconds = 3600
      maximum_retry_attempts       = 3
    }
  }
}

# Daily staleness probe.
#
# The obvious design - have the dump publish a success pulse and alarm on
# missing data over 40 days - is not expressible. CloudWatch caps an alarm's
# total evaluation range (period x evaluation_periods) at 604800 seconds, so
# the longest possible lookback is 7 days. A monthly pulse leaves ~29
# consecutive days with no datapoint, which any legal alarm reads as breaching.
#
# So freshness is measured instead of inferred: this task lists the bucket once
# a day and publishes the age of the newest dump, which the alarm below reads
# over a single 1-day period. It also catches an upload that the dump job
# believed succeeded but which never landed, since it trusts the bucket rather
# than the previous job's own report.
resource "aws_ecs_task_definition" "ztmf_db_dump_check" {
  execution_role_arn       = module.ops_task_execution.role_arn
  task_role_arn            = module.db_dump_check.role_arn
  family                   = "db-dump-check${local.name_suffix}"
  requires_compatibilities = ["FARGATE"]
  network_mode             = "awsvpc"
  cpu                      = 256
  memory                   = 512
  container_definitions = jsonencode([
    {
      name      = "ztmfdbdumpcheck"
      image     = "${local.ecr_ops_repo_url}:${data.aws_ssm_parameter.ztmf_ops_tag.insecure_value}"
      essential = true
      command   = ["/usr/local/bin/check-dump-age.sh"]

      environment = [
        { name = "ENVIRONMENT", value = var.environment },
        { name = "AWS_REGION", value = "us-east-1" },
        { name = "DUMP_BUCKET", value = aws_s3_bucket.ztmf_db_dumps.bucket },
      ]

      logConfiguration = {
        logDriver = "awslogs"
        options = {
          "awslogs-group"         = aws_cloudwatch_log_group.ztmf_db_dump.name
          "awslogs-region"        = "us-east-1"
          "awslogs-stream-prefix" = "check"
        }
      }
    }
  ])
}

# Read-only against the bucket. It never needs the database credentials, so it
# does not get them.
module "db_dump_check" {
  name      = "ztmf_db_dump_check${local.underscore_sfx}"
  source    = "./modules/role"
  principal = { Service = "ecs-tasks.amazonaws.com" }
}

resource "aws_iam_role_policy" "ztmf_db_dump_check" {
  name = "dbDumpCheckPermissions"
  role = module.db_dump_check.role_id
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action   = ["s3:ListBucket"]
        Effect   = "Allow"
        Resource = [aws_s3_bucket.ztmf_db_dumps.arn]
      },
      {
        Action   = ["cloudwatch:PutMetricData"]
        Effect   = "Allow"
        Resource = ["*"]
        Condition = {
          StringEquals = {
            "cloudwatch:namespace" = "ZTMF/Backup"
          }
        }
      },
    ]
  })
}

resource "aws_iam_role_policy" "ztmf_db_dump_check_scheduler" {
  name = "dbDumpCheckSchedulerPermissions"
  role = module.db_dump_scheduler.role_id
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action   = ["ecs:RunTask"]
        Effect   = "Allow"
        Resource = ["${aws_ecs_task_definition.ztmf_db_dump_check.arn_without_revision}:*"]
        Condition = {
          ArnLike = {
            "ecs:cluster" = aws_ecs_cluster.ztmf.arn
          }
        }
      },
      {
        Action   = ["iam:PassRole"]
        Effect   = "Allow"
        Resource = [module.db_dump_check.role_arn]
        Condition = {
          StringLike = {
            "iam:PassedToService" = "ecs-tasks.amazonaws.com"
          }
        }
      },
    ]
  })
}

# 07:45 UTC daily, clear of both the Aurora backup window and the monthly dump
# so a same-day dump is already in the bucket when the age is measured.
resource "aws_scheduler_schedule" "ztmf_db_dump_check" {
  name        = "ztmf-db-dump-check-${var.environment}"
  description = "Daily freshness probe for the ZTMF database dumps"
  state       = var.db_dump_enabled ? "ENABLED" : "DISABLED"

  schedule_expression          = "cron(45 7 * * ? *)"
  schedule_expression_timezone = "UTC"

  flexible_time_window {
    mode = "OFF"
  }

  target {
    arn      = aws_ecs_cluster.ztmf.arn
    role_arn = module.db_dump_scheduler.role_arn

    ecs_parameters {
      task_definition_arn = aws_ecs_task_definition.ztmf_db_dump_check.arn
      launch_type         = "FARGATE"
      task_count          = 1

      network_configuration {
        assign_public_ip = false
        subnets          = data.aws_subnets.private.ids
        security_groups  = [aws_security_group.ztmf_db_dump_task.id]
      }
    }

    retry_policy {
      maximum_event_age_in_seconds = 3600
      maximum_retry_attempts       = 3
    }
  }
}

# Fires when the newest dump is older than 40 days - a month plus margin for a
# missed run - or when the probe itself stops reporting.
#
# Total evaluation range is 86400s, inside the CloudWatch 604800s ceiling.
# treat_missing_data stays "breaching" because the failure worth catching is
# the silent one: a disabled schedule, a broken image, a revoked role. Those
# produce no datapoint at all, and an alarm keyed on task exit status would
# stay green forever while backups quietly stopped.
#
# Counted off var.db_dump_enabled so environments that deliberately run without
# dumps (impl) do not sit permanently in ALARM against a live SNS subscription.
# Note that in an environment where it IS enabled, the alarm goes ALARM on
# first apply and clears once the probe has run - it does not wait 40 days to
# tell you it has no data.
resource "aws_cloudwatch_metric_alarm" "ztmf_db_dump_stale" {
  count               = var.db_dump_enabled ? 1 : 0
  alarm_name          = "ztmf-db-dump-stale-${var.environment}"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = "1"
  datapoints_to_alarm = "1"
  metric_name         = "DaysSinceLastDump"
  namespace           = "ZTMF/Backup"
  period              = "86400"
  statistic           = "Maximum"
  threshold           = "40"
  treat_missing_data  = "breaching"
  alarm_description   = "The newest ZTMF database dump is more than 40 days old, or the daily freshness probe has stopped reporting. Check the ztmf-db-dump and ztmf-db-dump-check schedules and the ztmf_db_dump log group."
  alarm_actions       = [aws_sns_topic.ztmf_alarms.arn]
  ok_actions          = [aws_sns_topic.ztmf_alarms.arn]

  dimensions = {
    Environment = var.environment
  }

  tags = {
    Name        = "ZTMF DB Dump Stale Alarm"
    Environment = var.environment
  }
}
