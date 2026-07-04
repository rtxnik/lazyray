#!/usr/bin/env bash
# Release preflight: verify a version tag is safe to push (local runbook mode)
# or safe to build (CI mode, first job of release.yml).
# Usage: preflight.sh vX.Y.Z[-rc.N]
#   - If the tag exists (CI mode / re-run), checks apply to the tag's target.
#   - If it does not (local pre-tag mode), checks apply to origin/main's tip —
#     the commit about to be tagged.
# Runs ALL checks and reports each one; exits 1 if any failed.
# Requires: git, jq, gh (authenticated, or anonymous against a public repo).
set -uo pipefail

REPO="${GOVERNANCE_REPO:-rtxnik/lazyray}"
HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CHANGELOG="$HERE/../../CHANGELOG.md"
TAG="${1:-}"
FAIL=0

ok()  { echo "ok:   $*"; }
bad() { echo "FAIL: $*"; FAIL=1; }

[ -n "$TAG" ] || { echo "usage: preflight.sh vX.Y.Z[-rc.N]" >&2; exit 1; }

# 1. Tag format: strict semver, optional -rc.N prerelease.
if echo "$TAG" | grep -qE '^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-rc\.[1-9][0-9]*)?$'; then
  ok "tag format ($TAG)"
else
  bad "tag '$TAG' is not vMAJOR.MINOR.PATCH[-rc.N]"
fi

VERSION="${TAG#v}"
case "$TAG" in
  *-rc.*) RC=1; VERSION="${VERSION%%-rc.*}" ;;
  *)      RC=0 ;;
esac

# 2. CHANGELOG carries a dated section for exactly this version.
#    Prereleases are exempt: their section lands with the final release prep.
if [ "$RC" -eq 1 ]; then
  ok "changelog section check skipped for prerelease $TAG"
else
  ver_re="$(printf '%s' "$VERSION" | sed 's/\./\\./g')"
  if grep -qE "^## \[$ver_re\] - [0-9]{4}-[0-9]{2}-[0-9]{2}$" "$CHANGELOG"; then
    ok "CHANGELOG section [$VERSION] present"
  else
    bad "CHANGELOG.md lacks a '## [$VERSION] - YYYY-MM-DD' section"
  fi
fi

# 3. Resolve the commit under test.
git fetch -q origin main 2>/dev/null || true
if git rev-parse -q --verify "refs/tags/$TAG" >/dev/null 2>&1; then
  SHA="$(git rev-list -n1 "$TAG")"
  echo "note: tag $TAG exists; checking its target $SHA"
elif SHA="$(gh api "repos/$REPO/git/ref/tags/$TAG" --jq '.object.sha' 2>/dev/null)"; then
  # Remote tag (CI without local tags). Dereference annotated tag objects.
  otype="$(gh api "repos/$REPO/git/ref/tags/$TAG" --jq '.object.type' 2>/dev/null)"
  if [ "$otype" = "tag" ]; then
    SHA="$(gh api "repos/$REPO/git/tags/$SHA" --jq '.object.sha' 2>/dev/null)"
  fi
  echo "note: remote tag $TAG; checking its target $SHA"
else
  SHA="$(git rev-parse origin/main 2>/dev/null || git rev-parse HEAD)"
  echo "note: tag $TAG not found; checking origin/main tip $SHA (pre-tag mode)"
fi

# 4. That commit is on main.
if MAIN_STATUS="$(gh api "repos/$REPO/compare/main...$SHA" --jq '.status' 2>/dev/null)"; then
  case "$MAIN_STATUS" in
    identical|behind) ok "commit $SHA is on main ($MAIN_STATUS)" ;;
    *) bad "commit $SHA is not on main (compare status: $MAIN_STATUS)" ;;
  esac
else
  bad "cannot compare $SHA with main (network/auth)"
fi

# 5. CI is green there: the aggregate 'CI OK' check-run succeeded.
CONCLUSION="$(gh api "repos/$REPO/commits/$SHA/check-runs?check_name=CI%20OK" \
  --jq '[.check_runs[] | select(.name == "CI OK")][0].conclusion // "absent"' 2>/dev/null || echo unreadable)"
if [ "$CONCLUSION" = "success" ]; then
  ok "CI OK check succeeded on $SHA"
else
  bad "CI OK on $SHA: $CONCLUSION (expected: success)"
fi

# 6. Nothing labeled blocks-release is open.
COUNT="$(gh api "search/issues?q=repo:$REPO+label:blocks-release+state:open" \
  --jq '.total_count' 2>/dev/null || echo unreadable)"
if [ "$COUNT" = "0" ]; then
  ok "no open blocks-release items"
else
  bad "open blocks-release items: $COUNT"
fi

if [ "$FAIL" -ne 0 ]; then
  echo "preflight: FAIL ($TAG)"
  exit 1
fi
echo "preflight: OK ($TAG)"
