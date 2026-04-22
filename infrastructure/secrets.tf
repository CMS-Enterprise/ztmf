resource "aws_secretsmanager_secret" "ztmf_va_trust_provider" {
  name = "ztmf_va_trust_provider"
}

# cert and key are the TLS digicert certificate purchased by Elizabeth S.
# initially we tried to use them on the Fargate container but decided to 
# simplify things by just generating self-signed certs during container builds
# leaving them here so they arent stored locally in case we need the value again
resource "aws_secretsmanager_secret" "ztmf_tls_cert" {
  name = "ztmf_tls_cert"
}

resource "aws_secretsmanager_secret" "ztmf_tls_key" {
  name = "ztmf_tls_key"
}

# DB user is only used to create the DB, its value is then copied into the RDS-managed auto-rotated secret
resource "aws_secretsmanager_secret" "ztmf_db_user" {
  name = "ztmf_db_user"
}

# host, port, and credentials for logging in to CMS SMTP service
resource "aws_secretsmanager_secret" "ztmf_smtp" {
  name = "ztmf_smtp"
}

# CA certs for validating TLS connection to SMTP service
resource "aws_secretsmanager_secret" "ztmf_smtp_ca_root" {
  name = "ztmf_smtp_ca_root"
}

resource "aws_secretsmanager_secret" "ztmf_smtp_intermediate" {
  name = "ztmf_smtp_intermediate"
}

# Snowflake credentials for data sync Lambda function (dev environment)
resource "aws_secretsmanager_secret" "ztmf_snowflake_dev" {
  count = var.environment == "dev" ? 1 : 0
  name  = "ztmf_snowflake_dev"

  description = "Snowflake credentials for ZTMF data sync in dev environment"

  tags = {
    Name        = "ZTMF Snowflake Dev Credentials"
    Environment = "dev"
    Purpose     = "Lambda data sync"
  }
}

# Snowflake credentials for data sync Lambda function (prod environment)
resource "aws_secretsmanager_secret" "ztmf_snowflake_prod" {
  count = var.environment == "prod" ? 1 : 0
  name  = "ztmf_snowflake_prod"

  description = "Snowflake credentials for ZTMF data sync in prod environment"

  tags = {
    Name        = "ZTMF Snowflake Prod Credentials"
    Environment = "prod"
    Purpose     = "Lambda data sync"
  }
}

# Kion App API key for the Lambda that rotates it daily (dev environment)
# Seeded manually once per account; thereafter rotated by ztmf-kion-key-rotate-dev
resource "aws_secretsmanager_secret" "ztmf_kion_dev" {
  count = var.environment == "dev" ? 1 : 0
  name  = "ztmf_kion_dev"

  description = "Kion App API key for ZTMF dev account. Rotated daily by ztmf-kion-key-rotate-dev Lambda. Payload: {api_key, base_url, rotated_at}."

  tags = {
    Name        = "ZTMF Kion API Key Dev"
    Environment = "dev"
    Purpose     = "Kion API access"
  }
}

# Kion App API key for the Lambda that rotates it daily (prod environment)
resource "aws_secretsmanager_secret" "ztmf_kion_prod" {
  count = var.environment == "prod" ? 1 : 0
  name  = "ztmf_kion_prod"

  description = "Kion App API key for ZTMF prod account. Rotated daily by ztmf-kion-key-rotate-prod Lambda. Payload: {api_key, base_url, rotated_at}."

  tags = {
    Name        = "ZTMF Kion API Key Prod"
    Environment = "prod"
    Purpose     = "Kion API access"
  }
}

# Slack webhook URL for data sync alerts (shared across environments)
resource "aws_secretsmanager_secret" "ztmf_slack_webhook" {
  name = "ztmf_slack_webhook"

  description = "Slack webhook URL for ZTMF data sync alerts and notifications"

  tags = {
    Name        = "ZTMF Slack Webhook"
    Environment = "shared"
    Purpose     = "Data sync notifications"
  }
}
