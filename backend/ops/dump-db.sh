#!/usr/bin/env bash
#
# Monthly logical backup of the ZTMF Aurora cluster to S3.
#
# Run as a scheduled Fargate task off the same ops image used for interactive
# database access (see infrastructure/backup-dumps.tf). This is deliberately a
# logical dump rather than another snapshot: recovery for a single bad row is
# never "restore the cluster", because every Aurora restore path produces a new
# cluster and rolling production back weeks to fix one row would discard
# everything else. The real procedure is to stand up a reference copy, diff the
# affected rows, and apply a targeted UPDATE, so the artifact's readability
# matters more than its restorability. A pg_dump restores into local docker in
# minutes; a snapshot needs a new cluster.
#
# It is also the only copy with no expiry, which matters given the usage
# pattern: an unnoticed edit can plausibly sit for a year.
#
# Required environment (supplied by the task definition):
#   DB_SECRET_ID  Secrets Manager ARN of the RDS-managed master credential
#   DB_ENDPOINT   cluster writer endpoint
#   DB_PORT       postgres port
#   DB_NAME       database to dump
#   DUMP_BUCKET   destination S3 bucket
#   ENVIRONMENT   dev | prod (key prefix and metric dimension)
#   AWS_REGION    region for the AWS CLI calls

set -euo pipefail

: "${DB_SECRET_ID:?DB_SECRET_ID is required}"
: "${DB_ENDPOINT:?DB_ENDPOINT is required}"
: "${DB_PORT:?DB_PORT is required}"
: "${DB_NAME:?DB_NAME is required}"
: "${DUMP_BUCKET:?DUMP_BUCKET is required}"
: "${ENVIRONMENT:?ENVIRONMENT is required}"

readonly METRIC_NAMESPACE="ZTMF/Backup"

# Assigned then marked readonly separately: `readonly X="$(cmd)"` makes the
# builtin's exit status the one `set -e` sees, so a failing mktemp would leave
# WORKDIR empty and hand `rm -rf ""` to the trap.
WORKDIR="$(mktemp -d)"
readonly WORKDIR
trap 'rm -rf "${WORKDIR}"' EXIT

timestamp="$(date -u +%Y%m%dT%H%M%SZ)"
year="$(date -u +%Y)"
prefix="pgdump/${ENVIRONMENT}/${year}"
dump_file="${WORKDIR}/ztmf-${ENVIRONMENT}-${timestamp}.dump"
globals_file="${WORKDIR}/ztmf-${ENVIRONMENT}-${timestamp}.globals.sql"

log() { echo "[$(date -u +%Y-%m-%dT%H:%M:%SZ)] $*"; }

log "fetching database credentials"
secret_json="$(aws secretsmanager get-secret-value \
  --secret-id "${DB_SECRET_ID}" \
  --query SecretString \
  --output text)"

PGUSER="$(jq -r '.username' <<<"${secret_json}")"
PGPASSWORD="$(jq -r '.password' <<<"${secret_json}")"
export PGUSER PGPASSWORD
unset secret_json

export PGHOST="${DB_ENDPOINT}"
export PGPORT="${DB_PORT}"
# verify-full, not require. `require` encrypts but performs no certificate or
# hostname validation, so it defeats passive sniffing while leaving an in-path
# substitution undetected. The RDS global trust bundle is baked into the image
# and the cluster pins ca_cert_identifier, so full verification costs nothing.
export PGSSLMODE="verify-full"
export PGSSLROOTCERT="/etc/ssl/certs/rds-global-bundle.pem"

log "dumping ${DB_NAME} in custom format"
pg_dump --format=custom --compress=9 --no-owner --no-privileges \
  --file="${dump_file}" "${DB_NAME}"

# Roles and other cluster-wide objects live outside any single database, so a
# database-only dump is not enough to reconstruct a working reference copy.
# --database pins the connection database: pg_dumpall otherwise defaults to
# `postgres`, which is not guaranteed to exist or be reachable by this role.
log "dumping globals"
pg_dumpall --globals-only --no-role-passwords \
  --database="${DB_NAME}" --file="${globals_file}"

# Verify the archive parses before it is uploaded and the local copy is
# discarded. A dump that cannot be listed cannot be restored, and finding that
# out at recovery time defeats the point of having it.
log "verifying archive integrity"
pg_restore --list "${dump_file}" >/dev/null

dump_bytes="$(wc -c <"${dump_file}")"
if [[ "${dump_bytes}" -lt 1024 ]]; then
  log "ERROR: dump is only ${dump_bytes} bytes, refusing to upload"
  exit 1
fi
log "dump is ${dump_bytes} bytes"

log "uploading to s3://${DUMP_BUCKET}/${prefix}/"
aws s3 cp "${dump_file}" "s3://${DUMP_BUCKET}/${prefix}/$(basename "${dump_file}")"
aws s3 cp "${globals_file}" "s3://${DUMP_BUCKET}/${prefix}/$(basename "${globals_file}")"

# Informational only. Staleness is detected by check-dump-age.sh reading the
# bucket daily, not by this pulse, because CloudWatch caps an alarm's total
# evaluation range at seven days and a monthly heartbeat cannot be alarmed on
# across a 40-day window.
#
# Deliberately non-fatal: both objects are already durably in S3 by this point,
# so failing the task over a CloudWatch hiccup would report a backup as missing
# when it is sitting in the bucket.
log "publishing success metric"
aws cloudwatch put-metric-data \
  --namespace "${METRIC_NAMESPACE}" \
  --metric-name DumpSucceeded \
  --value 1 \
  --unit Count \
  --dimensions "Environment=${ENVIRONMENT}" ||
  log "WARN: could not publish DumpSucceeded; the dump itself succeeded"

log "done"
