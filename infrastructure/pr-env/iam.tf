module "task_execution" {
  source              = "../modules/role"
  name                = "ztmf_${local.env_name}_task_execution"
  principal           = { Service = "ecs-tasks.amazonaws.com" }
  managed_policy_arns = ["arn:aws:iam::aws:policy/service-role/AmazonECSTaskExecutionRolePolicy"]
}

// Container `secrets` resolve through the execution role: only this
// environment's two parameters, plus decrypt on the SSM default key.
resource "aws_iam_role_policy" "task_execution_secrets" {
  name = "prEnvSecrets"
  role = module.task_execution.role_id
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect   = "Allow"
        Action   = ["ssm:GetParameters"]
        Resource = [aws_ssm_parameter.hs256.arn, aws_ssm_parameter.db_password.arn]
      },
      {
        Effect   = "Allow"
        Action   = ["kms:Decrypt"]
        Resource = [data.aws_kms_alias.ssm.target_key_arn]
      },
    ]
  })
}

// The API needs no AWS access in test mode (no Secrets Manager DB creds, no
// SMTP). The task role exists for ECS Exec so a broken environment can be
// inspected without redeploying.
module "task" {
  source    = "../modules/role"
  name      = "ztmf_${local.env_name}_task"
  principal = { Service = "ecs-tasks.amazonaws.com" }
}

resource "aws_iam_role_policy" "task_exec_channel" {
  name = "ecsExec"
  role = module.task.role_id
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "ssmmessages:CreateControlChannel",
          "ssmmessages:CreateDataChannel",
          "ssmmessages:OpenControlChannel",
          "ssmmessages:OpenDataChannel",
        ]
        Resource = ["*"]
      },
    ]
  })
}
