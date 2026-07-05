# Security Policy

lazyray (`lzr`) handles credentials, server addresses, and a local proxy
datapath. We take vulnerability reports seriously and aim to be honest about
both what the project protects today and what it does not yet protect.

## Reporting a Vulnerability

**Please report security issues privately. Do not open a public issue for a
suspected vulnerability.**

Primary channel — GitHub Private Vulnerability Reporting:

  https://github.com/rtxnik/lazyray/security/advisories/new

Private Vulnerability Reporting is the only private security channel for this
project — there is deliberately no contact e-mail. If you cannot use it, open a
regular issue stating only that you need a private channel for a security
report — no technical details — and the maintainer will arrange one.

### Before you send anything

`lzr` configuration and runtime state contain secrets. **Do not paste
credentials, subscription URLs, server addresses, UUIDs, or private keys into a
report.** Redact them first. A report does not need any of these to be
actionable.

### What to include

- A clear description of the issue and its security impact.
- Redacted `lzr doctor` output (it summarizes environment and configuration
  health without printing secrets; double-check and redact anything sensitive
  before sending).
- `lzr --version`.
- Your OS and architecture.
- The minimal steps to reproduce. A proof of concept that uses made-up
  values (e.g. `example.com`, a dummy UUID) is preferred over anything drawn
  from a real deployment.

## Supported Versions

`lzr` keeps itself current through `lzr update` / `lzr self-update`, so only the
latest released version is supported. Older versions receive no security fixes —
upgrade to the latest release.
Fixes ship **fix-forward**: a new release cut from `main`, never a backport.
A confirmed vulnerability or regression targets a patch release within
48 hours of confirmation.

| Version  | Supported |
| -------- | --------- |
| `latest` | ✅        |
| older    | ❌        |

Cryptographically signed releases are the norm for this project: releases are
signed with the production minisign key, and the signing path is additionally
exercised in CI with an ephemeral key on every pull request. See **Trust Model
& Release Verification** below for exactly what verification covers today.

## Disclosure Process

lazyray is maintained by a single maintainer. The timelines below are honest
targets, not a contractual SLA:

- **Acknowledgement:** within **72 hours** of a report.
- **Coordinated disclosure:** we aim to release a fix and publish an advisory
  within **~90 days** of acknowledgement, sooner when a fix is straightforward.
- **Embargo:** please keep the report private until a patched release is
  available and the advisory is published.

**Safe harbor:** we will not pursue or support legal action against good-faith
security research that respects this policy and the privacy of users — that is,
research that avoids accessing or modifying others' data, gives us a reasonable
chance to remediate before disclosure, and stays within the scope below.

## Trust Model & Release Verification

`lzr` ships and updates itself over a few mechanisms. Each is described here
with what it protects **and** what it does not, so you can calibrate your own
risk.

### Release signatures (minisign over `checksums.txt`)

The intended root of trust is a published minisign public key. At release time a
single minisign signature is produced over `checksums.txt`, which itself
enumerates the SHA-256 of every archive and package — so one signature
transitively authenticates all artifacts.

**State of the world today:** the signing pipeline is live. Releases are signed
with the project's production minisign key (embedded in `internal/release/verify.go`
and `scripts/install.sh`), and the signature is exercised on every pull request
with an ephemeral CI key. `lzr self-update` and `install.sh --require-signature`
verify the signature against the embedded key and fail closed — refusing to
proceed rather than trust an unsigned or tampered artifact.

This mechanism does **not** provide SLSA build provenance, reproducible builds,
a transparency log, or per-artifact signatures. It authenticates the checksum
manifest, nothing more.

### Pinned, checksum-verified xray-core download

`lzr` does not bundle Xray-core in the binary; it downloads it from a pinned
upstream tag (default `v26.3.27`). Before the downloaded archive is extracted or
made executable, its SHA-256 is verified against the XTLS-published `.dgst`
checksum for that asset. Verification fails closed — a mismatch aborts the
update and the archive is never run.

This does **not** defend against a wholly compromised upstream XTLS release: the
`.dgst` file rides the same release channel as the archive, so an attacker who
controls that release controls both. It also does **not** pin GitHub's TLS
certificate — transport security relies on the system certificate store.

### Verified self-update

The self-updater verifies the minisign signature over `checksums.txt` and the
downloaded archive's SHA-256 against that manifest **before** atomically
replacing the running binary. The verifier is pure: it performs no network I/O
and never calls `os.Exit`, so it cannot be coerced into skipping a check or
tearing down the process mid-update.

As above, this verification is only as strong as the embedded release key. It tracks
the `latest` release and provides **no downgrade protection** — it will move to
whatever the latest published release is.

### Local handling

- Credential-bearing files (`servers.yaml`, the generated `config.json`) are
  written with `0600` permissions. The settings file (`lazyray.yaml`) holds no
  secrets and is `0644`; config directories are `0755`.
- The proxy binds `127.0.0.1` by default, and the local stats API is bound to
  `127.0.0.1`.
- The background service is **user-scoped and never runs as root**.
- Profiles can be exported encrypted with AES-256-GCM (key derived via PBKDF2).

**Known limitation to disclose:** SSH tunnels currently run with
`StrictHostKeyChecking=no`, which leaves them exposed to host-key
man-in-the-middle attacks. Config directories are `0755` (world-readable
metadata, though the credential files within are `0600`).

## Scope

**In scope:**

- The `lzr` binary itself.
- The update / self-update / verification path.
- Configuration and credential handling.
- The proxy datapath.

**Out of scope:**

- Vulnerabilities in upstream **Xray-core** — report those to the
  [XTLS/Xray-core](https://github.com/XTLS/Xray-core) project. lazyray pins a
  known, reviewed version (default `v26.3.27`).
- User misconfiguration (e.g. weak passwords, exposing the proxy on a public
  interface, disabling verification flags).

## Verifying a release

See the [Installation](README.md#installation) section of the README for how the
Linux-package and `install.sh` paths verify artifacts, including the
`install.sh --require-signature` flag that makes a missing `minisign` a hard
failure.
