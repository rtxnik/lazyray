#!/bin/sh
# install-dryrun.sh — acceptance harness for scripts/install.sh.
#
# Builds snapshot artifacts once, then drives install.sh through every trust
# scenario the fail-closed contract must satisfy. Signatures the scenarios need
# are minted with ephemeral keys; the embedded production trust-list is NEVER
# altered except on a throwaway COPY of the script (scenarios 3/4/5/6/7).
set -eu

repo_root=$(CDPATH='' cd -- "$(dirname -- "$0")/.." && pwd)
cd "$repo_root"

keydir=$(mktemp -d)
work=$(mktemp -d)
trap 'rm -rf "$keydir" "$work"' EXIT

# Ephemeral, passwordless signer keys: A signs the snapshot (via goreleaser),
# B is the second signer used by the 2-of-2 scenarios.
minisign -G -W -p "$keydir/a.pub" -s "$keydir/a.key" >/dev/null
minisign -G -W -p "$keydir/b.pub" -s "$keydir/b.key" >/dev/null
pubkey_a=$(sed -n 2p "$keydir/a.pub")
pubkey_b=$(sed -n 2p "$keydir/b.pub")

# goreleaser reads these to sign checksums.txt (with key A).
MINISIGN_SECRET_KEY_FILE="$keydir/a.key"
MINISIGN_PASSWORD=""
export MINISIGN_SECRET_KEY_FILE MINISIGN_PASSWORD

echo "==> building snapshot artifacts"
rm -rf dist
goreleaser release --snapshot --clean --skip=publish,sbom >/dev/null

ver=$(sed -n 's/.*"version": *"\([^"]*\)".*/\1/p' dist/metadata.json | head -n1)
[ -n "$ver" ] || { echo "FAIL: could not read snapshot version from dist/metadata.json"; exit 1; }

os=$(uname -s | tr '[:upper:]' '[:lower:]')
arch=$(uname -m)
case "$arch" in x86_64|amd64) arch=amd64 ;; aarch64|arm64) arch=arm64 ;; esac
archive="lazyray_${ver}_${os}_${arch}.tar.gz"
[ -f "dist/${archive}" ] || { echo "FAIL: expected dist/${archive}, found:"; ls dist; exit 1; }

# Ensure a key-A signature over checksums.txt exists (snapshot signing may or may
# not run depending on goreleaser config); mint it if goreleaser did not.
[ -f dist/checksums.txt.minisig ] || \
	minisign -S -s "$keydir/a.key" -m dist/checksums.txt -x dist/checksums.txt.minisig >/dev/null 2>&1
[ -f dist/checksums.txt.minisig ] || { echo "FAIL: could not produce checksums.txt.minisig (key A)"; exit 1; }

fails=0
pass() { printf 'PASS: %s\n' "$1"; }
fail() { printf 'FAIL: %s\n' "$1"; fails=$((fails + 1)); }

# new_base <name>: a private dir holding just the assets install.sh fetches, so
# per-scenario mutations do not leak between scenarios.
new_base() {
	_b="$work/$1"
	rm -rf "$_b"; mkdir -p "$_b"
	cp "dist/${archive}" dist/checksums.txt dist/checksums.txt.minisig "$_b"/
	printf '%s' "$_b"
}

# run_case <script> <base_dir> <prefix> [install-args...] — returns install.sh's
# exit code; output captured to $work/out.log. Called only in `if` conditions.
# Pre-creates the PREFIX (writable, user-owned) so install.sh takes its non-sudo
# install path — the test must NOT depend on sudo, and sudo-owned files would be
# root-owned and un-cleanable by the trap.
run_case() {
	_script=$1; _base=$2; _pfx=$3; shift 3
	mkdir -p "$_pfx"
	env LAZYRAY_VERSION="$ver" LAZYRAY_BASE_URL="file://$_base" PREFIX="$_pfx" \
		sh "$_script" "$@" >"$work/out.log" 2>&1
}

# sign_b <base_dir>: add a key-B signature over checksums.txt as checksums.txt.b.minisig.
sign_b() { minisign -S -s "$keydir/b.key" -m "$1/checksums.txt" -x "$1/checksums.txt.b.minisig" >/dev/null 2>&1; }

# patch_signers <dest> <REQUIRED_SIGNERS-assignment>: throwaway copy of install.sh
# with the trust-list line replaced (the production script is never modified).
# The replacement is passed via the environment (ENVIRON), not `awk -v`, because
# BSD/macOS awk rejects a literal newline in a -v value (multi-signer blocks).
patch_signers() {
	_repl=$2 awk 'BEGIN { r = ENVIRON["_repl"] } /^REQUIRED_SIGNERS=/ { print r; next } { print }' scripts/install.sh > "$1"
}

# A PATH with the tools install.sh needs but WITHOUT minisign (scenario 2b).
nomini="$work/nomini"; mkdir -p "$nomini"
for t in sh env uname mktemp tar chmod curl wget sha256sum shasum sed awk cat tr head mkdir cp rm sudo grep printf; do
	if p=$(command -v "$t" 2>/dev/null); then ln -sf "$p" "$nomini/$t"; fi
done

prod=scripts/install.sh

# Scenario 1 — checksum-only opt-out: --allow-unsigned with no signature present.
b=$(new_base s1); rm -f "$b/checksums.txt.minisig"; pfx="$work/p1"
if run_case "$prod" "$b" "$pfx" --allow-unsigned && [ -x "$pfx/bin/lzr" ]; then
	pass "1 checksum-only install with --allow-unsigned"
else
	fail "1 checksum-only install with --allow-unsigned"; cat "$work/out.log"
fi

# Scenario 2 — fail-closed refusal: required signature missing, no flag.
b=$(new_base s2); rm -f "$b/checksums.txt.minisig"; pfx="$work/p2"
if run_case "$prod" "$b" "$pfx"; then
	fail "2 fail-closed: should have REFUSED but exited 0"
elif [ -x "$pfx/bin/lzr" ]; then
	fail "2 fail-closed: refused but still wrote a binary"
else
	pass "2 fail-closed refusal without --allow-unsigned"
fi

# Scenario 2b — fail-closed when minisign is absent (empty PATH farm).
b=$(new_base s2b); pfx="$work/p2b"; mkdir -p "$pfx"
if env LAZYRAY_VERSION="$ver" LAZYRAY_BASE_URL="file://$b" PREFIX="$pfx" PATH="$nomini" \
		sh "$prod" >"$work/out.log" 2>&1; then
	fail "2b minisign-absent: should have REFUSED but exited 0"
elif [ -x "$pfx/bin/lzr" ]; then
	fail "2b minisign-absent: refused but still wrote a binary"
else
	pass "2b fail-closed when minisign is absent"
fi

# Scenario 3 — happy signature path (single signer = key A).
b=$(new_base s3); cp="$work/install-s3.sh"; pfx="$work/p3"
patch_signers "$cp" "REQUIRED_SIGNERS='$pubkey_a checksums.txt.minisig'"
if run_case "$cp" "$b" "$pfx" && [ -x "$pfx/bin/lzr" ] && grep -q 'signature OK' "$work/out.log"; then
	pass "3 happy signature path (1 signer verified)"
else
	fail "3 happy signature path"; cat "$work/out.log"
fi

# Scenario 4 — bad signature is fatal EVEN under --allow-unsigned.
# Trust-list = key B, but checksums.txt.minisig is signed by key A → mismatch.
b=$(new_base s4); cp="$work/install-s4.sh"; pfx="$work/p4"
patch_signers "$cp" "REQUIRED_SIGNERS='$pubkey_b checksums.txt.minisig'"
if run_case "$cp" "$b" "$pfx" --allow-unsigned; then
	fail "4 bad signature: should have REFUSED but exited 0"
elif [ -x "$pfx/bin/lzr" ]; then
	fail "4 bad signature: refused but still wrote a binary"
else
	pass "4 bad signature is fatal even with --allow-unsigned"
fi

# Scenario 5 — required-all negative: two signers, second signature missing.
b=$(new_base s5); cp="$work/install-s5.sh"; pfx="$work/p5"
patch_signers "$cp" "$(printf "REQUIRED_SIGNERS='%s checksums.txt.minisig\n%s checksums.txt.b.minisig'" "$pubkey_a" "$pubkey_b")"
if run_case "$cp" "$b" "$pfx"; then
	fail "5 required-all: should have REFUSED on a missing 2nd signature but exited 0"
elif [ -x "$pfx/bin/lzr" ]; then
	fail "5 required-all: refused but still wrote a binary"
else
	pass "5 required-all fails closed on a missing signer"
fi

# Scenario 6 — 2-of-2 happy path (both signers valid): guards the -x wiring.
b=$(new_base s6); sign_b "$b"; cp="$work/install-s6.sh"; pfx="$work/p6"
patch_signers "$cp" "$(printf "REQUIRED_SIGNERS='%s checksums.txt.minisig\n%s checksums.txt.b.minisig'" "$pubkey_a" "$pubkey_b")"
if run_case "$cp" "$b" "$pfx" && [ -x "$pfx/bin/lzr" ] && grep -q '2 required signer' "$work/out.log"; then
	pass "6 2-of-2 happy path (both signers verified)"
else
	fail "6 2-of-2 happy path (guards the minisign -x wiring)"; cat "$work/out.log"
fi

# Scenario 7 — --allow-unsigned still refuses a PRESENT-but-invalid signature
# even when another required signature is missing (partial-signature gap).
# checksums.txt.minisig is re-signed with key B (invalid for signer A); the
# second signer's asset (checksums.txt.b.minisig) is left missing.
b=$(new_base s7)
minisign -S -s "$keydir/b.key" -m "$b/checksums.txt" -x "$b/checksums.txt.minisig" >/dev/null 2>&1
cp="$work/install-s7.sh"; pfx="$work/p7"
patch_signers "$cp" "$(printf "REQUIRED_SIGNERS='%s checksums.txt.minisig\n%s checksums.txt.b.minisig'" "$pubkey_a" "$pubkey_b")"
if run_case "$cp" "$b" "$pfx" --allow-unsigned; then
	fail "7 partial-sig: should have REFUSED (present-but-invalid sig is fatal) but exited 0"
elif [ -x "$pfx/bin/lzr" ]; then
	fail "7 partial-sig: refused but still wrote a binary"
else
	pass "7 --allow-unsigned still fatal on a present-but-invalid signature"
fi

# Scenario 8 — an empty REQUIRED_SIGNERS list is a config error: refuse in EVERY
# mode (mirrors verify.go rejecting an empty requiredSigners), never fail open.
b=$(new_base s8); cp="$work/install-s8.sh"; pfx="$work/p8"
patch_signers "$cp" "REQUIRED_SIGNERS=''"
if run_case "$cp" "$b" "$pfx" --allow-unsigned; then
	fail "8 empty-signers: should have REFUSED even with --allow-unsigned but exited 0"
elif [ -x "$pfx/bin/lzr" ]; then
	fail "8 empty-signers: refused but still wrote a binary"
else
	pass "8 empty REQUIRED_SIGNERS refuses in every mode"
fi

echo ""
if [ "$fails" -eq 0 ]; then
	echo "PASS: all install.sh trust scenarios green (ver $ver)"
else
	echo "FAIL: $fails scenario(s) failed"; exit 1
fi
