#!/usr/bin/env bash
# Verify live repo governance matches the committed etalon.
# Usage: check.sh [ci|full]
#   ci   - rulesets only (works with the read-only Actions token)
#   full - also settings, release environment, secret homing, vuln alerts
set -uo pipefail

MODE="${1:-full}"
REPO="${GOVERNANCE_REPO:-rtxnik/lazyray}"
HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
GOV="$HERE/../../.github/governance"
FAIL=0

err() { echo "DRIFT: $*" >&2; FAIL=1; }

check_ruleset() {
  local file="$1" rname id live want
  rname="$(jq -r .name "$file")"
  id="$(gh api "repos/$REPO/rulesets" --jq ".[] | select(.name == \"$rname\") | .id" 2>/dev/null)"
  if [ -z "$id" ]; then
    err "ruleset '$rname' missing"
    return
  fi
  live="$(gh api "repos/$REPO/rulesets/$id")"
  want="$(jq -S . "$file")"
  if ! jq -e --argjson want "$want" '
      {name, target, enforcement, conditions, rules, bypass_actors}
      | contains($want)
        and ((.rules | length) == ($want.rules | length))
        and ((.bypass_actors | length) == ($want.bypass_actors | length))
        and (.enforcement == "active")
    ' <<<"$live" >/dev/null; then
    err "ruleset '$rname' diverges from $file"
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

  env_secrets="$(gh api "repos/$REPO/environments/release/secrets" \
    --jq '[.secrets[].name]' 2>/dev/null || echo '[]')"
  repo_secrets="$(gh api "repos/$REPO/actions/secrets" --jq '[.secrets[].name]')"
  for s in MINISIGN_SECRET_KEY MINISIGN_PASSWORD; do
    jq -e --arg s "$s" 'index($s)' <<<"$env_secrets" >/dev/null \
      || err "secret $s not in environment 'release'"
    jq -e --arg s "$s" 'index($s)' <<<"$repo_secrets" >/dev/null \
      && err "secret $s still at repo level"
  done

  gh api "repos/$REPO/vulnerability-alerts" >/dev/null 2>&1 \
    || err "vulnerability alerts disabled"
fi

if [ "$FAIL" -ne 0 ]; then
  echo "repo-governance check: FAIL"
  exit 1
fi
echo "repo-governance check: OK ($MODE)"
