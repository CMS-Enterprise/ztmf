#!/usr/bin/env bash
# Nightly backstop for per-PR environments (ztmf-misc#344, epic ztmf-misc#321).
# Teardown on PR close is the primary cleanup; this catches what it missed and
# reports what it cannot fix. Run by .github/workflows/pr-env-reaper.yml, and
# safe to run by hand with --dry-run.
#
# For every pr/<repo>/<n> state key in the dev state bucket:
#   - PR closed: destroy the pr-env root (one retry), remove the state object,
#     delete the PR's images, retire its GitHub Environment.
#   - PR open but idle (no deployment for TTL_DAYS): the same, plus a comment
#     on the PR saying the next push rebuilds it.
# Then sweep pr-* GitHub Environments in both repos whose PR is closed, and
# audit for orphans: a pr-* ECS service with no state key, a state key whose
# destroy failed, and an open PR's state key with no running service. Each
# orphan is one line in the report file, which the workflow turns into the
# tracking issue; an empty report means a clean run.
#
# Usage:
#   scripts/pr-env-reaper.sh [--dry-run] [--report FILE]
#
# Prereqs:
#   - AWS credentials for the dev account (the script does not set
#     AWS_PROFILE), terraform, gh, and jq on PATH.
#   - GH_TOKEN (or a gh login) that can read PRs and, outside --dry-run,
#     comment on them and manage environments and deployments in ztmf and
#     ztmf-ui. In CI that is PR_ENV_REAPER_TOKEN.

set -euo pipefail

DRY_RUN=false
REPORT=/dev/stdout
while [[ $# -gt 0 ]]; do
  case "$1" in
    --dry-run) DRY_RUN=true; shift ;;
    --report) REPORT="$2"; shift 2 ;;
    *) echo "usage: $0 [--dry-run] [--report FILE]" >&2; exit 2 ;;
  esac
done

TTL_DAYS="${TTL_DAYS:-7}"
STATE_BUCKET="${STATE_BUCKET:-ztmf-terraform-state-use1-dev}"
AWS_REGION="${AWS_REGION:-us-east-1}"
CLUSTER="${CLUSTER:-ztmf}"
OWNER="${OWNER:-CMS-Enterprise}"
TF_ROOT="$(cd "$(dirname "$0")/../infrastructure/pr-env" && pwd)"

findings=$(mktemp)
trap 'rm -f "$findings"' EXIT
finding() { echo "- $*" >> "$findings"; echo "FINDING: $*" >&2; }
log() { echo "$*" >&2; }
run() {
  if $DRY_RUN; then log "  [dry-run] $*"; else "$@"; fi
}

# ISO-8601 UTC strings compare lexicographically, so the cutoff is a string.
if cutoff=$(date -u -d "-${TTL_DAYS} days" +%Y-%m-%dT%H:%M:%SZ 2>/dev/null); then :; else
  cutoff=$(date -u -v-"${TTL_DAYS}"d +%Y-%m-%dT%H:%M:%SZ)
fi

gh_repo() { case "$1" in ztmf) echo "$OWNER/ztmf" ;; ui) echo "$OWNER/ztmf-ui" ;; esac; }
ecr_repo() { case "$1" in ztmf) echo "ztmf/api-pr" ;; ui) echo "ztmf/ui" ;; esac; }

# The tag values only shape the task definition being destroyed; these
# placeholders satisfy the root's validation for each repo's shape.
placeholder_vars() {
  case "$1" in
    ztmf) echo "-var api_image_tag=pr-ztmf-$2-0000000 -var ui_image_tag=ui-0000000" ;;
    ui) echo "-var api_image_tag=main-0000000 -var ui_image_tag=pr-ui-$2-0000000" ;;
  esac
}

destroy_env() {
  local repo="$1" n="$2" key="pr/$1/$2"
  # shellcheck disable=SC2046
  run terraform -chdir="$TF_ROOT" init -input=false -reconfigure \
    -backend-config="bucket=$STATE_BUCKET" \
    -backend-config="key=$key" \
    -backend-config="region=$AWS_REGION" >&2 || return 1
  local attempt
  for attempt in 1 2; do
    # shellcheck disable=SC2046
    if run terraform -chdir="$TF_ROOT" destroy -input=false -no-color -auto-approve \
      -var "repo=$repo" -var "pr_number=$n" $(placeholder_vars "$repo" "$n") >&2; then
      run aws s3 rm "s3://$STATE_BUCKET/$key" >&2
      return 0
    fi
    log "  destroy attempt $attempt for $key failed"
  done
  return 1
}

delete_images() {
  local repo="$1" n="$2" ecr ids
  ecr=$(ecr_repo "$repo")
  ids=$(aws ecr list-images --repository-name "$ecr" --filter tagStatus=TAGGED \
    --query "imageIds[?starts_with(imageTag, 'pr-$repo-$n-')]" --output json)
  if [[ "$ids" == "[]" ]]; then return 0; fi
  run aws ecr batch-delete-image --repository-name "$ecr" --image-ids "$ids" >/dev/null
}

# Deployments go inactive first so the PR's View deployment button stops
# pointing at a dead URL, then the environment itself is deleted.
retire_github_env() {
  local ghr="$1" env="$2" id
  if ! gh api "repos/$ghr/environments/$env" >/dev/null 2>&1; then return 0; fi
  for id in $(gh api "repos/$ghr/deployments?environment=$env&per_page=100" --jq '.[].id'); do
    run gh api "repos/$ghr/deployments/$id/statuses" -f state=inactive >/dev/null
  done
  run gh api -X DELETE "repos/$ghr/environments/$env"
}

reap() {
  local repo="$1" n="$2" reason="$3" ghr
  ghr=$(gh_repo "$repo")
  log "reaping pr-$repo-$n ($reason)"
  if ! destroy_env "$repo" "$n"; then
    finding "\`pr/$repo/$n\` ($ghr#$n, $reason): destroy failed twice; the environment is still running. Manual steps are in infrastructure/pr-env/README.md."
    return 0
  fi
  delete_images "$repo" "$n" || finding "\`pr/$repo/$n\`: destroyed, but deleting its images from $(ecr_repo "$repo") failed."
  retire_github_env "$ghr" "pr-$repo-$n" || finding "\`pr/$repo/$n\`: destroyed, but retiring the pr-$repo-$n GitHub Environment in $ghr failed."
  if [[ "$reason" == idle* ]]; then
    run gh api "repos/$ghr/issues/$n/comments" \
      -f body="This PR's environment was removed by the nightly reaper after ${TTL_DAYS} days without a deploy. The next push to the PR rebuilds it." >/dev/null \
      || finding "\`pr/$repo/$n\`: reaped as idle, but the comment on $ghr#$n failed."
  fi
}

# 1. State keys: the source of truth for what Terraform thinks is running.
keys=$(aws s3api list-objects-v2 --bucket "$STATE_BUCKET" --prefix pr/ \
  --query 'Contents[].Key' --output text 2>/dev/null | tr '\t' '\n' | grep -E '^pr/(ztmf|ui)/[0-9]+$' || true)
services=$(aws ecs list-services --cluster "$CLUSTER" --query 'serviceArns' --output text \
  | tr '\t' '\n' | sed 's#.*/##' | grep -E '^pr-(ztmf|ui)-[0-9]+$' || true)

for key in $keys; do
  repo=$(echo "$key" | cut -d/ -f2)
  n=$(echo "$key" | cut -d/ -f3)
  ghr=$(gh_repo "$repo")
  state=$(gh api "repos/$ghr/pulls/$n" --jq .state)
  if [[ "$state" == "closed" ]]; then
    reap "$repo" "$n" "PR closed"
    continue
  fi
  last=$(gh api "repos/$ghr/deployments?environment=pr-$repo-$n&per_page=1" --jq '.[0].created_at // empty')
  if [[ -z "$last" ]]; then
    last=$(gh api "repos/$ghr/pulls/$n" --jq .updated_at)
  fi
  if [[ "$last" < "$cutoff" ]]; then
    reap "$repo" "$n" "idle since $last"
    continue
  fi
  if ! echo "$services" | grep -qx "pr-$repo-$n"; then
    finding "\`pr/$repo/$n\` ($ghr#$n, open): state exists but no pr-$repo-$n ECS service is running; the last deploy likely failed partway."
  fi
  log "keeping pr-$repo-$n (open, last deploy $last)"
done

# 2. ECS services with no state key: Terraform cannot destroy what it has no
#    state for, so these are reported, never deleted blind.
for svc in $services; do
  repo=$(echo "$svc" | cut -d- -f2)
  n=$(echo "$svc" | cut -d- -f3)
  if ! echo "$keys" | grep -qx "pr/$repo/$n"; then
    finding "ECS service \`$svc\` in cluster $CLUSTER has no state key \`pr/$repo/$n\`; clean it up by hand (service, task definition, target group, listener rule, log group, SSM parameters, roles)."
  fi
done

# 3. GitHub Environments whose PR is closed, left by a teardown that failed
#    before the retire step or ran without the admin token.
for repo in ztmf ui; do
  ghr=$(gh_repo "$repo")
  envs=$(gh api "repos/$ghr/environments?per_page=100" --jq '.environments[].name' \
    | grep -E "^pr-$repo-[0-9]+$" || true)
  for env in $envs; do
    n=${env##*-}
    if [[ "$(gh api "repos/$ghr/pulls/$n" --jq .state)" == "closed" ]]; then
      log "retiring GitHub Environment $env in $ghr (PR closed)"
      retire_github_env "$ghr" "$env" || finding "GitHub Environment \`$env\` in $ghr: PR is closed but retiring it failed."
    fi
  done
done

cat "$findings" > "$REPORT"
