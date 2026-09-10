// Per-environment secrets. Random, disposable, and scoped to synthetic data:
// the HS256 key signs the test-admin bearer token the frontend carries, the
// DB password guards a sidecar only reachable inside the task.
resource "random_password" "hs256" {
  length  = 48
  special = false
}

resource "random_password" "db" {
  length  = 32
  special = false
}

resource "aws_ssm_parameter" "hs256" {
  name  = "/ztmf/pr/${var.repo}/${var.pr_number}/hs256-secret"
  type  = "SecureString"
  value = random_password.hs256.result
}

resource "aws_ssm_parameter" "db_password" {
  name  = "/ztmf/pr/${var.repo}/${var.pr_number}/db-password"
  type  = "SecureString"
  value = random_password.db.result
}
