# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

> **Releasing:** entries accumulate under `[Unreleased]`; a release is cut by
> tagging `vX.Y.Z` on a green `main` after a preflight gate, which drives
> GoReleaser to build, sign (minisign), and publish the artifacts with release
> notes generated from merged pull requests and their labels.

## [Unreleased]

### Added
- Reproducible release builds (`-trimpath` plus commit-timestamped module stamps, byte-identical binaries per commit) and a Syft-generated SPDX SBOM (`<archive>.sbom.json`) published alongside every release archive.
- SLSA build-provenance attestation for release artifacts, verifiable with `gh attestation verify <file> --repo rtxnik/lazyray`; releases are now published draft-first, so all assets and the attestation exist before a release becomes public.

### Changed
- `scripts/install.sh` now verifies the release signature by **default** and
  refuses to install without one (`--allow-unsigned` opts down to checksum-only);
  the embedded trust-list is a **required-signer** set (every listed key must
  verify, mirroring the runtime update path) rather than accept-any.
- Attestation verification (docs and the post-release verify job) now pins the
  signer workflow (`gh attestation verify --signer-workflow`), rejecting a
  provenance statement minted by any workflow other than the release pipeline.
- Release-candidate tags (`v*-rc.N`) now publish as GitHub prereleases, so
  `releases/latest` (used by `scripts/install.sh` and `lzr self-update`) never
  resolves to a rehearsal build.

### Fixed
- The post-release verification workflow now triggers off the completed
  `Release` run (plus a manual dispatch fallback): releases published with the
  workflow's own token emit no `release: published` event for other workflows,
  so the previous trigger never fired for pipeline-published releases.
- Shadowsocks SIP002 links with a trailing slash or query (`…:8388/`,
  `…/?plugin=…`) no longer fail to parse, and a `plugin=` requirement is
  rejected with an explicit message instead of a confusing port error.
- Shadowsocks plaintext userinfo is percent-decoded per SIP002, so passwords
  with encoded special characters (e.g. `p%40ss`) are stored correctly.
- VLESS, Trojan, and Shadowsocks share links bracket IPv6 hosts on export;
  previously the exported URL was malformed and could not be re-imported.
- Subscription bodies encoded with unpadded URL-safe base64 now decode
  instead of failing with "no valid proxy URLs found in subscription".
- Hysteria2 share links keep the base port when a port-hopping range is set
  (exported as `443,5000-6000`), so export→import no longer changes the
  connection port.
- The background supervisor now tracks the current xray process across
  auto-restarts, so teardown and crash self-heal always target the running
  process instead of a stale pre-restart PID.
- Crash-loop handling is paced and self-limiting: a healthy run restores the
  full restart budget, rapid repeated crashes are backed off and eventually
  abandoned, and a shutdown never spawns a doomed process.
- Starting `lzr start` while a session is already running no longer records a
  spurious startup failure (so `lzr status`/`lzr doctor` stay accurate).

### Security
- `lzr update apply` now verifies the downloaded xray-core archive against a
  checksum pinned inside the signed lazyray release, instead of a checksum
  fetched from the same origin as the archive — so a tampered or substituted
  engine is rejected before it is ever extracted or run. Only versions pinned by
  this release install by default; `--allow-unverified-xray` opts back into the
  old same-origin checksum path (a corruption check, explicitly not a security
  guarantee) with a prominent warning.
- `lzr self-update` and `lzr update apply` refuse to install a version older
  than the one already installed unless `--allow-downgrade` is passed, and
  `lzr update apply` enforces a hard minimum xray-core version floor — closing
  signed-downgrade attacks that reintroduce a previously patched vulnerability.
  The version floor is enforced by the updater itself, so every entry point
  (CLI and TUI) obeys it.
- A release is now accepted only when every configured signing key verifies it
  (previously any one trusted key sufficed), so a single compromised signing key
  can no longer mint an accepted release once a second key is in use.
- Engine updates now stage the whole file set and roll the entire set back with a
  re-verified backup if the update fails, macOS quarantine is cleared only on the
  verified binary (never the whole directory), and self-update writes are flushed
  to disk before the atomic swap.
- `lzr config backup` now encrypts archives by default — backups bundle
  proxy credentials, and plaintext archives were previously created
  world-readable. The passphrase comes from `--passphrase-file`, the
  `LAZYRAY_PASSPHRASE` environment variable, or an interactive prompt;
  `--no-encrypt` restores the old plaintext behavior (scripted,
  non-interactive backups must now pass one of these). Archives are written
  `0600` via atomic rename either way.
- The default backup filename now ends in `.tar.gz.enc` when encrypted;
  scripts that glob `lazyray-backup-*.tar.gz` should be updated (rotation is
  unaffected). `--no-encrypt` and `--passphrase-file` are now mutually
  exclusive flags.
- `lzr config restore` no longer follows a symlink planted at a destination
  path: restored files replace the destination atomically.
- Encrypted profile exports upgraded from PBKDF2 (100k iterations) to
  Argon2id with parameters stored in the container (`LZRENC2`); `LZRENC1`
  exports from older versions still import. Older lazyray versions cannot
  read `LZRENC2` data.
- `stats.json` is written `0600`, and freshly created config/data/log/backup
  directories are `0700`.
- Generated PAC files now JSON-encode every routing value, so a routing entry
  in an imported profile can no longer break out of the PAC JavaScript and
  inject script into the file browsers execute.
- Untrusted profile fields (name, server address, transport, and SSH host) are
  stripped of terminal control/escape bytes both on import and before any
  terminal output, preventing ANSI/OSC terminal-escape injection from a crafted
  share link, subscription, or encrypted export.
- Encrypted profile imports no longer silently apply the routing and DNS
  overrides they carry: those are dropped with a warning unless the new
  `lzr import --allow-routing` flag is given, and each DNS server is then
  checked against an allowlist (plain IP or a DoH/DoT URL).
- Profile validation is now enforced before a profile is saved on every import
  path and covers chained hops, so a share link or export with an out-of-range
  port can no longer persist a broken profile.
- Release signatures are now verified against an embedded trust-list of signing
  keys rather than a single key, and the in-binary self-updater is rotation-ready
  with no change to the signed-artifact filenames or the download path.
- SSH tunnels now verify the server host key: first connect requires explicit
  fingerprint confirmation (TUI prompt, CLI prompt, or `lzr tunnel trust`),
  pinned keys are stored in the profile and enforced with
  `StrictHostKeyChecking=yes` against a per-profile known_hosts, and a changed
  host key refuses to connect until explicitly re-trusted. The ssh destination
  is also passed after `--` and SSH user/host values starting with `-` are
  rejected, closing an argument-injection vector.
- The `curl | sh` install path is now **fail-closed on signature**: a bare
  one-liner without `minisign` (or with a missing/invalid required signature)
  refuses to install instead of silently degrading to checksum-only. Installation
  docs now lead with independent verification (`minisign` / `gh attestation
  verify`, which cover the Linux packages too) and no longer imply that a manual
  package download is signature-checked on its own.

## [1.0.0] - 2026-07-02

### Added
- `TROUBLESHOOTING.md` covering the common failure cases, cross-linked from the README and from `lzr doctor`.
- A "Concepts" glossary section in the README.
- Per-command examples in `lzr <cmd> --help` and the generated man pages, and an `ssh-tunnel` alias for `lzr tunnel`.
- Hysteria2 certificate pinning (`pinSHA256`) and port-hopping support.
- End-to-end hysteria2 test harness (apernet/hysteria interop, xray-inbound smoke, port-hopping) and structural `xray -test` config validation in CI.
- Minisign-signed releases: `checksums.txt` is signed to `checksums.txt.minisig`; the public key is embedded in the binary and in `scripts/install.sh`.
- `internal/release` package: pure-Go minisign + checksum verification (`aead.dev/minisign`) consumed before any downloaded artifact is executed.
- Homebrew formula (`brew install rtxnik/tap/lzr`), native Linux packages (deb/rpm/apk), and a POSIX `scripts/install.sh` installer.
- Shell completions (bash/zsh/fish) and a `lzr.1` man page, shipped in every archive and package.
- `xrayVersion` setting (default `v26.3.27`) pinning the xray-core download; `lzr update apply --version <tag>` overrides it.
- Contributor and security documentation: `docs/ARCHITECTURE.md`, `CONTRIBUTING.md`,
  `SECURITY.md`, a code of conduct, and issue/PR templates. The internal `DEVNOTES.md`
  scratch file was removed; its content now lives in the reference and architecture docs.

### Changed
- Command errors now print a single, consistent format with an actionable next step (`Error: …` / `  → try: …`), most pointing at `lzr doctor`.
- Every command now ships a complete `--help`: a `Long` description, usage examples, and documented flags.
- Hysteria2 now requires xray-core >= 26.2.6 (checked at start) and rejects unsupported `obfs` values; certificate pinning supersedes `allowInsecure` when both are present.
- `lzr update apply` now fetches a pinned xray-core tag (was `releases/latest`) and verifies the XTLS-published `.dgst` SHA-256 before extracting or executing it.
- TLS-branch profile import now validates a 64-hex `pinSHA256` at import time (matching the existing hysteria2-branch check).

### Fixed
- Clear guidance when xray rejects the removed `allowInsecure` option (re-import with `pinSHA256`).
- Documented Go toolchain is 1.24 (CI/release), correcting the prior 1.22 note.
- `lzr self-update` now works against published releases: it resolves the versioned archive name, verifies the signed checksum manifest, and extracts the `lzr` binary from the archive before atomically swapping it (previously it built a versionless asset name that matched nothing and wrote the raw tarball over the running binary).
- `lzr config duplicate` now deep-copies the source profile. Editing the new
  copy's server chain, tags, or routing lists no longer mutates the original
  profile.

### Security
- Windows notifications and scheduled-task installation now pass dynamic values
  (notification text and the executable path) to PowerShell as quoted literals
  instead of interpolating them into the command line, removing a command-injection
  vector.
- Exit-IP probe responses are read through a bounded reader (64 KiB), so a
  misbehaving or hostile IP-echo endpoint cannot force an unbounded read.

## [0.8.0] - 2026-02-27

### Added
- Shadowsocks protocol support (SIP002 URL parsing, xray config generation, export)
- New transport types: HTTP/2 (h2), HTTPUpgrade, SplitHTTP with full URL parsing and config generation
- System proxy management: `lzr proxy on`, `lzr proxy off`, `lzr proxy status` commands
- Platform-specific system proxy backends (macOS networksetup, Linux GNOME gsettings / KDE kwriteconfig5, Windows registry)
- PAC auto-configure: `lzr pac serve --system` enables system PAC proxy with graceful rollback on exit
- DNS routing rules with DoH/DoT support (`https://`, `tcp://` DNS servers)
- Per-profile conditional DNS rules in routing editor (`W` key)
- Comprehensive test suites for Shadowsocks, transports, system proxy, health checks, stats, updater, and CLI commands

### Changed
- `ParseProxyURL()` now dispatches `ss://` URLs to `ParseShadowsocks()`
- `ToProxyURL()` now exports Shadowsocks profiles to `ss://` SIP002 format
- `buildStreamSettings()` handles h2, httpupgrade, and splithttp transport types
- TransportConfig extended with `maxConcurrentUploads` and `maxUploadSize` fields for SplitHTTP
- Test coverage improved: core 55.1%, platform 55.8%, modals 51.2%, cmd 52.8%

## [0.7.0] - 2026-02-27

### Added
- Interactive onboarding wizard on first launch (protocol selection, URL import, name)
- Color-coded latency indicators next to profile names (green/yellow/red/grey dots)
- Auto-start xray when selecting a profile with Enter (with switch confirmation)
- Profile search and filtering with `/` key in Profiles panel (case-insensitive)
- Visual feedback when reordering profiles (inverted colors on moving item)
- Graceful shutdown: save traffic stats, close SSH tunnels, stop xray on quit
- Config migration system with schema versioning (configVersion: 2)
- Xray version compatibility check with minimum version warning (v1.8.0)
- Improved log rotation based on maxLogSize setting (up to 3 archives)
- Backup rotation with configurable maxFiles limit (default: 5)
- Full DEVNOTES documentation for Phase 6 and 7 features
- Comprehensive README with all features, CLI commands, and keybindings

### Fixed
- Tests no longer fail when xray process is running on the host (mockable findXrayPID)
- Real-time traffic stats not updating in Status panel (rewritten stats query)
- Traffic stats query compatible with both legacy and JSON xray API formats

## [0.6.1] - 2026-02-24

### Fixed
- All Go source files formatted with gofmt (15 files)
- TestTestConnection_UnreachableServer failing on some CI environments (use loopback instead of TEST-NET)
- Windows test failures: cross-platform isProcessAlive via OpenProcess API
- Windows test failures: replace hardcoded /tmp paths with t.TempDir()
- Traffic stats stuck at 0 B in Status panel (use correct xray statsquery API subcommand)

### Changed
- Moved isProcessAlive to platform-specific files (process_unix.go, process_windows.go)
- Refactored isTunnelProcessAlive to delegate to platform isProcessAlive
- Added fmt, check-fmt, test, and lint targets to Makefile

## [0.6.0] - 2026-02-24

### Added
- Subscription management modal in TUI (`S` keybinding) — add, update, delete subscription URLs
- Subscription auto-refresh scheduler at configurable intervals
- CLI subscription import (`lzr import --sub <url>`)
- Collapsible profile groups in Profiles panel (`Space` to toggle)
- Group cycling filter (`g` keybinding)
- Batch latency testing (`T` keybinding) with auto-sort and auto-select fastest profile
- Profile duplication (`Y` in TUI, `lzr config duplicate`)
- VMess protocol support (import, export, config generation)
- Trojan protocol support (import, export, config generation)
- PAC file generation for system proxy configuration
- Speed test functionality
- Encrypted config export
- Routing editor in TUI for per-profile bypass/block rules
- Traffic statistics dashboard
- Theme system with multiple color schemes
- Narrow/responsive layout for small terminals
- Custom keybinding configuration
- Extended test coverage: subscription, routing, crypto, core, platform, CLI, TUI tests

### Changed
- DEVNOTES updated with Phase 6A features and subscription architecture

## [0.5.0] - 2026-02-24

### Added
- Profile editing modal in TUI (`e` keybinding) with 16 editable fields
- Inline profile renaming (`R` keybinding)
- Profile reordering with `Shift+Up/Down` (`K/J`)
- Multi-hop proxy chain builder with chained outbounds
- Connection test before profile activation (TCP connect, 3s timeout)
- QR code export for VLESS profiles (`Q` in TUI, `lzr export --qr`)
- Config diff preview before profile switch (`D` keybinding)
- Self-update command (`lzr self-update`) via GitHub releases
- CLI connection test command (`lzr test [profile]`)
- Notification toggle in settings (`notifications.enabled`)
- Fallback notification tools (terminal-notifier on macOS, dunstify on Linux)
- Integration tests for CLI commands (cmd package)
- Unit tests for TUI message handling, panels, and styles
- golangci-lint in CI pipeline with errcheck, govet, staticcheck

### Fixed
- ANSI `[0m` artifacts on panel borders and traffic display
- Race condition in xray process management (added `sync.RWMutex`)
- Error handling audit: timeouts on downloads, retries on network errors, per-check limits
- SSH tunnel cleanup on unexpected TUI close

### Changed
- Lazy-load logs on startup (tail from end of file instead of full read)
- Cached `GetXrayVersion()` with 60s TTL (no longer called on every status tick)
- Debounced resize handling (100ms) to prevent excessive layout recalculation
- Chain visualization in Status panel (addr:port → addr:port)

## [0.4.0] - 2026-02-23

### Added
- Configurable health check endpoints in application settings
- Profile validation before xray config generation
- Graceful shutdown with SIGTERM and configurable timeout
- Gruvbox-dark color scheme across TUI
- Traffic speed calculation (KB/s) and spinner during operations
- 3-panel layout with hotkeys bar (replaced 4-panel layout)
- Profile deletion with confirmation dialog and `d` keybinding
- Duplicate UUID check on profile import
- Config backup and restore commands (`lzr config backup`, `lzr config restore`)
- Background update check on TUI startup
- Delete keybinding in keymap
- Expanded test coverage for config and core packages
- GitHub Actions CI workflow (build, test, lint on macOS/Linux/Windows)
- GoReleaser configuration with categorized release notes
- MIT license
- Categorized changelog for releases

### Fixed
- `IsRunning()` using `syscall.Signal(0)` instead of broken `os.Signal(nil)`
- Status not updating on TUI restart
- ANSI artifacts in UI output
- Version flag showing "dev" instead of actual version

### Changed
- Rewrote README with updated features and commands

### Removed
- Unused `templates/` directory

## [0.3.0] - 2026-02-23

### Added
- Error and success messages in TUI status bar (auto-clear 10s)
- Keybinding `c` to open xray config in `$EDITOR` with auto-restart
- System notification on health check failure
- Windows/arm64 build target in Makefile
- Auto-restart watchdog for xray process (10s interval, 3 retries)
- Health check and automatic rollback on xray update failure
- Traffic statistics via xray stats API
- Persistent CLI tunnels with PID file tracking
- Unit tests for VLESS parser, config generator, paths, and health checks
- VLESS URL export command (`lzr export`) and TUI keybinding (`x`)
- Per-profile routing rules for bypass and block lists
- Log rotation before xray startup (10 MB default)
- Development notes documentation

### Fixed
- Xray config regeneration and restart on profile switch
- Settings values used instead of hardcoded defaults

### Changed
- Removed dead code and fixed duplication

## [0.2.0] - 2026-02-23

### Added
- Background mode for `lzr start` with PID file tracking
- Panel titles embedded in border with focus indicator
- Log filtering (`f`) and search (`/`) with highlighting
- Enhanced health check modal with DNS leak test and async init
- Enhanced SSH tunnel modal with number keys and command display
- Platform asset name display in update modal
- Real-time status updates with async exit IP and latency
- Structured `lzr status --json` output

## [0.1.0] - 2026-02-23

### Added
- Go module and build system (Makefile with cross-compilation)
- Configuration models for paths, servers, and settings
- Core engine for xray process management, VLESS URL parsing, and health checks
- Platform abstraction for macOS, Linux, and Windows (autostart, notifications)
- Cobra CLI with all headless commands (start, stop, restart, status, health, import, config, update, tunnel, service)
- Bubble Tea TUI with 4-panel lazygit-style layout
- README and config templates

### Fixed
- Xray stderr capture to show actual error when process exits immediately
- Extraction of geoip.dat and geosite.dat from xray zip archive
- TUI visual bugs in modal overlay, logs, and panel sizing

<!-- Versions before 1.0.0 predate this repository's history: there are no
     tags to link, so those sections are intentionally unlinked. -->
[Unreleased]: https://github.com/rtxnik/lazyray/compare/v1.0.0...HEAD
[1.0.0]: https://github.com/rtxnik/lazyray/releases/tag/v1.0.0
