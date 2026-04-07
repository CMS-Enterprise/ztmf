# OIDC configuration for ALB authentication (each environment has its own Okta app)
resource "aws_secretsmanager_secret" "ztmf_va_trust_provider" {
  name = "${local.secret_prefix}_va_trust_provider"
}

# cert and key are the TLS digicert certificate purchased by Elizabeth S.
# initially we tried to use them on the Fargate container but decided to
# simplify things by just generating self-signed certs during container builds
# leaving them here so they arent stored locally in case we need the value again
resource "aws_secretsmanager_secret" "ztmf_tls_cert" {
  name = "${local.secret_prefix}_tls_cert"
}

resource "aws_secretsmanager_secret" "ztmf_tls_key" {
  name = "${local.secret_prefix}_tls_key"
}

# DB user is only used to create the DB, its value is then copied into the RDS-managed auto-rotated secret
resource "aws_secretsmanager_secret" "ztmf_db_user" {
  name = "${local.secret_prefix}_db_user"
}

# host, port, and credentials for logging in to CMS SMTP service
resource "aws_secretsmanager_secret" "ztmf_smtp" {
  name = "${local.secret_prefix}_smtp"
}

# CA certs for validating TLS connection to SMTP service
resource "aws_secretsmanager_secret" "ztmf_smtp_ca_root" {
  name = "${local.secret_prefix}_smtp_ca_root"
}

resource "aws_secretsmanager_secret" "ztmf_smtp_intermediate" {
  name = "${local.secret_prefix}_smtp_intermediate"
}

# Snowflake credentials for data sync Lambda function
resource "aws_secretsmanager_secret" "ztmf_snowflake" {
  name = "ztmf_snowflake_${var.environment}"

  description = "Snowflake credentials for ZTMF data sync in ${var.environment} environment"

  tags = {
    Name        = "ZTMF Snowflake ${title(var.environment)} Credentials"
    Environment = var.environment
    Purpose     = "Lambda data sync"
  }
}

# Slack webhook URL for data sync alerts
resource "aws_secretsmanager_secret" "ztmf_slack_webhook" {
  name = "${local.secret_prefix}_slack_webhook"

  description = "Slack webhook URL for ZTMF data sync alerts and notifications"

  tags = {
    Name        = "ZTMF Slack Webhook"
    Environment = var.environment
    Purpose     = "Data sync notifications"
  }
}
