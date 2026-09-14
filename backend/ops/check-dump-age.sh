#!/usr/bin/env bash
#
# Daily staleness probe for the monthly database dumps.
#
# Publishes ZTMF/Backup DaysSinceLastDump so the freshness of the backups can
# be alarmed on. This exists as a separate daily job rather than as a heartbeat
# emitted by the dump itself because CloudWatch caps an alarm's total
# evaluation range (period x evaluation_periods) at seven days. A monthly
# success pulse leaves ~29 consecutive days with no datapoint, so no legal
# alarm can distinguish "between scheduled dumps" from "dumps stopped
# happening". A daily gauge of the age of the newest object can.
#
# It reads the bucket rather than trusting a metric the dump job wrote, so it
# also catches an upload that was reported as successful but did not land.
#
# Required environment (supplied by the task definition):
#   DUMP_BUCKET   bucket holding the dumps
#   ENVIRONMENT   dev | prod (key prefix and metric dimension)
#   AWS_REGION    region for the AWS CLI calls

set -euo pipefail

: "${DUMP_BUCKET:?DUMP_BUCKET is required}"
: "${ENVIRONMENT:?ENVIRONMENT is required}"

readonly METRIC_NAMESPACE="ZTMF/Backup"
readonly PREFIX="pgdump/${ENVIRONMENT}/"

log() { echo "[$(date -u +%Y-%m-%dT%H:%M:%SZ)] $*"; }

log "listing s3://${DUMP_BUCKET}/${PREFIX}"

# Only the .dump archives count. A stray globals file without its archive is
# not a usable backup and must not reset the clock.
newest="$(aws s3api list-objects-v2 \
  --bucket "${DUMP_BUCKET}" \
  --prefix "${PREFIX}" \
  --query 'sort_by(Contents[?ends_with(Key, `.dump`)], &LastModified)[-1].LastModified' \
  --output text)"

if [[ -z "${newest}" || "${newest}" == "None" ]]; then
  # An empty bucket is indistinguishable from a bucket whose contents were
  # deleted. Both are reported as maximally stale rather than as missing data,
  # so a brand new environment surfaces immediately instead of looking healthy.
  log "no dumps found under ${PREFIX}; reporting maximum age"
  age_days=9999
else
  # GNU date, from the coreutils package added to the image for this. The
  # busybox date that wolfi-base ships by default does not parse the
  # "+00:00" offset S3 returns in LastModified.
  newest_epoch="$(date -u -d "${newest}" +%s)"
  now_epoch="$(date -u +%s)"
  age_days=$(((now_epoch - newest_epoch) / 86400))
  log "newest dump ${newest}, ${age_days} days old"
fi

aws cloudwatch put-metric-data \
  --namespace "${METRIC_NAMESPACE}" \
  --metric-name DaysSinceLastDump \
  --value "${age_days}" \
  --unit Count \
  --dimensions "Environment=${ENVIRONMENT}"

log "done"
