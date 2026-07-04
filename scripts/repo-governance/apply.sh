#!/usr/bin/env bash
# Apply the committed repository-governance baseline to the live repo.
# Owner-run: requires gh authenticated with admin rights on the repo.
set -euo pipefail

REPO="${GOVERNANCE_REPO:-rtxnik/lazyray}"
HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
GOV="$HERE/../../.github/governance"

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
