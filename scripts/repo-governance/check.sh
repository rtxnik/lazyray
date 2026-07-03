#!/usr/bin/env bash
# Verify live repo governance matches the committed etalon.
# Usage: check.sh [ci|full]
#   ci   - rulesets only; anonymous REST reads (public repo), needs no token
#   full - also settings, release environment, secret homing, vuln alerts
#          (requires owner-authenticated gh)
# Exit codes: 0 = OK, 1 = drift detected, 2 = could not verify (read failure)
set -uo pipefail

MODE="${1:-full}"
REPO="${GOVERNANCE_REPO:-rtxnik/lazyray}"
API="https://api.github.com/repos/$REPO"
HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
GOV="$HERE/../../.github/governance"
FAIL=0
ERR=0

err() { echo "DRIFT: $*" >&2; FAIL=1; }
rerr() { echo "ERROR: $*" >&2; ERR=1; }

# Rulesets on public repos are readable anonymously. Plain curl sidesteps the
# Actions-token permission question entirely (installation tokens 403 on the
# rulesets endpoints regardless of repo visibility). A fetch failure is
# reported as an ERROR (exit 2), never conflated with "ruleset missing".
fetch() { curl -fsS --retry 3 --retry-delay 30 -H "Accept: application/vnd.github+json" "$1"; }

if ! RULESETS="$(fetch "$API/rulesets")"; then
  echo "ERROR: cannot read rulesets list for $REPO (network/HTTP failure)" >&2
  exit 2
fi

check_ruleset() {
  local file="$1" rname id live want
  rname="$(jq -r .name "$file")"
  id="$(jq -r --arg n "$rname" '.[] | select(.name == $n) | .id' <<<"$RULESETS")"
  if [ -z "$id" ]; then
    err "ruleset '$rname' missing"
    return
  fi
  if ! live="$(fetch "$API/rulesets/$id")"; then
    rerr "cannot read ruleset '$rname' (id $id)"
    return
  fi
  want="$(jq -S . "$file")"
  if ! jq -e --argjson want "$want" '
      {name, target, enforcement, conditions, rules, bypass_actors}
      | contains($want)
        and (.conditions == $want.conditions)
        and ((.rules | length) == ($want.rules | length))
        and ((.bypass_actors | length) == ($want.bypass_actors | length))
        and (.enforcement == "active")
    ' <<<"$live" >/dev/null; then
    err "ruleset '$rname' diverges from $file"
  fi

  # jq `contains` is subset-based: it lets a live array gain elements over
  # the etalon and still "contain" it. Nested arrays need exact-match
  # guards on top of the checks above.
  local want_incl live_incl want_excl live_excl want_pr live_pr
  want_incl="$(jq -S '.conditions.ref_name.include' "$file")"
  live_incl="$(jq -S '.conditions.ref_name.include' <<<"$live")"
  if [ "$live_incl" != "$want_incl" ]; then
    err "ruleset '$rname' conditions.ref_name.include diverges (want=$want_incl live=$live_incl)"
  fi
  want_excl="$(jq -S '.conditions.ref_name.exclude' "$file")"
  live_excl="$(jq -S '.conditions.ref_name.exclude' <<<"$live")"
  if [ "$live_excl" != "$want_excl" ]; then
    err "ruleset '$rname' conditions.ref_name.exclude diverges (want=$want_excl live=$live_excl)"
  fi

  if want_pr="$(jq -e -S '.rules[] | select(.type == "pull_request") | .parameters.allowed_merge_methods' "$file" 2>/dev/null)"; then
    live_pr="$(jq -S '.rules[] | select(.type == "pull_request") | .parameters.allowed_merge_methods' <<<"$live")"
    if [ "$live_pr" != "$want_pr" ]; then
      err "ruleset '$rname' pull_request.allowed_merge_methods diverges (want=$want_pr live=$live_pr)"
    fi
  fi
}

check_ruleset "$GOV/rulesets/main.json"
check_ruleset "$GOV/rulesets/tags.json"

if [ "$MODE" = "full" ]; then
  live="$(gh api "repos/$REPO" --jq '{allow_squash_merge, allow_merge_commit,
    allow_rebase_merge, delete_branch_on_merge, has_discussions, has_issues,
    squash_merge_commit_title, squash_merge_commit_message}' | jq -S .)"
  want="$(jq -S . "$GOV/settings.json")"
  if [ "$live" != "$want" ]; then
    err "repo settings diverge"
    diff <(echo "$want") <(echo "$live") >&2 || true
  fi

  if ! gh api "repos/$REPO/environments/release" >/dev/null 2>&1; then
    err "environment 'release' missing"
  elif ! gh api "repos/$REPO/environments/release/deployment-branch-policies" \
        --jq '.branch_policies[] | select(.type == "tag") | .name' 2>/dev/null \
        | grep -qxF 'v*'; then
    err "environment 'release' lacks tag deployment policy v*"
  fi

  if ! env_secrets="$(gh api "repos/$REPO/environments/release/secrets" \
      --jq '[.secrets[].name]' 2>/dev/null)"; then
    err "cannot read environment secrets (missing environment or auth)"
    env_secrets='[]'
  fi
  if ! repo_secrets="$(gh api "repos/$REPO/actions/secrets" \
      --jq '[.secrets[].name]' 2>/dev/null)"; then
    err "cannot read repo-level secrets (auth?)"
    repo_secrets='[]'
  fi
  repo_secrets="${repo_secrets:-[]}"
  for s in MINISIGN_SECRET_KEY MINISIGN_PASSWORD; do
    jq -e --arg s "$s" 'index($s)' <<<"$env_secrets" >/dev/null \
      || err "secret $s not in environment 'release'"
    jq -e --arg s "$s" 'index($s)' <<<"$repo_secrets" >/dev/null \
      && err "secret $s still at repo level"
  done

  gh api "repos/$REPO/vulnerability-alerts" >/dev/null 2>&1 \
    || err "vulnerability alerts disabled"
fi

if [ "$ERR" -ne 0 ]; then
  echo "repo-governance check: ERROR (could not verify)"
  exit 2
fi
if [ "$FAIL" -ne 0 ]; then
  echo "repo-governance check: FAIL"
  exit 1
fi
echo "repo-governance check: OK ($MODE)"
