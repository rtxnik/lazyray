# Stability and versioning

lazyray follows [Semantic Versioning 2.0.0](https://semver.org/spec/v2.0.0.html).
This document defines exactly which surfaces the version number protects.

`lzr` is a program, not a library: the Go packages in this module are not a
public API and carry no compatibility promise.

## Covered surface (a breaking change here requires a major release)

| Surface | Notes |
|---|---|
| Subcommands and positional arguments | Removing or renaming a command or argument is breaking. |
| Flags | Removing or renaming a flag, or changing a default in a behavior-visible way, is breaking. |
| Environment variables | Documented variables such as `XRAY_BIN`. |
| Config file schema | `servers.yaml`, `lazyray.yaml`. Schema changes ship with an automatic migrator (see below). |
| State files | Locations and formats of PID files, logs, and backups. |
| Exit codes | As documented in [docs/reference/exit-codes.md](reference/exit-codes.md). |
| `--json` output | Removing, renaming, or retyping a field is breaking; adding fields is minor. |

## Not covered (may change in any release)

- TUI rendering, layout, colors, and themes
- Default keybindings (rebindable; default changes get a CHANGELOG note)
- Human-readable text: log lines, error messages, help text
- Performance characteristics
- Content of generated docs, completions, and man pages
- Go package APIs inside this module

## Release rules

- **Major** — any breaking change to a covered surface.
- **Minor** — new features; config-format changes that ship with an automatic
  migrator (the old format keeps loading); additive `--json` fields.
- **Patch** — bug and security fixes only.

Deprecations follow [docs/deprecations.md](deprecations.md).

## Go toolchain policy

Target policy: `lzr` builds with the two most recent Go releases, and `go.mod`
declares the older of the two. Current state: the module declares Go 1.26.4 and
CI builds with exactly that version; the two-release window is adopted at the
next Go minor release.

## Support window

Latest release only, fix-forward: fixes ship as a new release cut from `main`,
never as backports ([SECURITY.md](../SECURITY.md#supported-versions)).
