#!/usr/bin/env bash
# One-time smoke against the operator's REAL hysteria2 server. The link is read
# from $LZR_SMOKE_LINK or stdin and is NEVER written to the repo.
#
#   make smoke-real            # prompts for the link (hidden)
#   LZR_SMOKE_LINK=... make smoke-real
#
# Requires: an xray binary (XRAY_BIN, >= v26.2.6, with geoip.dat/geosite.dat in
# its dir), curl, python3, and a Go toolchain (or LZR_BIN).
set -euo pipefail

link="${LZR_SMOKE_LINK:-}"
if [[ -z "$link" ]]; then
	read -r -s -p "Paste hysteria2:// link (hidden, not stored): " link
	echo
fi
case "$link" in
hysteria2://* | hy2://*) ;;
*)
	echo "not a hysteria2 link"
	exit 2
	;;
esac

xray_src="${XRAY_BIN:-$(command -v xray || true)}"
if [[ -z "$xray_src" || ! -x "$xray_src" ]]; then
	echo "set XRAY_BIN to a hysteria2-capable xray (>= v26.2.6) whose dir has geoip.dat/geosite.dat"
	exit 2
fi

repo="$(cd "$(dirname "$0")/../../.." && pwd)"
home="$(mktemp -d)"
data="$home/.local/share/lazyray"
mkdir -p "$data"
cp "$xray_src" "$data/xray"
for d in geoip.dat geosite.dat; do
	if [[ -f "$(dirname "$xray_src")/$d" ]]; then cp "$(dirname "$xray_src")/$d" "$data/$d"; fi
done

export HOME="$home" XDG_CONFIG_HOME="$home/.config" XDG_DATA_HOME="$home/.local/share"
lzr="${LZR_BIN:-}"
if [[ -z "$lzr" ]]; then
	lzr="$home/lzr"
	(cd "$repo" && go build -o "$lzr" .)
fi

cleanup() {
	"$lzr" stop >/dev/null 2>&1 || true
	rm -rf "$home"
}
trap cleanup EXIT

"$lzr" import "$link" --force >/dev/null
"$lzr" start --no-proxy
sleep 3
port="$(python3 -c "import json;print(next(i['port'] for i in json.load(open('$data/config.json'))['inbounds'] if i['protocol']=='socks'))")"
echo "egress IP through your hysteria2 server:"
if curl -fsS --max-time 10 --socks5-hostname "127.0.0.1:$port" https://api.ipify.org; then
	echo
	echo "SMOKE OK — traffic egressed through your hysteria2 server"
else
	echo "SMOKE FAILED — check 'lzr status' and the xray error log"
	exit 1
fi
