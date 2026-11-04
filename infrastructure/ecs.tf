resource "aws_ecs_cluster" "ztmf" {
  name = local.ztmf_name

  setting {
    name  = "containerInsights"
    value = "enabled"
  }
}

module "api_task_execution" {
  name                = "ztmf_api_task_execution${local.underscore_sfx}"
  source              = "./modules/role"
  principal           = { Service = "ecs-tasks.amazonaws.com" }
  managed_policy_arns = ["arn:aws:iam::aws:policy/service-role/AmazonECSTaskExecutionRolePolicy"]
}

module "api_task" {
  name      = "ztmf_api_task${local.underscore_sfx}"
  source    = "./modules/role"
  principal = { Service = "ecs-tasks.amazonaws.com" }
}

// ECS resolves container `secrets` (the session signing key) using the task
// EXECUTION role, so it needs read access to that secret. Only granted when
// the dual-IdP feature is enabled.
resource "aws_iam_role_policy" "ztmf_api_task_execution_entra" {
  count = var.entra_enabled ? 1 : 0
  name  = "entraSessionSecret"
  role  = module.api_task_execution.role_id
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action   = ["secretsmanager:GetSecretValue", "secretsmanager:DescribeSecret"]
        Effect   = "Allow"
        Resource = [aws_secretsmanager_secret.ztmf_session_signing_key.arn]
      },
    ]
  })
}

// CrowdStrike Falcon sensor (ztmf-misc#437). FALCONCTL_OPTS (CID,
// provisioning token, tags) lives in a SecureString created outside Terraform,
// one per cluster. The ARN is built from its name rather than read with a data
// source, so the value never lands in state.
locals {
  falconctl_opts_arn = "arn:${data.aws_partition.current.partition}:ssm:${data.aws_region.current.region}:${local.account_id}:parameter/crowdstrike/falconctl-opts/${local.ztmf_name}"
}

// Container `secrets` resolve through the execution role. The parameter uses
// the aws/ssm default key, so no kms:Decrypt grant is needed.
resource "aws_iam_role_policy" "falcon_task_execution" {
  for_each = var.falcon_enabled ? {
    api = module.api_task_execution.role_id
    ops = module.ops_task_execution.role_id
  } : {}
  name = "falconSensorOpts"
  role = each.value
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action   = ["ssm:GetParameters"]
        Effect   = "Allow"
        Resource = [local.falconctl_opts_arn]
      },
    ]
  })
}

resource "aws_iam_role_policy" "ztmf_api_task" {
  name = "taskExecutionPermissions"
  role = module.api_task.role_id
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = [
          "secretsmanager:GetSecretValue",
          "secretsmanager:DescribeSecret"
        ]
        Effect = "Allow"
        Resource = [
          local.db_cred_secret,
          local.smtp_secret_arn,
          local.smtp_ca_root_arn,
          local.smtp_intermediate_arn,
        ]
      },
    ]
  })
}

resource "aws_cloudwatch_log_group" "ztmf_api" {
  name = local.ztmf_api_log_group
  # Match CMS Cloud loggroups-retention-policy-lambda which resets every
  # log group to 731 days on the 1st of each month. Without this terraform
  # would drift back to "never expire" after every apply.
  retention_in_days = 731
}

resource "aws_ecs_task_definition" "ztmf_api" {
  execution_role_arn       = module.api_task_execution.role_arn
  task_role_arn            = module.api_task.role_arn
  family                   = "api${local.name_suffix}"
  requires_compatibilities = ["FARGATE"]
  network_mode             = "awsvpc"
  cpu                      = 256
  // Headroom for the sensor process that runs inside the container.
  memory                = var.falcon_enabled ? 1024 : 512
  container_definitions = module.falcon_api.container_definitions

  dynamic "volume" {
    for_each = module.falcon_api.volume_names
    content {
      name = volume.value
    }
  }
}

module "falcon_api" {
  source             = "./modules/falcon-sensor"
  enabled            = var.falcon_enabled
  image              = var.falcon_image
  falconctl_opts_arn = local.falconctl_opts_arn
  init_log_configuration = {
    logDriver = "awslogs"
    options = {
      "awslogs-group"         = aws_cloudwatch_log_group.ztmf_api.name
      "awslogs-region"        = "us-east-1"
      "awslogs-stream-prefix" = "falcon-init"
    }
  }
  // distroless image with no ENTRYPOINT: the binary becomes the wrapped
  // entrypoint and there is no command, so it still runs with no arguments.
  entrypoints = {
    ztmfapi = { entry_point = ["/usr/local/bin/ztmfapi"] }
  }
  container_definitions = jsonencode([
    {
      name             = "ztmfapi"
      command          = ["/usr/local/bin/ztmfapi"]
      workingDirectory = "/api"
      image            = "${local.ecr_api_repo_url}:${data.aws_ssm_parameter.ztmf_api_tag.insecure_value}"
      essential        = true
      portMappings     = [{ containerPort = 443 }]

      environment = concat([
        {
          name  = "ENVIRONMENT" // only for logging, application code should not depend on any particular value
          value = var.environment
        },
        {
          name  = "PORT"
          value = "443"
        },
        {
          name  = "CERT_FILE"
          value = "/src/cert.pem"
        },
        {
          name  = "KEY_FILE"
          value = "/src/key.pem"
        },
        {
          name  = "DB_NAME"
          value = "ztmf"
        },
        {
          name  = "DB_ENDPOINT"
          value = aws_rds_cluster.ztmf.endpoint
        },
        {
          name  = "DB_PORT"
          value = "5432"
        },
        {
          name  = "DB_SECRET_ID"
          value = local.db_cred_secret
        },
        {
          name  = "AWS_REGION"
          value = "us-east-1"
        },
        {
          name  = "AUTH_TOKEN_KEY_URL"
          value = "https://public-keys.auth.elb.us-east-1.amazonaws.com/"
        },
        {
          name  = "AUTH_HEADER_FIELD"
          value = "x-amzn-oidc-data"
        },
        {
          name  = "SMTP_CONFIG_SECRET_ID"
          value = local.smtp_secret_arn
        },
        {
          name  = "SMTP_CA_ROOT_SECRET_ID"
          value = local.smtp_ca_root_arn
        },
        {
          name  = "SMTP_CA_INT_SECRET_ID"
          value = local.smtp_intermediate_arn
        }
      ], local.entra_api_env)
      secrets = local.entra_api_secrets
      logConfiguration = {
        logDriver = "awslogs"
        options = {
          "awslogs-group"         = aws_cloudwatch_log_group.ztmf_api.name
          "awslogs-region"        = "us-east-1"
          "awslogs-stream-prefix" = "api"
        }
      }
    }
  ])
}

resource "aws_security_group" "ztmf_api_task" {
  name        = local.ztmf_api_task_sg
  description = "Allow TLS inbound traffic"
  vpc_id      = data.aws_vpc.ztmf.id

  ingress {
    description = "HTTPS from VPC CIDR"
    from_port   = 443
    to_port     = 443
    protocol    = "tcp"
    cidr_blocks = [data.aws_vpc.ztmf.cidr_block]
  }

  egress {
    from_port   = 443
    to_port     = 443
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  egress {
    description = "Aurora Postgres"
    from_port   = 5432
    to_port     = 5432
    protocol    = "tcp"
    cidr_blocks = [for subnet in data.aws_subnet.private : subnet.cidr_block]
  }

  egress {
    description     = "SMTP"
    from_port       = 587
    to_port         = 587
    protocol        = "tcp"
    prefix_list_ids = [data.aws_ec2_managed_prefix_list.shared_services.id]
  }
}

resource "aws_ecs_service" "ztmf_api" {
  name            = local.ztmf_api_name
  cluster         = aws_ecs_cluster.ztmf.id
  task_definition = aws_ecs_task_definition.ztmf_api.arn
  launch_type     = "FARGATE"
  desired_count   = var.ecs_service_task_count

  load_balancer {
    target_group_arn = aws_lb_target_group.ztmf_rest_api.arn
    container_name   = "ztmfapi"
    container_port   = 443
  }

  network_configuration {
    assign_public_ip = false
    subnets          = data.aws_subnets.private.ids
    security_groups  = [aws_security_group.ztmf_api_task.id]
  }
  wait_for_steady_state = true

  // The execution role must be able to read FALCONCTL_OPTS before the new
  // task definition rolls out.
  depends_on = [aws_iam_role_policy.falcon_task_execution]
}

# On-demand ops task (replaces the EC2 bastion). Operators launch via
# `make db-shell-<env>` / `make db-forward-<env>` (scripts/db-tunnel.sh).
# No service / desired_count: launched via `aws ecs run-task`, stopped when the
# operator session ends. Image built from backend/ops/Dockerfile.

module "ops_task_execution" {
  name                = "ztmf_ops_task_execution${local.underscore_sfx}"
  source              = "./modules/role"
  principal           = { Service = "ecs-tasks.amazonaws.com" }
  managed_policy_arns = ["arn:aws:iam::aws:policy/service-role/AmazonECSTaskExecutionRolePolicy"]
}

module "ops_task" {
  name      = "ztmf_ops_task${local.underscore_sfx}"
  source    = "./modules/role"
  principal = { Service = "ecs-tasks.amazonaws.com" }
}

resource "aws_iam_role_policy" "ztmf_ops_task" {
  name = "opsTaskPermissions"
  role = module.ops_task.role_id
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
        Action = [
          "ssmmessages:CreateControlChannel",
          "ssmmessages:CreateDataChannel",
          "ssmmessages:OpenControlChannel",
          "ssmmessages:OpenDataChannel",
        ]
        Effect   = "Allow"
        Resource = ["*"]
      },
    ]
  })
}

resource "aws_cloudwatch_log_group" "ztmf_ops" {
  name              = local.ztmf_ops_log_group
  retention_in_days = 30
}

resource "aws_ssm_parameter" "ztmf_ops_tag" {
  name  = "ztmf_ops_tag${local.underscore_sfx}"
  type  = "String"
  tier  = "Standard"
  value = "bootstrap"

  lifecycle {
    ignore_changes = [value]
  }
}

resource "aws_ecs_task_definition" "ztmf_ops" {
  execution_role_arn       = module.ops_task_execution.role_arn
  task_role_arn            = module.ops_task.role_arn
  family                   = "ops${local.name_suffix}"
  requires_compatibilities = ["FARGATE"]
  network_mode             = "awsvpc"
  cpu                      = 256
  // Headroom for the sensor process that runs inside the container.
  memory                = var.falcon_enabled ? 1024 : 512
  container_definitions = module.falcon_ops.container_definitions

  dynamic "volume" {
    for_each = module.falcon_ops.volume_names
    content {
      name = volume.value
    }
  }
}

module "falcon_ops" {
  source             = "./modules/falcon-sensor"
  enabled            = var.falcon_enabled
  image              = var.falcon_image
  falconctl_opts_arn = local.falconctl_opts_arn
  init_log_configuration = {
    logDriver = "awslogs"
    options = {
      "awslogs-group"         = aws_cloudwatch_log_group.ztmf_ops.name
      "awslogs-region"        = "us-east-1"
      "awslogs-stream-prefix" = "falcon-init"
    }
  }
  // Image CMD is ["sleep", "infinity"] with no ENTRYPOINT (backend/ops/Dockerfile).
  // In wolfi-base, sleep is a busybox applet, so start busybox itself rather
  // than the /bin/sleep symlink, which depends on argv[0] surviving the wrapper.
  entrypoints = {
    ztmfops = { entry_point = ["/usr/bin/busybox"], command = ["sleep", "infinity"] }
  }
  container_definitions = jsonencode([
    {
      name      = "ztmfops"
      image     = "${local.ecr_ops_repo_url}:${data.aws_ssm_parameter.ztmf_ops_tag.insecure_value}"
      essential = true

      environment = [
        { name = "ENVIRONMENT", value = var.environment },
        { name = "AWS_REGION", value = "us-east-1" },
        { name = "DB_NAME", value = "ztmf" },
        { name = "DB_ENDPOINT", value = aws_rds_cluster.ztmf.endpoint },
        { name = "DB_PORT", value = "5432" },
        { name = "DB_SECRET_ID", value = local.db_cred_secret },
      ]

      logConfiguration = {
        logDriver = "awslogs"
        options = {
          "awslogs-group"         = aws_cloudwatch_log_group.ztmf_ops.name
          "awslogs-region"        = "us-east-1"
          "awslogs-stream-prefix" = "ops"
        }
      }
    }
  ])
}

resource "aws_security_group" "ztmf_ops_task" {
  name        = local.ztmf_ops_task_sg
  description = "on-demand ops task: ECS Exec + Aurora access"
  vpc_id      = data.aws_vpc.ztmf.id

  egress {
    description = "HTTPS to VPC endpoints (ECR, SSM, Secrets Manager, CloudWatch)"
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

  // The Falcon sensor image comes from CMS Artifactory (shared-services
  // network) and the sensor reports to the CrowdStrike cloud through NAT, so
  // the ops task needs general 443 egress like the api task already has.
  dynamic "egress" {
    for_each = var.falcon_enabled ? [1] : []
    content {
      description = "HTTPS for the Falcon sensor image and CrowdStrike cloud"
      from_port   = 443
      to_port     = 443
      protocol    = "tcp"
      cidr_blocks = ["0.0.0.0/0"]
    }
  }
}
