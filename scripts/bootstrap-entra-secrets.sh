#!/usr/bin/env bash
#
# Seed the Entra OIDC config and the application session signing key into
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
#   ENTRA_TENANT_ID       Entra tenant GUID
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

command -v jq >/dev/null || { echo "jq is required" >&2; exit 1; }
: "${ENTRA_TENANT_ID:?set ENTRA_TENANT_ID}"
: "${ENTRA_CLIENT_ID:?set ENTRA_CLIENT_ID}"
: "${ENTRA_CLIENT_SECRET:?set ENTRA_CLIENT_SECRET}"
SUFFIX="${SECRET_SUFFIX:-}"

base="https://login.microsoftonline.com/${ENTRA_TENANT_ID}"

# Secret values are written to a locked-down temp file and passed to the AWS CLI
# via file://, never on the argv (where they would be visible in `ps`). The
# trap wipes the file even on error.
workdir=$(mktemp -d)
chmod 700 "$workdir"
trap 'rm -rf "$workdir"' EXIT

entra_file="$workdir/entra.json"

# jq --arg escapes every value correctly, so a client secret containing quotes
# or backslashes cannot corrupt the JSON. Microsoft v2.0 endpoints are derived
# from the tenant id; the /v2.0 issuer suffix is significant and must match the
# iss claim the backend pins.
jq -n \
  --arg base "$base" \
  --arg tid "$ENTRA_TENANT_ID" \
  --arg cid "$ENTRA_CLIENT_ID" \
  --arg secret "$ENTRA_CLIENT_SECRET" \
  '{
    authorization_endpoint: ($base + "/oauth2/v2.0/authorize"),
    token_endpoint:         ($base + "/oauth2/v2.0/token"),
    user_info_endpoint:     "https://graph.microsoft.com/oidc/userinfo",
    issuer:                 ($base + "/v2.0"),
    jwks_uri:               ($base + "/discovery/v2.0/keys"),
    tenant_id:              $tid,
    client_id:              $cid,
    client_secret:          $secret
  }' > "$entra_file"

echo "Seeding ztmf_entra_oidc${SUFFIX}..."
aws secretsmanager put-secret-value \
  --secret-id "ztmf_entra_oidc${SUFFIX}" \
  --secret-string "file://${entra_file}" >/dev/null

# Generate a high-entropy session signing key only if a non-empty one is not
# already set, so re-running does not rotate live sessions. A present-but-empty
# value is treated as unset (an empty HMAC key is unsafe and the backend fails
# closed on it).
existing=$(aws secretsmanager get-secret-value \
  --secret-id "ztmf_session_signing_key${SUFFIX}" \
  --query SecretString --output text 2>/dev/null || true)

if [ -n "$existing" ] && [ "$existing" != "None" ]; then
  echo "ztmf_session_signing_key${SUFFIX} already has a value, leaving it as-is."
else
  echo "Generating and seeding ztmf_session_signing_key${SUFFIX}..."
  key_file="$workdir/signing.key"
  openssl rand -base64 48 | tr -d '\n' > "$key_file"
  aws secretsmanager put-secret-value \
    --secret-id "ztmf_session_signing_key${SUFFIX}" \
    --secret-string "file://${key_file}" >/dev/null
fi

echo "Done. Set entra_enabled = true in the target tfvars and re-apply."
