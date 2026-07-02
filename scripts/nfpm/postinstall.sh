#!/bin/sh
# nfpm postinstall scriptlet for lazyray (deb/rpm/apk).
# Informational only: this package installs the lzr binary, shell completions,
# and the man page. It does NOT download xray-core and does NOT register or
# start any background service (the lazyray service is user-scoped and must be
# installed per-user, never as root).
set -e

echo "lazyray (lzr) installed."
echo "Next steps (run as your normal user, NOT root):"
echo "  lzr update apply     # download xray-core + geoip/geosite"
echo "  lzr service install  # install the user-scoped background service"

exit 0
