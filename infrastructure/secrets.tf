// Per-env OIDC config for ALB authenticate-oidc actions. Suffix-renamed for
// impl so the dev account holds two distinct secrets (ztmf_va_trust_provider
// for dev, ztmf_va_trust_provider_impl for impl) without state collision.
// First apply for any env: terraform creates the empty secret; operator
// seeds the OIDC JSON; re-apply consumes it via the data source below.
// Same two-phase bootstrap dev went through.
resource "aws_secretsmanager_secret" "ztmf_va_trust_provider" {
  name = "ztmf_va_trust_provider${local.underscore_sfx}"

  // Operator-seeded OIDC JSON is not in terraform state. A `terraform
  // destroy` (or accidental resource removal) would orphan the credentials
  // and break the ALB OIDC handshake until reseeded. Block the destroy.
  lifecycle {
    prevent_destroy = true
  }
}

# cert and key are the TLS digicert certificate purchased by Elizabeth S.
# initially we tried to use them on the Fargate container but decided to
# simplify things by just generating self-signed certs during container builds
# leaving them here so they arent stored locally in case we need the value again
#
# Account-singletons; impl reuses dev's via data sources rather than racing
# state ownership.
resource "aws_secretsmanager_secret" "ztmf_tls_cert" {
  count = local.manage_account_singletons ? 1 : 0
  name  = "ztmf_tls_cert"
}

resource "aws_secretsmanager_secret" "ztmf_tls_key" {
  count = local.manage_account_singletons ? 1 : 0
  name  = "ztmf_tls_key"
}

# DB user holds the master_username for the Aurora cluster. The password
# itself is auto-generated and auto-rotated by RDS via
# manage_master_user_password=true on the cluster, so this secret is touched
# once at bootstrap and never again.
# Suffix-renamed for impl: each env's Aurora cluster has its own seeded
# username. First apply for any env: terraform creates the empty secret;
# operator seeds a username string; re-apply creates the cluster.
resource "aws_secretsmanager_secret" "ztmf_db_user" {
  name = "ztmf_db_user${local.underscore_sfx}"

  // Aurora master_username is set at cluster creation and cannot be
  // changed afterward. Losing this secret would orphan the value and
  // make rotation/recovery messy. Block accidental destroy.
  lifecycle {
    prevent_destroy = true
  }
}

# host, port, and credentials for logging in to CMS SMTP service.
# Account-singleton; impl reuses dev's CMS SMTP credentials.
resource "aws_secretsmanager_secret" "ztmf_smtp" {
  count = local.manage_account_singletons ? 1 : 0
  name  = "ztmf_smtp"
}

# CA certs for validating TLS connection to SMTP service. Account-singletons.
resource "aws_secretsmanager_secret" "ztmf_smtp_ca_root" {
  count = local.manage_account_singletons ? 1 : 0
  name  = "ztmf_smtp_ca_root"
}

resource "aws_secretsmanager_secret" "ztmf_smtp_intermediate" {
  count = local.manage_account_singletons ? 1 : 0
  name  = "ztmf_smtp_intermediate"
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

# Slack webhook URL for data sync alerts. Account-singleton: impl reuses
# dev's webhook via the data source in data.tf. Single channel covers both
# environments since the alert source already includes ENVIRONMENT in the
# message payload.
resource "aws_secretsmanager_secret" "ztmf_slack_webhook" {
  count = local.manage_account_singletons ? 1 : 0
  name  = "ztmf_slack_webhook"

  description = "Slack webhook URL for ZTMF data sync alerts and notifications"

  tags = {
    Name        = "ZTMF Slack Webhook"
    Environment = "shared"
    Purpose     = "Data sync notifications"
  }
}
