#!/bin/sh
# install.sh — POSIX installer for lazyray (lzr).
#
# Trust model:
#   * ALWAYS verifies the archive's SHA-256 against the signed checksums.txt.
#   * If `minisign` is on PATH, ALSO verifies checksums.txt.minisig against the
#     embedded trust-list of public keys below (full supply-chain chain of trust).
#   * If `minisign` is absent, prints a loud WARN + install instructions and
#     continues at checksum level — UNLESS --require-signature was passed, in
#     which case the absence is fatal.
#   * Out-of-band (optional): a published release also carries a keyless SLSA
#     build-provenance attestation, an INDEPENDENT trust root disjoint from the
#     minisign key. This installer stays minisign-primary and does NOT require
#     it; verify separately with:
#       gh attestation verify <archive> --repo rtxnik/lazyray
#
# Honest trade-off: a bare `curl … | sh` without minisign gives only
# checksum-level protection (corruption / MITM-beyond-TLS / tampered mirror),
# NOT protection against a compromised release. For the full chain install via
# Homebrew/nfpm, or run this script on a host that has minisign.
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/rtxnik/lazyray/main/scripts/install.sh | sh
#   sh install.sh [--require-signature]
#
# Environment overrides:
#   LAZYRAY_VERSION       install a specific tag (e.g. v0.9.0) instead of latest
#   PREFIX                install root (default /usr/local); binary -> $PREFIX/bin
#   LAZYRAY_BASE_URL      override the asset base URL (testing seam)

set -eu

# --- embedded release-signing trust-list -----------------------------------
# One minisign public key per line. MUST stay byte-in-sync with
# releaseSigningPubKeys in internal/release/verify.go. The installer accepts the
# download when ANY listed key verifies checksums.txt.minisig (accept-any).
RELEASE_PUBKEYS='RWT1X2unwbak2iRSpo1E/k3BWHDjQCzAwgPJft7dtXwRS+3IFxNkR0Ag'

REPO="rtxnik/lazyray"
PREFIX="${PREFIX:-/usr/local}"
REQUIRE_SIGNATURE=0

# --- output helpers --------------------------------------------------------
info() { printf '==> %s\n' "$*"; }
warn() { printf 'WARN: %s\n' "$*" >&2; }
err()  { printf 'ERROR: %s\n' "$*" >&2; }
die()  { err "$@"; exit 1; }

# --- argument parsing ------------------------------------------------------
while [ $# -gt 0 ]; do
	case "$1" in
		--require-signature) REQUIRE_SIGNATURE=1 ;;
		-h|--help)
			sed -n '2,30p' "$0" | sed 's/^# \{0,1\}//'
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

# checksums.txt.minisig is optional at download time: only fatal later if --require-signature.
have_sig=0
if DOWNLOAD "${base}/checksums.txt.minisig" "${workdir}/checksums.txt.minisig" 2>/dev/null; then
	have_sig=1
fi

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

# verify_signature is only invoked from the conditional block below; shellcheck
# can't see the dynamic call, so silence the "never invoked" finding here only.
# shellcheck disable=SC2329
# --- verify_signature: minisign over checksums.txt -------------------------
verify_signature() {
	# Try every embedded trust-list key; accept iff ANY verifies (accept-any).
	verified=0
	old_ifs=$IFS
	IFS='
'
	set -f # iterate raw key lines without globbing
	i=0
	for key in $RELEASE_PUBKEYS; do
		[ -n "$key" ] || continue
		i=$((i + 1))
		pubfile="${workdir}/lazyray_${i}.pub"
		printf 'untrusted comment: lazyray release signing key\n%s\n' "$key" > "$pubfile"
		if minisign -V -p "$pubfile" -m "${workdir}/checksums.txt" >/dev/null 2>&1; then
			verified=1
			break
		fi
	done
	set +f
	IFS=$old_ifs
	[ "$verified" -eq 1 ] \
		|| die "minisign verification of checksums.txt FAILED against every embedded key — refusing to install"
	info "minisign signature OK (trust-list key #${i})"
}

verify_checksum

if command -v minisign >/dev/null 2>&1; then
	if [ "$have_sig" -eq 1 ]; then
		verify_signature
	elif [ "$REQUIRE_SIGNATURE" -eq 1 ]; then
		die "checksums.txt.minisig not available but --require-signature was set"
	else
		warn "checksums.txt.minisig not available; installed with checksum-level verification only"
	fi
else
	if [ "$REQUIRE_SIGNATURE" -eq 1 ]; then
		die "--require-signature was set but minisign is not installed. Install it: https://jedisct1.github.io/minisign/"
	fi
	warn "minisign not found on PATH — installed with CHECKSUM-LEVEL protection only."
	warn "This does NOT protect against a compromised release, only corruption / MITM / tampered mirrors."
	warn "For the full supply-chain chain of trust, install minisign and re-run, or use Homebrew/your distro package:"
	warn "  macOS:  brew install minisign"
	warn "  Debian/Ubuntu:  apt-get install minisign"
	warn "  docs:  https://jedisct1.github.io/minisign/"
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
	sudo sh -c "mkdir -p '$dest_dir' && cp '${workdir}/lzr' '$dest' && chmod 0755 '$dest'"
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
