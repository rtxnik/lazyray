# Troubleshooting

Common problems and fixes. When in doubt, run `lzr doctor` first — it diagnoses
install, config, session, routing, and connectivity in one pass and points at
the right section below.

## "xray rejected the removed `allowInsecure` option"

**Symptom:** a profile that used to work now fails to start after an xray-core update.

**Cause:** xray-core **v26 removed** `allowInsecure` / `insecure`. Profiles that
relied on skipping TLS verification no longer start.

**Fix:** re-import the profile using certificate pinning instead — add a
`pinSHA256` (the hex SHA-256 of the server certificate) to the link and import
it again. See the Hysteria2 section of the README for the link format.

## Hysteria2 profile will not start

**Symptom:** `lzr start` aborts for a `hysteria2://` profile with a version error.

**Cause:** Hysteria2 requires a hysteria2-capable **xray-core ≥ v26.2.6**; `lzr`
blocks startup on older builds.

**Fix:** fetch the pinned xray-core with `lzr update apply` (the default pin,
`v26.3.27`, satisfies the requirement). Verify with `lzr doctor`, which reports
the detected xray-core version.

## Reality handshake fails when the clock is off

**Symptom:** a Reality (VLESS) profile fails to connect even though the link is
correct and the server is reachable.

**Cause:** Reality folds the system time into its handshake, so a local clock
that has drifted more than about a minute from real time breaks the connection.

**Fix:** sync the clock against a network time source (enable automatic time on
the OS, or run `ntpdate`/`chronyc`/`w32tm /resync`), then retry.

## "geoip.dat not found" / routing rules do nothing

**Symptom:** xray-core logs a missing `geoip.dat` or `geosite.dat`, or geo-based
routing rules are ignored.

**Cause:** the geo data files ship with xray-core and are extracted by the
updater; they are missing if xray-core was never fetched through `lzr`.

**Fix:** run `lzr update apply` to download and extract xray-core together with
its geo data. `lzr doctor` flags missing geo files under its install checks.

## macOS: "lzr cannot be opened because the developer cannot be verified"

**Symptom:** macOS Gatekeeper blocks the binary on first run.

**Cause:** binaries downloaded outside the App Store carry a quarantine flag.

**Fix:** install via Homebrew (`brew install rtxnik/tap/lzr`), which is not
quarantined; or clear the flag on a manually-downloaded binary with
`xattr -d com.apple.quarantine ./lzr`; or allow it once under
System Settings → Privacy & Security.

## Linux: system proxy does not change

**Symptom:** `lzr proxy on` reports success but applications still go direct.

**Cause:** desktops differ — GNOME uses `gsettings`, KDE uses `kwriteconfig`, and
headless or other environments expose no system-proxy backend. Some apps read
proxy settings only at launch.

**Fix:** check `lzr proxy status` to see what was applied, then restart the
application. On unsupported desktops, configure the app to use the local
SOCKS5/HTTP listener directly (see `lzr status` for the listen addresses).

## "no profiles configured" or "profile not found"

**Symptom:** a command exits with `no profiles configured` or `profile "x" not found`.

**Cause:** the profile store is empty, or the name does not match.

**Fix:** the error's `→ try:` line names the next step — import a profile with
`lzr import <url>`, or list the exact names with `lzr config list`.

## Running on a server with no terminal UI

**Symptom:** `lzr` (no subcommand) fails to render or you are over SSH with no TTY.

**Cause:** the bare `lzr` command launches the interactive TUI, which needs a
terminal.

**Fix:** use the headless subcommands instead — `lzr start`, `lzr status`,
`lzr stop`, `lzr import`, `lzr config …`. `lzr doctor` runs headless and reports
the degraded mode.

## `lzr service install` fails

**Symptom:** installing the autostart service errors out.

**Cause:** the service is **user-scoped** (launchd LaunchAgent / `systemctl --user`
/ a per-user Scheduled Task) and never runs as root; a missing user session bus
or an unwritable path stops it.

**Fix:** run `lzr service install` as your normal user (not `sudo`). Run
`lzr doctor`, whose startup checks explain a failed service registration.

## An update was rejected for a bad signature or checksum

**Symptom:** `lzr update apply` or `lzr self-update` refuses to install.

**Cause:** the downloaded artifact failed checksum or minisign verification —
the integrity boundary that protects you from a tampered or corrupted download.

**Fix:** this is working as intended — do **not** bypass it. Retry (a corrupted
download self-corrects); if it persists, fetch the release manually from the
GitHub releases page and verify `checksums.txt.minisig` yourself.
