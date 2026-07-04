#!/usr/bin/env bash
# Verify live repo governance matches the committed etalon.
# Usage: check.sh [ci|full]
#   ci   - rulesets only; anonymous REST reads (public repo), needs no token
#   full - also settings, release environment, secret homing, vuln alerts
#          (requires owner-authenticated gh)
# Exit codes: 0 = OK, 1 = drift detected, 2 = could not verify (read failure).
# Confirmed drift outranks read noise: if both occur, the exit code is 1.
# Note: bypass_actors are NOT visible to anonymous reads (the API returns
# null); ci mode therefore checks everything except bypass_actors, and full
# mode verifies them exactly via authenticated gh.
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
# curl --retry covers transient 429/5xx; a 403 (rate limit or permissions) is
# surfaced distinctly instead of hiding inside a generic curl failure.
fetch() {
  local url="$1" out code body
  if ! out="$(curl -sS --retry 3 --retry-delay 30 -w '\n%{http_code}' \
        -H "Accept: application/vnd.github+json" "$url")"; then
    echo "curl failure for $url" >&2
    return 1
  fi
  code="${out##*$'\n'}"
  body="${out%$'\n'*}"
  case "$code" in
    200) printf '%s' "$body"; return 0 ;;
    403|429) echo "HTTP $code for $url (rate limit or forbidden)" >&2; return 1 ;;
    *) echo "HTTP $code for $url" >&2; return 1 ;;
  esac
}

if ! RULESETS="$(fetch "$API/rulesets?per_page=100")"; then
  echo "ERROR: cannot read rulesets list for $REPO (network/HTTP failure)" >&2
  exit 2
fi
if [ "$(jq 'length' <<<"$RULESETS")" -ge 100 ]; then
  rerr "rulesets list hit the per_page=100 ceiling; add pagination before trusting this check"
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
  want="$(jq -S 'del(.bypass_actors)' "$file")"
  if ! jq -e --argjson want "$want" '
      {name, target, enforcement, conditions, rules}
      | contains($want)
        and (.conditions == $want.conditions)
        and ((.rules | length) == ($want.rules | length))
        and (.enforcement == "active")
    ' <<<"$live" >/dev/null; then
    err "ruleset '$rname' diverges from $file"
  fi

  # bypass_actors are redacted (null) on anonymous reads; verify them
  # exactly, authenticated, in full mode only.
  if [ "$MODE" = "full" ]; then
    local want_b live_b
    want_b="$(jq -S '.bypass_actors' "$file")"
    if ! live_b="$(gh api "repos/$REPO/rulesets/$id" --jq '.bypass_actors // []' 2>/dev/null | jq -S .)"; then
      rerr "cannot read bypass_actors for ruleset '$rname' (auth?)"
      return
    fi
    if [ "$live_b" != "$want_b" ]; then
      err "ruleset '$rname' bypass_actors diverge (want=$want_b live=$live_b)"
    fi
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

  # required_status_checks must match EXACTLY (context AND integration_id),
  # in both directions: a live check added on top of the etalon is drift too.
  local want_rsc live_rsc
  if want_rsc="$(jq -e -S '.rules[] | select(.type == "required_status_checks")
        | .parameters.required_status_checks | sort_by(.context)' "$file" 2>/dev/null)"; then
    live_rsc="$(jq -S '.rules[] | select(.type == "required_status_checks")
        | .parameters.required_status_checks | sort_by(.context)' <<<"$live")"
    if [ "$live_rsc" != "$want_rsc" ]; then
      err "ruleset '$rname' required_status_checks diverge (want=$(jq -c . <<<"$want_rsc") live=$(jq -c . <<<"$live_rsc"))"
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

# Confirmed drift outranks read noise (FAIL wins over ERR): automation must
# treat "we saw drift" as the stronger signal even if some reads also failed.
if [ "$FAIL" -ne 0 ]; then
  echo "repo-governance check: FAIL"
  exit 1
fi
if [ "$ERR" -ne 0 ]; then
  echo "repo-governance check: ERROR (could not verify)"
  exit 2
fi
echo "repo-governance check: OK ($MODE)"
