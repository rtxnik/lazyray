# Signing-key custody and rotation

This runbook covers how the release-signing key is held, how to rotate it, and
what to do if it is compromised. It documents a **capability**: the verifier and
installer already accept an embedded *trust-list* of keys, so a rotation is a
signing-key swap, not a client-breaking change. No rotation is performed until a
trigger (a compromise, or a deliberate decision) actually requires one.

## How trust is rooted

- The binary embeds an ordered trust-list, `releaseSigningPubKeys`, in
  `internal/release/verify.go`. A release verifies if **any** key in the list
  verifies the signed `checksums.txt` (accept-any); verification fails closed if
  none do.
- `scripts/install.sh` embeds the same list as `RELEASE_PUBKEYS` (one key per
  line) and applies the same accept-any rule. The installer is always fetched
  fresh, so new installs already track the current key; the rotation problem is
  only the in-binary self-update path of already-installed binaries.
- There is a single canonical signature asset, `checksums.txt.minisig`. The
  self-update download path never changes across a rotation — only the key that
  produced that file changes.

## Key custody

- [OWNER] The signing private key and its passphrase must live in a password
  manager, not in cleartext on disk. Move both out of
  `~/lazyray-signing-backup/{lazyray.key, password.txt}` and delete the on-disk
  copies. Keep one offline backup of the current key (needed to sign an overlap
  release during a planned rotation), storing the passphrase separately from the
  key material.
- [OWNER] The live signing copy is the `MINISIGN_SECRET_KEY` secret in the
  `release` GitHub Environment, gated to `v*` tags. That gate is the primary
  custody control and stays in place; the private key is never present in an
  ordinary CI job.
- Cadence: rotate on a trigger, not on a fixed clock. The trust-list only grows;
  an old key is removed only after a deliberate "the tail of binaries still on
  the old key is negligible" decision, or when a compromise forces it.

## Planned rotation (introduce a new key without breaking clients)

Each phase is a normal release; [OWNER] cuts the tag and swaps the environment
secret.

1. **Phase 0 - ship the capability (already done).** A release whose verifier
   and installer accept any key in the embedded list; the list holds only the
   current key. No user-visible change; the binary is now rotation-ready.
2. **Phase 1 - introduce the new key.** [OWNER] generate the new key. Add its
   public key as a second entry in `releaseSigningPubKeys` (verify.go) and in
   `RELEASE_PUBKEYS` (install.sh), keeping them byte-in-sync, and ship a normal
   release. Keep signing releases with the **current** key: the new key must be
   distributed to clients *before* it is ever used to sign. Adoption of this
   release begins the overlap window. Before shipping, confirm both embedded
   lists still parse and that `go build ./...` and a test `minisign -V` against
   each new key succeed: the verifier reads every embedded key up front and
   fails closed on the first malformed entry, so one mistyped key would break
   verification of all releases.
3. **Phase 2 - flip the signer.** [OWNER] switch the `release` environment's
   `MINISIGN_SECRET_KEY` from the old key to the new key. Publish under the
   **same** `checksums.txt.minisig` filename. Binaries at Phase 1 or later verify
   via the new key; the download path is untouched.
4. **Phase 3 - retire the old key.** In a later release, once the population of
   binaries still trusting only the old key is judged negligible, drop the old
   key from `releaseSigningPubKeys` and `RELEASE_PUBKEYS`.

### Observable pre-transition failure mode

At the Phase 2 flip, a binary that embeds only the old key downloads
`checksums.txt.minisig` (now signed by the new key), finds no trusted key that
verifies it, and self-update **refuses and leaves the running binary untouched**
(fail-closed, exactly as today). Such a user recovers by reinstalling with the
always-current installer:

    curl -fsSLO https://raw.githubusercontent.com/rtxnik/lazyray/main/scripts/install.sh
    sh install.sh --require-signature

This tail is unavoidable for any system whose trust root is frozen into shipped
binaries, and is the reason the capability (Phase 0) ships as early as possible.

## Key compromise

Rotation **alone is not a sufficient** compromise response: a leaked key cannot
be remotely un-trusted in binaries that already shipped with it, and rotation
only helps if the legitimate holder moves faster than the attacker.

1. [OWNER] Treat the leaked key as burned immediately. Publish a security
   advisory and a GitHub Security Advisory (private vulnerability reporting is
   already enabled).
2. [OWNER] Generate a new key and run the planned-rotation phases as fast as
   adoption allows (add the new key, then flip). Binaries that trust only the
   burned key cannot be protected this way and must reinstall.
3. **Build-provenance attestation is the independent recovery root.** Release
   provenance is keyless: its trust is anchored in the CI workflow identity and
   the public transparency log, cryptographically and operationally disjoint from
   the maintainer-held signing key. An attacker with the leaked signing key still
   cannot forge an attestation without also compromising CI. While the signing
   key is burned, users and the installer can still confirm authenticity via
   `gh attestation verify` and the post-release verify job. Keep both roots:
   signing hedges against CI compromise, attestation against signing-key
   compromise.
4. Publishing two signatures on one release (one per key) is **not** the default
   response; reserve it only as an emergency fast-path if both keys must be valid
   on a single release without a client update. The default path keeps the single
   canonical `checksums.txt.minisig` filename.

## Scope note

Replay/downgrade protection (accepting an older, validly-signed artifact) is a
separate concern from key rotation and is tracked independently; this runbook
does not address it.
