#!/bin/sh
# install.sh — POSIX installer for lazyray (lzr).
#
# Trust model (fail-closed by default):
#   * ALWAYS verifies the archive's SHA-256 against checksums.txt.
#   * By DEFAULT also REQUIRES a valid minisign signature over checksums.txt
#     from EVERY required signer (see REQUIRED_SIGNERS) — the required-all model
#     that mirrors internal/release/verify.go. If minisign is not installed, or a
#     required signature is missing, the install REFUSES.
#   * --allow-unsigned downgrades to checksum-only (explicit opt-out) with a loud
#     warning. A signature that is PRESENT but INVALID is ALWAYS fatal.
#   * Out-of-band (recommended): every release also carries a keyless SLSA
#     build-provenance attestation — a trust root disjoint from the minisign key.
#     Cross-check with:
#       gh attestation verify <asset> --repo rtxnik/lazyray
#
# The strongest trust comes from VERIFICATION (the minisign-signed checksums.txt
# enumerates every asset — packages included — and `gh attestation verify`), not
# from the install method. Prefer `brew install rtxnik/tap/lzr`, or verify a
# downloaded asset against the signed checksums before using it.
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/rtxnik/lazyray/main/scripts/install.sh | sh
#   sh install.sh [--allow-unsigned]
#
# Pinning an OLDER release? Set LAZYRAY_VERSION and fetch THIS script from the
# matching git tag (…/<tag>/scripts/install.sh), not main — the main script
# enforces the current signer set and would refuse a release predating a signer.
#
# Environment overrides:
#   LAZYRAY_VERSION       install a specific tag (e.g. v0.9.0) instead of latest
#   PREFIX                install root (default /usr/local); binary -> $PREFIX/bin
#   LAZYRAY_BASE_URL      override the asset base URL (testing seam)

set -eu

# --- required release signers ----------------------------------------------
# EVERY signer listed here MUST produce a valid minisign signature over
# checksums.txt for the download to be accepted (required-all — mirrors
# requiredSigners in internal/release/verify.go; defeats single-key compromise).
# Retire a key by REMOVING its line; activate 2-of-2 by APPENDING a second
# signer line. Per-line format: "<minisign-public-key> <sig-asset-filename>".
REQUIRED_SIGNERS='RWT1X2unwbak2iRSpo1E/k3BWHDjQCzAwgPJft7dtXwRS+3IFxNkR0Ag checksums.txt.minisig'

REPO="rtxnik/lazyray"
PREFIX="${PREFIX:-/usr/local}"
ALLOW_UNSIGNED=0

# --- output helpers --------------------------------------------------------
info() { printf '==> %s\n' "$*"; }
warn() { printf 'WARN: %s\n' "$*" >&2; }
err()  { printf 'ERROR: %s\n' "$*" >&2; }
die()  { err "$@"; exit 1; }

# --- argument parsing ------------------------------------------------------
while [ $# -gt 0 ]; do
	case "$1" in
		--allow-unsigned) ALLOW_UNSIGNED=1 ;;
		--require-signature)
			# Signature verification is the default now; kept as an accepted
			# no-op alias so previously-scripted one-liners keep working.
			info "--require-signature is the default now; flag accepted as a no-op" ;;
		-h|--help)
			sed -n '2,/^$/p' "$0" | sed 's/^# \{0,1\}//'
			exit 0
			;;
		*) die "unknown argument: $1 (try --help)" ;;
	esac
	shift
done

# --- dependency check ------------------------------------------------------
need() { command -v "$1" >/dev/null 2>&1 || die "required tool not found: $1"; }
need uname
need mktemp
need tar
need chmod

# A downloader: prefer curl, fall back to wget.
if command -v curl >/dev/null 2>&1; then
	DOWNLOAD() { curl -fsSL "$1" -o "$2"; }
elif command -v wget >/dev/null 2>&1; then
	DOWNLOAD() { wget -q "$1" -O "$2"; }
else
	die "need curl or wget to download release assets"
fi

# A SHA-256 tool: sha256sum (Linux) or shasum -a 256 (macOS).
if command -v sha256sum >/dev/null 2>&1; then
	SHA256() { sha256sum "$1" | awk '{print $1}'; }
elif command -v shasum >/dev/null 2>&1; then
	SHA256() { shasum -a 256 "$1" | awk '{print $1}'; }
else
	die "need sha256sum or 'shasum -a 256' to verify the download"
fi

# --- platform detection ----------------------------------------------------
os=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$os" in
	linux)  os=linux ;;
	darwin) os=darwin ;;
	*) die "unsupported OS: $(uname -s) (this installer supports Linux and macOS; on Windows use the .zip from GitHub Releases)" ;;
esac

arch=$(uname -m)
case "$arch" in
	x86_64|amd64)  arch=amd64 ;;
	aarch64|arm64) arch=arm64 ;;
	*) die "unsupported architecture: $arch" ;;
esac

# --- resolve version -------------------------------------------------------
version="${LAZYRAY_VERSION:-}"
if [ -z "$version" ]; then
	info "resolving latest release tag"
	tmp_api=$(mktemp)
	DOWNLOAD "https://api.github.com/repos/${REPO}/releases/latest" "$tmp_api" \
		|| die "could not reach the GitHub API to resolve the latest tag (set LAZYRAY_VERSION to install offline)"
	version=$(sed -n 's/.*"tag_name": *"\([^"]*\)".*/\1/p' "$tmp_api" | head -n1)
	rm -f "$tmp_api"
	[ -n "$version" ] || die "could not parse latest tag_name from the GitHub API response"
fi

# Strip a leading 'v' for the archive name; archives are lazyray_<ver>_<os>_<arch>.tar.gz
ver_noprefix=${version#v}
archive="lazyray_${ver_noprefix}_${os}_${arch}.tar.gz"

# Base URL: testing seam (file:// dist) or the real GitHub release download path.
base="${LAZYRAY_BASE_URL:-https://github.com/${REPO}/releases/download/${version}}"

# --- download into a scratch dir -------------------------------------------
workdir=$(mktemp -d)
trap 'rm -rf "$workdir"' EXIT INT TERM

info "downloading ${archive} (${version})"
DOWNLOAD "${base}/${archive}"              "${workdir}/${archive}"     || die "failed to download ${archive}"
DOWNLOAD "${base}/checksums.txt"           "${workdir}/checksums.txt"  || die "failed to download checksums.txt"

# --- verify_checksum: always run -------------------------------------------
# Match the archive's line in checksums.txt, then compare the SHA-256.
verify_checksum() {
	expected=$(awk -v f="$archive" '$2 == f || $2 == "*"f {print $1}' "${workdir}/checksums.txt" | head -n1)
	[ -n "$expected" ] || die "no checksum entry for ${archive} in checksums.txt"
	actual=$(SHA256 "${workdir}/${archive}")
	if [ "$expected" != "$actual" ]; then
		die "checksum mismatch for ${archive}: expected ${expected}, got ${actual}"
	fi
	info "checksum OK (sha256 ${actual})"
}

# --- download_required_sigs: best-effort fetch of every signer's .minisig -----
# Downloads whatever sig-assets are available; missing ones are simply absent
# from the workdir afterwards. Never fatal on its own — presence and validity are
# judged by verify_present_signatures and the decision block below.
download_required_sigs() {
	_oifs=$IFS; IFS='
'
	set -f
	for _rec in $REQUIRED_SIGNERS; do
		[ -n "$_rec" ] || continue
		_asset=${_rec#* }
		[ "$_asset" != "$_rec" ] || continue   # malformed record (no space) — skip
		DOWNLOAD "${base}/${_asset}" "${workdir}/${_asset}" 2>/dev/null || :
	done
	set +f
	IFS=$_oifs
}

# --- verify_present_signatures: verify every PRESENT required signature -------
# A present-but-invalid signature is ALWAYS fatal (every mode). Missing sigs are
# tolerated here; the decision block decides whether a missing one is acceptable.
# Sets globals: sigs_total (required signers) and sigs_ok (present AND verified).
verify_present_signatures() {
	sigs_total=0; sigs_ok=0
	_oifs=$IFS; IFS='
'
	set -f
	for _rec in $REQUIRED_SIGNERS; do
		[ -n "$_rec" ] || continue
		sigs_total=$((sigs_total + 1))
		_pubkey=${_rec%% *}
		_asset=${_rec#* }
		[ -f "${workdir}/${_asset}" ] || continue   # sig not present — tolerated here
		_pubfile="${workdir}/lazyray_signer_${sigs_total}.pub"
		printf 'untrusted comment: lazyray release signing key\n%s\n' "$_pubkey" > "$_pubfile"
		# -x is load-bearing: point minisign at THIS signer's sig-asset. Without
		# it minisign defaults to checksums.txt.minisig for every signer, so a
		# second signer would be checked against the first signer's signature.
		if ! minisign -V -p "$_pubfile" -x "${workdir}/${_asset}" -m "${workdir}/checksums.txt" >/dev/null 2>&1; then
			set +f; IFS=$_oifs
			die "minisign verification FAILED for required signer #${sigs_total} (${_asset}) — refusing to install"
		fi
		sigs_ok=$((sigs_ok + 1))
	done
	set +f
	IFS=$_oifs
}

# --- guidance shown when the fail-closed default cannot complete ------------
fail_closed_guidance() {
	err "signature verification is required by default but could not be completed."
	err "Choose a verifiable path:"
	err "  1. brew install rtxnik/tap/lzr  (or verify checksums.txt.minisig / 'gh attestation verify', then install any asset)"
	err "  2. install minisign and re-run:  https://jedisct1.github.io/minisign/  (macOS: brew install minisign; Debian/Ubuntu: apt-get install minisign)"
	err "  3. accept checksum-only integrity (NOT protection against a compromised release):  re-run with --allow-unsigned"
	err "  4. cross-check out-of-band:  gh attestation verify ${archive} --repo ${REPO}"
}

# --- signature decision ----------------------------------------------------
verify_checksum

# A configured-empty signer list is a packaging error, never a valid state
# (mirrors internal/release/verify.go rejecting an empty requiredSigners). Guard
# it unconditionally so a botched key-retirement edit fails closed instead of
# installing an unverified payload.
[ "$(printf '%s\n' "$REQUIRED_SIGNERS" | grep -c '[^[:space:]]' || :)" -gt 0 ] \
	|| die "no required signers configured in the installer — refusing to install"

sigs_total=0; sigs_ok=0
if command -v minisign >/dev/null 2>&1; then
	download_required_sigs
	verify_present_signatures    # dies on ANY present-but-invalid signature
fi

if [ "$sigs_total" -gt 0 ] && [ "$sigs_ok" -eq "$sigs_total" ]; then
	# Every required signer's signature was present and verified — fully trusted,
	# regardless of --allow-unsigned.
	info "minisign signature OK (${sigs_ok} required signer(s) verified)"
elif [ "$ALLOW_UNSIGNED" -eq 1 ]; then
	# Opt-out: at least one required signature is ABSENT (a present-but-invalid
	# one would already have been fatal above). Proceed at checksum level.
	warn "installing with CHECKSUM-ONLY verification (--allow-unsigned)."
	warn "this protects against corruption / MITM / tampered mirrors, NOT against a compromised release."
	warn "for the full chain, prefer 'brew install rtxnik/tap/lzr' or 'gh attestation verify ${archive} --repo ${REPO}'."
else
	command -v minisign >/dev/null 2>&1 || err "minisign is not installed, so the required signature could not be verified."
	fail_closed_guidance
	exit 1
fi

# --- extract lzr -----------------------------------------------------------
info "extracting lzr from ${archive}"
tar -xzf "${workdir}/${archive}" -C "$workdir" lzr \
	|| die "could not extract 'lzr' from ${archive}"
[ -f "${workdir}/lzr" ] || die "extracted archive did not contain an 'lzr' binary"
chmod 0755 "${workdir}/lzr"

# --- install into $PREFIX/bin (sudo only when needed) ----------------------
dest_dir="${PREFIX}/bin"
dest="${dest_dir}/lzr"

run_install() {
	mkdir -p "$dest_dir"
	# Copy then chmod so install survives cross-device moves.
	cp "${workdir}/lzr" "$dest"
	chmod 0755 "$dest"
}

if [ -w "$PREFIX" ] || { [ -d "$dest_dir" ] && [ -w "$dest_dir" ]; }; then
	info "installing to ${dest}"
	run_install
elif command -v sudo >/dev/null 2>&1; then
	info "installing to ${dest} (requires sudo)"
	# Pass paths as positional args, NOT interpolated into the -c string, so a
	# PREFIX containing shell metacharacters cannot inject a root command.
	sudo sh -c 'mkdir -p "$1" && cp "$2" "$3" && chmod 0755 "$3"' _ "$dest_dir" "${workdir}/lzr" "$dest"
else
	die "cannot write to ${dest_dir} and sudo is not available; set PREFIX to a writable location (e.g. PREFIX=\$HOME/.local)"
fi

# --- next steps ------------------------------------------------------------
info "installed lzr ${version} to ${dest}"
cat <<EOF

lazyray (lzr) is installed. Next steps:

  1. Make sure ${dest_dir} is on your PATH.
  2. Download xray-core:        lzr update apply
  3. (optional) run in background:  lzr service install

Run 'lzr' to launch the TUI, or 'lzr --help' for all commands.
EOF
