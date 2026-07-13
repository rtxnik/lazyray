<p align="center">
  <img src="assets/logo.png" alt="lazyray" width="320">
</p>

<h1 align="center">lazyray</h1>

<p align="center">
  <b>Terminal UI for managing Xray-core proxy configurations</b><br>
  <sub>A lazygit-inspired terminal interface for Xray-core. Manage proxy profiles, monitor traffic, and control your network — all from the terminal.</sub>
</p>

<p align="center">
  <a href="https://go.dev/"><img src="https://img.shields.io/github/go-mod/go-version/rtxnik/lazyray?style=flat&color=00ADD8" alt="Go Version"></a>
  <a href="https://github.com/rtxnik/lazyray/releases/latest"><img src="https://img.shields.io/github/v/release/rtxnik/lazyray?style=flat&color=d79921" alt="Release"></a>
  <a href="LICENSE"><img src="https://img.shields.io/github/license/rtxnik/lazyray?style=flat&color=98971a" alt="License"></a>
  <a href="https://github.com/rtxnik/lazyray/actions"><img src="https://img.shields.io/github/actions/workflow/status/rtxnik/lazyray/ci.yml?branch=main&style=flat&label=CI" alt="CI"></a>
  <a href="https://goreportcard.com/report/github.com/rtxnik/lazyray"><img src="https://goreportcard.com/badge/github.com/rtxnik/lazyray" alt="Go Report Card"></a>
  <a href="https://scorecard.dev/viewer/?uri=github.com/rtxnik/lazyray"><img src="https://api.securityscorecards.dev/projects/github.com/rtxnik/lazyray/badge" alt="OpenSSF Scorecard"></a>
</p>

---

## Features

- **Multi-protocol** — VLESS, VMess, Trojan, Shadowsocks, Hysteria2; import via URL or subscription.
- **Full TUI** — three-panel layout (profiles, status, logs), modal editing, health checks, QR export, keyboard-driven.
- **System proxy** — `lzr proxy on/off/status` on macOS, Linux (GNOME/KDE), and Windows; PAC auto-configuration.
- **DNS security** — DNS-over-HTTPS and DNS-over-TLS with per-profile routing rules.
- **Traffic monitoring** — real-time upload/download stats, persistent history, speed testing through the proxy.
- **Subscriptions** — import from subscription URLs with auto-refresh; batch latency testing and auto-sort.
- **Customizable** — themes (Gruvbox, Nord, Catppuccin, Solarized), custom keybindings, responsive layout.
- **Cross-platform** — macOS (launchd), Linux (systemd), Windows (Task Scheduler); service install/uninstall.

## Installation

The binary is named **`lzr`**. Every release ships a **minisign-signed** checksum
manifest (`checksums.txt` + `checksums.txt.minisig`) that lists **every** asset —
archives and `.deb`/`.rpm`/`.apk` packages alike — plus a keyless SLSA
build-provenance attestation. **Verifying one of those is what establishes
trust**, whichever install method you use below.

### Verify what you download (recommended)

Either check is sufficient; each uses a trust root independent of the download.
Verify the signed manifest first, then confirm your asset against it:

```bash
# minisign — one signature authenticates every asset listed in checksums.txt
minisign -Vm checksums.txt -P RWT1X2unwbak2iRSpo1E/k3BWHDjQCzAwgPJft7dtXwRS+3IFxNkR0Ag
sha256sum -c --ignore-missing checksums.txt      # then confirm your asset's line

# or GitHub's SLSA build provenance (a disjoint, keyless root)
gh attestation verify lazyray_<version>_<os>_<arch>.tar.gz --repo rtxnik/lazyray
```

### Homebrew (macOS / Linux)

```bash
brew install rtxnik/tap/lzr
```

Homebrew pins each release's `sha256` in the tap formula (a second channel) and
verifies it on download.

### Linux packages (deb / rpm / apk)

Download the matching package from
[**Releases**](https://github.com/rtxnik/lazyray/releases/latest), **verify it**
against the signed `checksums.txt` (or `gh attestation verify`) as above, then:

```bash
sudo dpkg -i  lazyray_<version>_linux_amd64.deb     # Debian / Ubuntu
sudo rpm -i   lazyray_<version>_linux_amd64.rpm     # Fedora / RHEL / openSUSE
sudo apk add --allow-untrusted lazyray_<version>_linux_amd64.apk   # Alpine
```

> A downloaded package is **not** signature-checked by `dpkg`/`rpm`/`apk` on its
> own — the verification step above is what makes this path trustworthy.

Packages install the `lzr` binary, shell completions, and the man page. They do
not register or start any background service and never require root for config —
the proxy service is user-scoped (`lzr service install`, run as your user).
After installing, fetch xray-core with `lzr update apply`.

### Install script (Linux / macOS)

```bash
curl -fsSL https://raw.githubusercontent.com/rtxnik/lazyray/main/scripts/install.sh | sh
```

The script always verifies the archive's SHA-256 against `checksums.txt` and, **by
default, requires a valid minisign signature** over it from every release signer
(fail-closed). If `minisign` is not installed, or a required signature is missing,
**it refuses to install** and points you at the options above.

To accept checksum-only integrity when you cannot run `minisign` — this protects
against corruption/MITM/tampered mirrors but **not** against a compromised
release:

```bash
curl -fsSL https://raw.githubusercontent.com/rtxnik/lazyray/main/scripts/install.sh \
  | sh -s -- --allow-unsigned
```

> **Trust model:** the strongest path is a **verified** download (minisign or
> `gh attestation verify`), which covers packages too. A bare `curl … | sh` now
> fails closed without `minisign`; `--allow-unsigned` downgrades to checksum-only.
> A signature that is present but invalid is always fatal.

> **Download-then-inspect / pinning an older version:** you can fetch the script,
> read it, and run it locally. If you pin `LAZYRAY_VERSION` to an **older**
> release, fetch `install.sh` from that release's **git tag**, not `main` — the
> `main` script enforces the current signer set and would refuse an older release
> that predates a newly-added signer:
> ```bash
> curl -fsSL https://raw.githubusercontent.com/rtxnik/lazyray/<tag>/scripts/install.sh -o install.sh
> less install.sh && LAZYRAY_VERSION=<tag> sh install.sh
> ```

### go install

```bash
go install github.com/rtxnik/lazyray@latest
```

> **Note:** `go install` produces a binary named **`lazyray`**, not `lzr`
> (Go names the binary after the module path), and it is built without the
> release version stamp or signature verification. All other channels install
> `lzr`. Symlink it if you want the canonical name:
> `ln -s "$(go env GOPATH)/bin/lazyray" "$(go env GOPATH)/bin/lzr"`.

### Build from source

```bash
git clone https://github.com/rtxnik/lazyray.git
cd lazyray
make build    # produces ./lzr
```

## Quick Start

```bash
# Launch TUI — onboarding wizard guides first-time setup
lzr

# Import a proxy profile
lzr import "vless://uuid@host:port?params#name"

# Start the proxy
lzr start

# Check status
lzr status

# Show your exit IP
lzr ip
```

## Concepts

A few terms recur throughout lazyray and its docs:

- **proxy profile** — a saved connection definition (protocol, transport, security) in `servers.yaml`. Most commands act on a profile.
- **proxy server** — the remote endpoint a profile connects to.
- **system proxy** — your OS-level proxy settings, toggled with `lzr proxy on`/`off` (separate from a proxy profile).
- **xray-core** — the proxy engine `lzr` drives; the generated **xray config** is the JSON `lzr` builds from your active profile.
- **profile store** — the YAML config (`servers.yaml`, `lazyray.yaml`) where profiles and settings live.
- **SSH tunnel** — `lzr tunnel` opens an SSH tunnel to a server's admin panel; it does **not** route your traffic and is unrelated to the proxy itself.
- **diagnostics vs health** — `lzr doctor` is a full diagnostic sweep; `lzr health` is a quick connectivity probe of the active profile.

## CLI Commands

The everyday core:

| Command | Description |
|---------|-------------|
| `lzr` | Launch interactive TUI |
| `lzr start` / `lzr stop` / `lzr restart` | Control the xray proxy |
| `lzr status` / `lzr health` / `lzr ip` | Status, health check, exit IP (`--json` supported) |
| `lzr import <url>` | Import a proxy URL (`--sub <url>` for subscriptions) |
| `lzr export [name] [--qr]` | Export a profile as URL or QR code |
| `lzr config <list\|switch\|show\|edit\|...>` | Manage profiles and configs |
| `lzr test [name] [--all]` | Test connection / batch latency |
| `lzr proxy <on\|off\|status>` | System proxy management |
| `lzr update <check\|apply>` | Manage the xray-core engine |
| `lzr service <install\|uninstall>` | Manage the autostart service |
| `lzr self-update` | Update lazyray itself |
| `lzr doctor` | Full diagnostic sweep |

More (`speedtest`, `stats`, `logs`, `pac`, `tunnel`, every flag) — in the
generated [command reference](docs/reference/cli/lzr.md), or `lzr --help`.

### Hysteria2 links

`lzr import` accepts `hysteria2://` / `hy2://` links with these parameters:

- `obfs=salamander` + `obfs-password` — salamander obfuscation (the only type xray-core supports).
- `sni`, `pinSHA256` — TLS. Prefer `pinSHA256` (hex cert fingerprint): xray-core >= v26 removed `insecure` / `allowInsecure`.
- Inline port-hopping in the `host:port` slot, e.g. `host:443,5000-6000`.
- `alpn`, `fp` are accepted as non-standard extensions.

Hysteria2 requires a hysteria2-capable xray-core (>= 26.2.6); `lzr` blocks startup on older builds. The pinned default fetched by `lzr update apply` (`v26.3.27`) already satisfies this. See `test/e2e/hysteria2/README.md` for the e2e harness.

## Configuration

Configuration files are stored in `~/.config/lazyray/` (macOS/Linux) or `%APPDATA%\lazyray\` (Windows):

| File | Purpose |
|------|---------|
| `servers.yaml` | Server profiles — protocols, transport, security, routing |
| `lazyray.yaml` | Application settings — ports, health checks, UI, subscriptions |
| `keys.yaml` | Custom keybinding overrides (optional) |

Data files (xray binary, logs, backups) are stored in `~/.local/share/lazyray/` (macOS/Linux) or `%LOCALAPPDATA%\lazyray\` (Windows).

See the [configuration reference](docs/reference/configuration.md) for every setting.

## Troubleshooting

Hit a snag? See [TROUBLESHOOTING.md](TROUBLESHOOTING.md) for the common cases
(removed `allowInsecure`, Hysteria2 version gate, missing geo data, macOS
Gatekeeper, Linux system proxy, headless use). `lzr doctor` diagnoses most
problems in one pass.

## Documentation

- [Command reference](docs/reference/cli/lzr.md) — generated
- [Keybindings reference](docs/reference/keybindings.md) — generated
- [Configuration reference](docs/reference/configuration.md)
- [Exit codes](docs/reference/exit-codes.md)
- [Architecture](docs/ARCHITECTURE.md) — code map, invariants, dependency graph
- [Contributing](CONTRIBUTING.md) — dev setup, tests, the project's invariants
- [Security policy](SECURITY.md) — reporting, trust model, release verification

## Requirements

- [Xray-core](https://github.com/XTLS/Xray-core) — downloaded automatically via `lzr update apply`, pinned to a known-good version (default `v26.3.27`); override with `lzr update apply --version <tag>`. The download is verified against XTLS's published `.dgst` SHA-256 checksum before it is executed.
- Go 1.26+ (building from source only)

## License

[MIT](LICENSE)
