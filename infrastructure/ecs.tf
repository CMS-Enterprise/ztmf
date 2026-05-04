resource "aws_ecs_cluster" "ztmf" {
  name = "ztmf"

  setting {
    name  = "containerInsights"
    value = "enabled"
  }
}

module "api_task_execution" {
  name                = "ztmf_api_task_execution"
  source              = "./modules/role"
  principal           = { Service = "ecs-tasks.amazonaws.com" }
  managed_policy_arns = ["arn:aws:iam::aws:policy/service-role/AmazonECSTaskExecutionRolePolicy"]
}

module "api_task" {
  name      = "ztmf_api_task"
  source    = "./modules/role"
  principal = { Service = "ecs-tasks.amazonaws.com" }
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
          aws_secretsmanager_secret.ztmf_smtp.arn,
          aws_secretsmanager_secret.ztmf_smtp_ca_root.arn,
          aws_secretsmanager_secret.ztmf_smtp_intermediate.arn,
        ]
      },
    ]
  })
}

resource "aws_cloudwatch_log_group" "ztmf_api" {
  name = "ztmf_api"
  # Match CMS Cloud loggroups-retention-policy-lambda which resets every
  # log group to 731 days on the 1st of each month. Without this terraform
  # would drift back to "never expire" after every apply.
  retention_in_days = 731
}

resource "aws_ecs_task_definition" "ztmf_api" {
  execution_role_arn       = module.api_task_execution.role_arn
  task_role_arn            = module.api_task.role_arn
  family                   = "api"
  requires_compatibilities = ["FARGATE"]
  network_mode             = "awsvpc"
  cpu                      = 256
  memory                   = 512
  container_definitions = jsonencode([
    {
      name             = "ztmfapi"
      command          = ["/usr/local/bin/ztmfapi"]
      workingDirectory = "/api"
      image            = "${aws_ecr_repository.ztmf_api.repository_url}:${data.aws_ssm_parameter.ztmf_api_tag.insecure_value}"
      essential        = true
      portMappings     = [{ containerPort = 443 }]

      environment = [
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
          value = aws_secretsmanager_secret.ztmf_smtp.arn
        },
        {
          name  = "SMTP_CA_ROOT_SECRET_ID"
          value = aws_secretsmanager_secret.ztmf_smtp_ca_root.arn
        },
        {
          name  = "SMTP_CA_INT_SECRET_ID"
          value = aws_secretsmanager_secret.ztmf_smtp_intermediate.arn
        }
      ]
      logConfiguration = {
        logDriver = "awslogs"
        options = {
          "awslogs-group"         = "ztmf_api"
          "awslogs-region"        = "us-east-1"
          "awslogs-stream-prefix" = "api"
        }
      }
    }
  ])
}

resource "aws_security_group" "ztmf_api_task" {
  name        = "ztmf-api-task"
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
  name            = "ztmf-api"
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
}

# On-demand ops task (replaces the EC2 bastion). Operators launch via
# `make db-shell-<env>` / `make db-forward-<env>` (scripts/db-tunnel.sh).
# No service / desired_count: launched via `aws ecs run-task`, stopped when the
# operator session ends. Image built from backend/ops/Dockerfile.

module "ops_task_execution" {
  name                = "ztmf_ops_task_execution"
  source              = "./modules/role"
  principal           = { Service = "ecs-tasks.amazonaws.com" }
  managed_policy_arns = ["arn:aws:iam::aws:policy/service-role/AmazonECSTaskExecutionRolePolicy"]
}

module "ops_task" {
  name      = "ztmf_ops_task"
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
  name              = "ztmf_ops"
  retention_in_days = 30
}

resource "aws_ssm_parameter" "ztmf_ops_tag" {
  name  = "ztmf_ops_tag"
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
  family                   = "ops"
  requires_compatibilities = ["FARGATE"]
  network_mode             = "awsvpc"
  cpu                      = 256
  memory                   = 512
  container_definitions = jsonencode([
    {
      name      = "ztmfops"
      image     = "${aws_ecr_repository.ztmf_ops.repository_url}:${data.aws_ssm_parameter.ztmf_ops_tag.insecure_value}"
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
  name        = "ztmf_ops_task"
  description = "on-demand ops task: ECS Exec + Aurora access"
  vpc_id      = data.aws_vpc.ztmf.id

  egress {
    description     = "HTTPS to VPC endpoints (ECR, SSM, Secrets Manager, CloudWatch)"
    from_port       = 443
    to_port         = 443
    protocol        = "tcp"
    security_groups = [aws_security_group.ztmf_vpc_endpoints.id]
  }

  egress {
    description     = "PostgreSQL to Aurora"
    from_port       = 5432
    to_port         = 5432
    protocol        = "tcp"
    security_groups = [aws_security_group.ztmf_db.id]
  }
}

