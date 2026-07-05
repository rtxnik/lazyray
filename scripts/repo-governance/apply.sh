#!/usr/bin/env bash
# Apply the committed repository-governance baseline to the live repo.
# Owner-run: requires gh authenticated with admin rights on the repo.
set -euo pipefail

REPO="${GOVERNANCE_REPO:-rtxnik/lazyray}"
HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
GOV="$HERE/../../.github/governance"

# --- Staleness guard -------------------------------------------------------
# A stale local checkout silently no-ops this apply (it computes no diff versus
# the etalon it was built from) and then check.sh passes against that same
# stale etalon -- a false green. Refuse to apply from a HEAD that differs from
# origin/main unless the operator explicitly overrides (ALLOW_STALE=1 or --force).
ALLOW_STALE="${ALLOW_STALE:-0}"
for arg in "$@"; do
  case "$arg" in
    --force) ALLOW_STALE=1 ;;
    *) echo "unknown argument: $arg" >&2; exit 2 ;;
  esac
done

if git -C "$HERE" rev-parse --git-dir >/dev/null 2>&1; then
  if git -C "$HERE" fetch --quiet origin 2>/dev/null; then
    local_head="$(git -C "$HERE" rev-parse HEAD)"
    remote_head="$(git -C "$HERE" rev-parse origin/main)"
    if [ "$local_head" != "$remote_head" ]; then
      echo "############################################################" >&2
      echo "# STALE CHECKOUT" >&2
      echo "#   HEAD        $local_head" >&2
      echo "#   origin/main $remote_head" >&2
      echo "# Applying from a stale tree can silently no-op and then pass" >&2
      echo "# check.sh against the same stale etalon (a false green)." >&2
      echo "# Refusing. Re-run on an up-to-date main, or override with:" >&2
      echo "#   ALLOW_STALE=1 $0        (or:  $0 --force)" >&2
      echo "############################################################" >&2
      if [ "$ALLOW_STALE" != "1" ]; then
        exit 3
      fi
      echo "ALLOW_STALE override set -- proceeding despite stale HEAD." >&2
    fi
  else
    echo "WARNING: could not fetch origin; staleness not verified." >&2
  fi
fi
# ---------------------------------------------------------------------------

echo "==> Repo settings"
gh api -X PATCH "repos/$REPO" --input "$GOV/settings.json" >/dev/null
echo "settings applied"

echo "==> Vulnerability alerts + automated security fixes"
gh api -X PUT "repos/$REPO/vulnerability-alerts" >/dev/null
gh api -X PUT "repos/$REPO/automated-security-fixes" >/dev/null

echo "==> Rulesets"
if ! rulesets_json="$(gh api "repos/$REPO/rulesets" 2>/dev/null)"; then
  echo "ERROR: cannot list rulesets (auth/network)" >&2
  exit 1
fi
for name in main tags; do
  file="$GOV/rulesets/$name.json"
  rname="$(jq -r .name "$file")"
  id="$(jq -r --arg n "$rname" '.[] | select(.name == $n) | .id' <<<"$rulesets_json")"
  if [ -n "$id" ]; then
    gh api -X PUT "repos/$REPO/rulesets/$id" --input "$file" >/dev/null
    echo "updated ruleset $rname (id $id)"
  else
    gh api -X POST "repos/$REPO/rulesets" --input "$file" >/dev/null
    echo "created ruleset $rname"
  fi
done

echo "==> Release environment (deployment restricted to v* tags)"
printf '{"deployment_branch_policy":{"protected_branches":false,"custom_branch_policies":true}}' \
  | gh api -X PUT "repos/$REPO/environments/release" --input - >/dev/null
if ! gh api "repos/$REPO/environments/release/deployment-branch-policies" \
     --jq '.branch_policies[] | select(.type == "tag") | .name' 2>/dev/null \
     | grep -qxF 'v*'; then
  gh api -X POST "repos/$REPO/environments/release/deployment-branch-policies" \
    -f name='v*' -f type=tag >/dev/null
  echo "created tag deployment policy v*"
fi

cat <<'EOF'
==> MANUAL: move release secrets into the environment (values are owner-held).
Substitute the real key-file path in the first command:
  gh secret set MINISIGN_SECRET_KEY --env release -R rtxnik/lazyray < /path/to/minisign-secret.key
  gh secret set MINISIGN_PASSWORD  --env release -R rtxnik/lazyray
  gh secret delete MINISIGN_SECRET_KEY -R rtxnik/lazyray
  gh secret delete MINISIGN_PASSWORD  -R rtxnik/lazyray
Then verify: scripts/repo-governance/check.sh full
EOF
