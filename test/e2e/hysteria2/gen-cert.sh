#!/usr/bin/env bash
# Generate a self-signed ECDSA cert for the e2e hysteria2 server and emit the
# pinSHA256 the client must use. Certs are ephemeral and git-ignored.
set -euo pipefail
dir="$(cd "$(dirname "$0")" && pwd)/certs"
mkdir -p "$dir"
openssl req -x509 -newkey ec -pkeyopt ec_paramgen_curve:prime256v1 \
  -sha256 -days 2 -nodes \
  -keyout "$dir/server.key" -out "$dir/server.crt" \
  -subj "/CN=hy2.test.local" -addext "subjectAltName=DNS:hy2.test.local"
# pinSHA256 = lowercase hex of the cert DER SHA-256, separators stripped
# (the canonical form lazyray's parser also produces).
openssl x509 -in "$dir/server.crt" -noout -fingerprint -sha256 \
  | sed 's/.*=//; s/://g' | tr 'A-Z' 'a-z' >"$dir/pin.sha256"
echo "pinSHA256=$(cat "$dir/pin.sha256")"
