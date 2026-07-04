#!/usr/bin/env bash
# Sync repository labels from the committed manifest.
# Idempotent and additive-only: creates manifest labels that are missing,
# updates color/description of manifest labels that exist, NEVER deletes or
# renames labels absent from the manifest (stock labels survive).
# Owner-run: requires gh authenticated with push rights on the repo.
set -euo pipefail

REPO="${GOVERNANCE_REPO:-rtxnik/lazyray}"
HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
MANIFEST="$HERE/../../.github/governance/labels.json"

# --paginate emits one JSON array per page; jq -s 'add' merges them.
existing="$(gh api "repos/$REPO/labels" --paginate --jq '[.[].name]' | jq -s 'add')"

jq -c '.[]' "$MANIFEST" | while IFS= read -r row; do
  name="$(jq -r '.name' <<<"$row")"
  color="$(jq -r '.color' <<<"$row")"
  desc="$(jq -r '.description' <<<"$row")"
  encoded="$(jq -rn --arg n "$name" '$n|@uri')"
  if jq -e --arg n "$name" 'index($n)' <<<"$existing" >/dev/null; then
    gh api -X PATCH "repos/$REPO/labels/$encoded" \
      -f new_name="$name" -f color="$color" -f description="$desc" >/dev/null
    echo "updated: $name"
  else
    gh api -X POST "repos/$REPO/labels" \
      -f name="$name" -f color="$color" -f description="$desc" >/dev/null
    echo "created: $name"
  fi
done
echo "label sync complete"
