locals {
  env_name    = "pr-${var.repo}-${var.pr_number}"
  path_prefix = "/pr/${var.repo}/${var.pr_number}"
  url         = "https://dev.ztmf.cms.gov${local.path_prefix}/"

  // Priority bands keep PR rules clear of the dev root's 1-4 and of each other
  // across repos (ALB max is 50000).
  listener_priority = (var.repo == "ztmf" ? 10000 : 20000) + var.pr_number

  api_image = "${data.aws_ecr_repository.api.repository_url}:${var.api_image_tag}"
  ui_image  = "${data.aws_ecr_repository.ui.repository_url}:${var.ui_image_tag}"

  // The API listens on plain HTTP inside the task; nginx terminates TLS for the
  // target group and proxies to it over localhost (awsvpc shares the namespace).
  api_port = 8080

  log_config = {
    for name in ["postgres", "api", "ui"] : name => {
      logDriver = "awslogs"
      options = {
        "awslogs-group"         = "/ztmf/${local.env_name}"
        "awslogs-region"        = "us-east-1"
        "awslogs-stream-prefix" = name
      }
    }
  }
}
