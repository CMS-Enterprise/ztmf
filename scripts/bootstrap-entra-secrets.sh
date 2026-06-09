#!/usr/bin/env bash
#
# Seed the HHS Entra OIDC config and the application session signing key into
# AWS Secrets Manager for one ZTMF environment.
#
# Terraform owns the secret CONTAINERS (ztmf_entra_oidc, ztmf_session_signing_key
# in infrastructure/secrets.tf); their VALUES are seeded here, out of band, so no
# tenant id, client secret, or signing key ever lands in the repo or in
# terraform state. Run this once per account after the first `terraform apply`
# (which creates the empty containers) and before flipping entra_enabled = true.
#
# Required environment (provided by the operator, never committed):
#   AWS_PROFILE / AWS credentials for the target account (ztmf-dev or ztmf-prod)
#   ENTRA_TENANT_ID       HHS Entra tenant GUID
#   ENTRA_CLIENT_ID       ZTMF app registration client id
#   ENTRA_CLIENT_SECRET   ZTMF app registration client secret
#
# Optional:
#   SECRET_SUFFIX         "" for dev/prod (default), "_impl" for an impl account
#
# Usage:
#   AWS_PROFILE=ztmf-dev ENTRA_TENANT_ID=... ENTRA_CLIENT_ID=... \
#     ENTRA_CLIENT_SECRET=... ./scripts/bootstrap-entra-secrets.sh

set -euo pipefail

: "${ENTRA_TENANT_ID:?set ENTRA_TENANT_ID}"
: "${ENTRA_CLIENT_ID:?set ENTRA_CLIENT_ID}"
: "${ENTRA_CLIENT_SECRET:?set ENTRA_CLIENT_SECRET}"
SUFFIX="${SECRET_SUFFIX:-}"

base="https://login.microsoftonline.com/${ENTRA_TENANT_ID}"

# Microsoft v2.0 endpoints are derived from the tenant id, so only the tenant
# and client credentials are operator input. The /v2.0 suffix on the issuer is
# significant and must match the iss claim the backend pins.
entra_json=$(cat <<JSON
{
  "authorization_endpoint": "${base}/oauth2/v2.0/authorize",
  "token_endpoint": "${base}/oauth2/v2.0/token",
  "user_info_endpoint": "https://graph.microsoft.com/oidc/userinfo",
  "issuer": "${base}/v2.0",
  "jwks_uri": "${base}/discovery/v2.0/keys",
  "tenant_id": "${ENTRA_TENANT_ID}",
  "client_id": "${ENTRA_CLIENT_ID}",
  "client_secret": "${ENTRA_CLIENT_SECRET}"
}
JSON
)

echo "Seeding ztmf_entra_oidc${SUFFIX}..."
aws secretsmanager put-secret-value \
  --secret-id "ztmf_entra_oidc${SUFFIX}" \
  --secret-string "${entra_json}" >/dev/null

# Generate a high-entropy session signing key only if one is not already set,
# so re-running this script does not rotate live sessions by accident.
if aws secretsmanager get-secret-value --secret-id "ztmf_session_signing_key${SUFFIX}" \
    --query SecretString --output text >/dev/null 2>&1; then
  echo "ztmf_session_signing_key${SUFFIX} already has a value, leaving it as-is."
else
  echo "Generating and seeding ztmf_session_signing_key${SUFFIX}..."
  signing_key=$(openssl rand -base64 48)
  aws secretsmanager put-secret-value \
    --secret-id "ztmf_session_signing_key${SUFFIX}" \
    --secret-string "${signing_key}" >/dev/null
fi

echo "Done. Set entra_enabled = true in the target tfvars and re-apply."
