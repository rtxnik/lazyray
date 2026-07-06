#!/bin/sh
# verify-release.sh — independent, ordinary-user verification of a published
# lazyray release:
#   1. minisign signature over checksums.txt, against the embedded release key
#   2. build-provenance attestation over the archive (gh attestation verify)
#   3. scripts/install.sh end-to-end smoke (real network path, --require-signature)
#
# Usage: verify-release.sh <tag>
set -eu

TAG="${1:?usage: verify-release.sh <tag>}"
REPO="rtxnik/lazyray"
# The workflow whose OIDC identity signs the attestation; --signer-workflow
# rejects an attestation minted by any other workflow, even one in this
# repository. Must be updated if .github/workflows/release.yml is renamed.
SIGNER_WORKFLOW="rtxnik/lazyray/.github/workflows/release.yml"
# Byte-identical to the first entry of RELEASE_PUBKEYS in scripts/install.sh
# and releaseSigningPubKeys in internal/release/verify.go.
RELEASE_PUBKEY="RWT1X2unwbak2iRSpo1E/k3BWHDjQCzAwgPJft7dtXwRS+3IFxNkR0Ag"

ver_noprefix="${TAG#v}"
archive="lazyray_${ver_noprefix}_linux_amd64.tar.gz"
work="$(mktemp -d)"
prefix="$(mktemp -d)"
trap 'rm -rf "$work" "$prefix"' EXIT INT TERM

echo "==> downloading ${TAG} assets as an ordinary user"
gh release download "$TAG" --repo "$REPO" --dir "$work" \
  --pattern "$archive" --pattern checksums.txt --pattern checksums.txt.minisig

echo "==> minisign: checksums.txt against the embedded release key"
pub="${work}/release.pub"
printf 'untrusted comment: lazyray release signing key\n%s\n' "$RELEASE_PUBKEY" > "$pub"
minisign -V -p "$pub" -m "${work}/checksums.txt" -x "${work}/checksums.txt.minisig"

echo "==> attestation: ${archive} (SLSA build provenance)"
gh attestation verify "${work}/${archive}" --repo "$REPO" \
  --signer-workflow "$SIGNER_WORKFLOW"

echo "==> install.sh end-to-end smoke (real release, signature required)"
LAZYRAY_VERSION="$TAG" PREFIX="$prefix" sh scripts/install.sh --require-signature
"${prefix}/bin/lzr" --version

echo "PASS: ${TAG} verified (minisign + attestation + install smoke)"
