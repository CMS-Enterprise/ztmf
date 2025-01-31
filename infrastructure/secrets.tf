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

resource "aws_secretsmanager_secret" "ztmf_db_user" {
  name = "ztmf_db_user"
}
