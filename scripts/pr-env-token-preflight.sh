#!/usr/bin/env bash
# Token preflight for the PR environment reaper. Run by
# .github/workflows/pr-env-reaper.yml before anything else, and safe to run by
# hand.
#
# The reaper files its tracking issue with the same token it reaps with, so
# once that token stops working it cannot report the failure. This checks the
# token first:
#   - The token must be valid and able to read each repo the reaper uses. A
#     refusal is a ::error:: naming the repo, also written to the report file
#     so the tracking issue names it when the token can still post there. ztmf
#     and ztmf-ui are public, so for them this catches an expired or revoked
#     token but not one that lost access to that repo; Reap fails on that
#     later.
#   - A refusal from ztmf or ztmf-ui exits 1 once all three are checked, since
#     Reap cannot work without those two. A refusal from ztmf-misc alone does
#     not stop the run: Reap only talks to ztmf and ztmf-ui, so the script
#     exits 0 and, in CI, sets the step output misc_refused=true for the
#     workflow's last step to fail the job on.
#   - GitHub returns a PAT's expiry in the
#     github-authentication-token-expiration header. Within TOKEN_WARN_DAYS of
#     that date, a warning goes in the report file, and the workflow's Report
#     step posts it to the tracking issue while the token still works. A
#     header that does not parse is reported the same way. No header means
#     the token has no expiry date; that is logged and passes.
# The report file is empty when there is nothing to warn about.
#
# Usage:
#   scripts/pr-env-token-preflight.sh [--report FILE]
#
# Prereqs: gh on PATH, and GH_TOKEN set to the token to check (in CI,
# PR_ENV_REAPER_TOKEN).

set -euo pipefail

REPORT=/dev/stdout
while [[ $# -gt 0 ]]; do
  case "$1" in
    --report) REPORT="$2"; shift 2 ;;
    *) echo "usage: $0 [--report FILE]" >&2; exit 2 ;;
  esac
done

TOKEN_WARN_DAYS="${TOKEN_WARN_DAYS:-14}"
OWNER="${OWNER:-CMS-Enterprise}"
SECRET="PR_ENV_REAPER_TOKEN"

log() { echo "$*" >&2; }

# GNU date first (CI), then BSD date (a run by hand on macOS).
to_epoch() {
  date -u -d "$1" +%s 2>/dev/null || date -u -j -f '%Y-%m-%d %H:%M:%S %Z' "$1" +%s 2>/dev/null
}

out=$(mktemp)
err=$(mktemp)
trap 'rm -f "$out" "$err"' EXIT

refused=false
blocking=false
misc_refused=false
refusals=""
expiry=""
for repo in ztmf ztmf-ui ztmf-misc; do
  if ! gh api -i "repos/$OWNER/$repo" > "$out" 2> "$err"; then
    reason=$(head -n 1 "$err")
    log "::error::$SECRET was refused by $OWNER/$repo: $reason"
    refusals+="\`$SECRET\` was refused by $OWNER/$repo: $reason"$'\n\n'
    refused=true
    if [[ "$repo" == ztmf-misc ]]; then misc_refused=true; else blocking=true; fi
    continue
  fi
  log "$OWNER/$repo accepts the token"
  if [[ -z "$expiry" ]]; then
    expiry=$(tr -d '\r' < "$out" | sed -n 's/^[Gg]ithub-[Aa]uthentication-[Tt]oken-[Ee]xpiration: *//p' | head -n 1)
  fi
done

report=""
if [[ -n "$expiry" ]]; then
  if expires=$(to_epoch "$expiry"); then
    left=$(( expires - $(date -u +%s) ))
    log "$SECRET expires $expiry, in $(( left / 86400 )) days"
    if (( left <= TOKEN_WARN_DAYS * 86400 )); then
      log "::warning::$SECRET expires $expiry, in $(( left / 86400 )) days."
      report="\`$SECRET\` (a secret in $OWNER/ztmf) expires on ${expiry%% *}, in $(( left / 86400 )) days. Regenerate the fine-grained PAT with access to ztmf, ztmf-ui, and ztmf-misc, and update the secret before then. Once it expires, the reaper fails and cannot report that here."
    fi
  else
    log "::warning::Could not parse the $SECRET expiry \"$expiry\"."
    report="The reaper could not read when \`$SECRET\` (a secret in $OWNER/ztmf) expires: GitHub sent \`$expiry\`. Check the token's expiry by hand."
  fi
elif ! $refused; then
  log "::warning::No github-authentication-token-expiration header, so $SECRET has no expiry date or is not a PAT. Expiry not checked."
fi

{
  printf '%s' "$refusals"
  if [[ -n "$report" ]]; then echo "$report"; fi
} > "$REPORT"
if $blocking; then exit 1; fi
if $misc_refused; then
  log "Only $OWNER/ztmf-misc refused the token. Reap still runs, and the job fails at its end."
  if [[ -n "${GITHUB_OUTPUT:-}" ]]; then echo "misc_refused=true" >> "$GITHUB_OUTPUT"; fi
fi
