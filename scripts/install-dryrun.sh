#!/bin/sh
# install-dryrun.sh — acceptance harness for scripts/install.sh.
# Builds snapshot artifacts with goreleaser, serves dist/ over a file:// base,
# and runs install.sh against a throwaway PREFIX, asserting it verifies the
# checksum and installs the lzr binary. minisign is intentionally NOT required:
# --snapshot produces no .minisig, so this exercises the checksum-only path.
set -eu

repo_root=$(CDPATH='' cd -- "$(dirname -- "$0")/.." && pwd)
cd "$repo_root"

# The repo's .goreleaser.yml signs checksums.txt with minisign, so a
# snapshot build needs a secret key even though we never publish it. Generate an
# ephemeral, passwordless key just for this dry-run; it is discarded on exit and
# never touches the embedded RELEASE_PUBKEYS trust-list the script verifies against.
keydir=$(mktemp -d)
trap 'rm -rf "$keydir" "${prefix:-}"' EXIT
minisign -G -W -p "$keydir/k.pub" -s "$keydir/k.key" >/dev/null
MINISIGN_SECRET_KEY_FILE="$keydir/k.key"
MINISIGN_PASSWORD=""
export MINISIGN_SECRET_KEY_FILE MINISIGN_PASSWORD

echo "==> building snapshot artifacts"
rm -rf dist
goreleaser release --snapshot --clean --skip=publish

# Resolve the snapshot version goreleaser stamped (e.g. 0.8.0-next or 0.8.1-SNAPSHOT-<sha>).
ver=$(goreleaser release --snapshot --clean --skip=publish 2>/dev/null >/dev/null; \
      sed -n 's/.*"version": *"\([^"]*\)".*/\1/p' dist/metadata.json | head -n1)
[ -n "$ver" ] || { echo "FAIL: could not read snapshot version from dist/metadata.json"; exit 1; }

# The snapshot's checksums.txt.minisig is signed by the ephemeral key above, NOT
# by any key in the release RELEASE_PUBKEYS trust-list embedded in install.sh, so a real signature
# check could not pass. Remove it so the script takes the documented checksum-only
# graceful-degradation path regardless of whether minisign is installed on this
# host. The full-signature path is proven by the Go round-trip tests and the CI
# minisign snapshot gate.
rm -f dist/checksums.txt.minisig

os=$(uname -s | tr '[:upper:]' '[:lower:]')
arch=$(uname -m)
case "$arch" in
	x86_64|amd64) arch=amd64 ;;
	aarch64|arm64) arch=arm64 ;;
esac

archive="lazyray_${ver}_${os}_${arch}.tar.gz"
[ -f "dist/${archive}" ] || { echo "FAIL: expected dist/${archive}, found:"; ls dist; exit 1; }

prefix=$(mktemp -d)

echo "==> running install.sh (checksum-only path) into $prefix"
LAZYRAY_VERSION="$ver" \
LAZYRAY_BASE_URL="file://${repo_root}/dist" \
PREFIX="$prefix" \
	sh scripts/install.sh

[ -x "$prefix/bin/lzr" ] || { echo "FAIL: lzr not installed at $prefix/bin/lzr"; exit 1; }
"$prefix/bin/lzr" --version >/dev/null 2>&1 || { echo "FAIL: installed lzr is not runnable"; exit 1; }

echo "PASS: install.sh verified checksum and installed lzr to $prefix/bin/lzr"
