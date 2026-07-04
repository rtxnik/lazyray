# Releasing

Maintainer runbook for cutting a lazyray release. The release act is pushing a
`v*` tag: tag creation is restricted to the repository admin, and the `release`
environment holds the signing secrets, so nobody else can trigger this path.

## Cadence

- On-demand, with a soft monthly ceiling: release when at least one
  user-visible PR has merged AND (4 weeks have passed since the last release
  OR a user-blocking fix is waiting).
- A confirmed regression or security fix ships as a patch release within 48 hours.
- Any open issue or PR labeled `blocks-release` halts every release until
  resolved (the preflight enforces this).
- `-rc.N` prereleases only before a major or a state-migration release.
- The milestone for the target minor is the ordering unit: drain or retarget it
  before releasing.

## Cutting vX.Y.Z

1. **Release-prep PR** (a normal PR; all gates apply):
   - Move the `[Unreleased]` content of `CHANGELOG.md` into a new
     `## [X.Y.Z] - YYYY-MM-DD` section, keeping an empty `[Unreleased]`
     section on top.
   - Update the link block at the bottom: `[Unreleased]` compares
     `vX.Y.Z...HEAD`, and add
     `[X.Y.Z]: https://github.com/rtxnik/lazyray/compare/v<prev>...vX.Y.Z`.
   - Title: `chore: prepare vX.Y.Z release`.
2. **Merge it** (squash; checks green; branch up to date).
3. **Preflight locally** from an up-to-date `main` checkout — all checks must
   pass BEFORE the tag exists; a bad tag caught here costs nothing, a pushed
   tag triggers the pipeline:

   ```bash
   git checkout main && git pull
   scripts/repo-governance/preflight.sh vX.Y.Z
   ```

4. **Tag and push** (this is the release trigger and the human gate):

   ```bash
   git tag -a vX.Y.Z -m "vX.Y.Z"
   git push origin vX.Y.Z
   ```

5. **Watch the pipeline** — the `Release` workflow re-runs the same preflight
   as a gate job, then GoReleaser builds, signs (minisign over
   `checksums.txt`), and publishes with notes generated from PR labels:

   ```bash
   gh run watch "$(gh run list --workflow release.yml -L 1 --json databaseId --jq '.[0].databaseId')" --exit-status
   ```

6. **Verify as a user** (from a scratch directory):

   ```bash
   curl -fsSLO https://raw.githubusercontent.com/rtxnik/lazyray/main/scripts/install.sh
   bash install.sh --require-signature
   lzr --version
   ```

## If the pipeline fails

Nothing was published (GoReleaser aborts before the release goes public): fix
the cause on `main` via a normal PR. If the tag itself is wrong, delete it and
re-tag after the fix (the admin tag bypass exists for exactly this):

```bash
git push origin :refs/tags/vX.Y.Z
```

If a bad release WAS published: fix forward — patch release within 48 hours,
add a warning note to the broken release, never delete a published release.

## Emergency path (bypassing the PR gate on main)

There is deliberately NO standing bypass on `main` — including for the
maintainer. If an emergency truly requires a direct push:

1. Temporarily disable enforcement on the `main-protection` ruleset
   (Settings -> Rules -> Rulesets -> main-protection -> Enforcement status ->
   Disabled).
2. Make the minimal push.
3. Re-enable enforcement immediately and run
   `scripts/repo-governance/check.sh full` — it must report OK.
4. Record what happened and why in the maintainer decision log.

The weekly governance drift check goes red while enforcement is off. That is by
design: silently disabled protection must be impossible.
