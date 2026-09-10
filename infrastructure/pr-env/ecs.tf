resource "aws_cloudwatch_log_group" "env" {
  name              = "/ztmf/${local.env_name}"
  retention_in_days = 7
}

resource "aws_security_group" "task" {
  name        = "ztmf-${local.env_name}-task"
  description = "PR environment task: HTTPS from the ALB only"
  vpc_id      = data.aws_vpc.ztmf.id

  ingress {
    description     = "HTTPS from the internal ALB"
    from_port       = 443
    to_port         = 443
    protocol        = "tcp"
    security_groups = [data.aws_security_group.alb.id]
  }

  egress {
    description = "Image pulls (ECR, ECR Public) and SSM"
    from_port   = 443
    to_port     = 443
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }
}

// Three containers in one task share localhost: Postgres seeds nothing itself,
// the API migrates and seeds the empire data on start (ENVIRONMENT=test plus
// DB_POPULATE), and nginx terminates TLS for the target group, serves the
// bundle under the PR prefix, and proxies api/ to the API.
resource "aws_ecs_task_definition" "env" {
  family                   = local.env_name
  requires_compatibilities = ["FARGATE"]
  network_mode             = "awsvpc"
  cpu                      = 512
  memory                   = 1024
  execution_role_arn       = module.task_execution.role_arn
  task_role_arn            = module.task.role_arn

  container_definitions = jsonencode([
    {
      name      = "postgres"
      image     = var.postgres_image
      essential = true
      environment = [
        { name = "POSTGRES_DB", value = "ztmf" },
        { name = "POSTGRES_USER", value = "ztmf" },
      ]
      secrets = [
        { name = "POSTGRES_PASSWORD", valueFrom = aws_ssm_parameter.db_password.arn },
      ]
      healthCheck = {
        command     = ["CMD-SHELL", "pg_isready -U ztmf -d ztmf"]
        interval    = 5
        timeout     = 3
        retries     = 10
        startPeriod = 10
      }
      logConfiguration = local.log_config["postgres"]
    },
    {
      name      = "api"
      image     = local.api_image
      essential = true
      command   = ["/usr/local/bin/ztmfapi"]
      dependsOn = [{ containerName = "postgres", condition = "HEALTHY" }]
      environment = [
        { name = "ENVIRONMENT", value = "test" },
        { name = "PORT", value = tostring(local.api_port) },
        { name = "DB_ENDPOINT", value = "localhost" },
        { name = "DB_PORT", value = "5432" },
        { name = "DB_NAME", value = "ztmf" },
        { name = "DB_USER", value = "ztmf" },
        { name = "DB_POPULATE", value = "/app/_test_data_empire.sql" },
        { name = "AUTH_HEADER_FIELD", value = "Authorization" },
        { name = "AWS_REGION", value = "us-east-1" },
      ]
      secrets = [
        { name = "DB_PASS", valueFrom = aws_ssm_parameter.db_password.arn },
        { name = "AUTH_HS256_SECRET", valueFrom = aws_ssm_parameter.hs256.arn },
      ]
      logConfiguration = local.log_config["api"]
    },
    {
      name         = "ui"
      image        = local.ui_image
      essential    = true
      dependsOn    = [{ containerName = "api", condition = "START" }]
      portMappings = [{ containerPort = 443, protocol = "tcp" }]
      environment = [
        { name = "PR_PATH_PREFIX", value = local.path_prefix },
        { name = "API_UPSTREAM", value = "http://127.0.0.1:${local.api_port}" },
        { name = "TEST_USER_EMAIL", value = var.test_user_email },
      ]
      secrets = [
        { name = "AUTH_HS256_SECRET", valueFrom = aws_ssm_parameter.hs256.arn },
      ]
      logConfiguration = local.log_config["ui"]
    },
  ])
}

resource "aws_ecs_service" "env" {
  name                   = local.env_name
  cluster                = data.aws_ecs_cluster.ztmf.id
  task_definition        = aws_ecs_task_definition.env.arn
  launch_type            = "FARGATE"
  desired_count          = 1
  enable_execute_command = true
  // A crash loop (bad image, failed migration) fails the apply instead of
  // leaving a half-up environment behind a green workflow.
  wait_for_steady_state = true

  load_balancer {
    target_group_arn = aws_lb_target_group.env.arn
    container_name   = "ui"
    container_port   = 443
  }

  network_configuration {
    assign_public_ip = false
    subnets          = data.aws_subnets.private.ids
    security_groups  = [aws_security_group.task.id]
  }

  depends_on = [aws_lb_listener_rule.env]
}
