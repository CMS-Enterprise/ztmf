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
          local.db_cred_secret
        ]
      },
    ]
  })
}

resource "aws_cloudwatch_log_group" "ztmf_api" {
  name = "ztmf_api"
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
          name  = "ENVIRONMENT"
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
    from_port   = 5432
    to_port     = 5432
    protocol    = "tcp"
    cidr_blocks = [for subnet in data.aws_subnet.private : subnet.cidr_block]
  }
}

resource "aws_ecs_service" "ztmf_api" {
  name            = "ztmf-api"
  cluster         = aws_ecs_cluster.ztmf.id
  task_definition = aws_ecs_task_definition.ztmf_api.arn
  launch_type     = "FARGATE"
  desired_count   = 1

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

