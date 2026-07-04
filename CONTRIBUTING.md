# Contributing to lazyray

Thanks for taking the time to contribute. lazyray is a terminal proxy client
(`lzr`) written in Go. This guide covers how to set up a development
environment, build and test the project, and get a change merged. By
participating you agree to abide by the [Code of Conduct](CODE_OF_CONDUCT.md).

## Development setup

You need Go 1.26.4 or newer — the version `go.mod` declares and CI builds with
(toolchain policy: [docs/stability.md](docs/stability.md)). Docker is only
required for the end-to-end suite (see below).

```bash
git clone https://github.com/rtxnik/lazyray.git
cd lazyray
make build      # produces ./lzr in the repo root
./lzr --help
```

## Build, test, lint

| Command | What it does |
|---------|--------------|
| `make build` | Build the `lzr` binary into the repo root. |
| `make test` | Run the unit suite (`go test ./...`). |
| `make lint` | Run `check-fmt` then `golangci-lint run ./...`. |
| `make check-fmt` | Fail if any file is unformatted (`gofmt -l .`). |
| `make docs` | Regenerate the generated reference, completions, and man pages. |
| `make test-e2e` | Run the Docker-based hysteria2 end-to-end suite. |

## Before you push — the gate

Run `make lint` before every push. At a minimum run `make check-fmt`.

**`go build` and `go vet` do not catch `gofmt` formatting drift.** CI runs
`gofmt -l .`, `go vet ./...`, and `golangci-lint` as three separate gates, so a
file that builds and vets cleanly will still fail CI if it is not gofmt-clean.
`make lint` runs the formatting check first, which is why it is the command to
reach for.

Test-driven development is the norm here: write the failing test first, watch it
fail, then make it pass. New behavior should arrive with the test that pins it.

## Generated documentation

These paths are generated, not hand-edited:

- `docs/reference/cli/*.md` — per-command reference
- `docs/reference/keybindings.md` — TUI keybinding reference
- `completions/` — shell completions
- `man/man1/` — man pages

They are produced by `make docs`. If you change a command, a flag, or a
keybinding, run `make docs` and commit the regenerated output in the same PR.
CI's docs-freshness gate uses `git status --porcelain`, so even a brand-new
command whose reference page has never been committed will fail the gate until
you commit it.

## Architecture and invariants (the Guards)

Start with [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md). The
[Code Map](docs/ARCHITECTURE.md#code-map) tells you which package owns what, and
[Invariants & Guards](docs/ARCHITECTURE.md#invariants--guards) describes the five
architectural rules that are enforced by guard tests.

If a guard test starts failing when you add a keybinding, command, color, or
panel, you have tripped one of those five invariants. See
[Invariants & Guards](docs/ARCHITECTURE.md#invariants--guards) for what each one
protects and how to satisfy it, rather than working around the test.

The most common change — adding a TUI command — has a short recipe: add a
`KeyMap` field and a matching `Registry` command with the identically named ID,
wire an `a.keys.<ID>` dispatch branch in `app.go`, bind a single-rune primary
key if the command is palette-launchable, and map its `Scope` in `panelForScope`
if it focuses a panel. The full per-guard checklist lives in
[Extension Points](docs/ARCHITECTURE.md#extension-points).

## Commit and pull-request process

- Branch from `main`, named `<type>/<short-description>` using the same types
  as commit prefixes — for example `fix/tunnel-cleanup`, `feat/pac-flag`,
  `docs/stability-policy`.
- Use [Conventional Commits](https://www.conventionalcommits.org/) for commit
  messages (for example `feat:`, `fix:`, `docs:`, `refactor:`).
- **The PR title must follow Conventional Commits** — the required `PR title`
  check enforces it, and squash merges use the title verbatim as the commit
  subject on `main`. Intra-PR commit messages are local hygiene, not enforced.
- Open the PR against `main`. The
  [pull request template](.github/pull_request_template.md) lists the
  checklist that has to pass.
- Keep PRs focused and reviewable — one logical change per PR, as a guideline
  under ~400 changed lines. Split bigger work into a series.
- Update the `[Unreleased]` section of [CHANGELOG.md](CHANGELOG.md) for any
  user-facing change — the required `Changelog OK` check enforces this. For
  changes with no user-visible effect (refactors, CI), apply the
  `no-changelog` label instead.
- Bug-fix PRs reference their issue with `Closes #N`.
- Add exactly one `type/*` label — release notes are grouped by it.
- Releases are cut from `main` by the maintainer: [docs/releasing.md](docs/releasing.md).

## End-to-end tests

`make test-e2e` runs the hysteria2 harness. It needs:

- Docker (the harness brings up a real
  [apernet/hysteria](https://github.com/apernet/hysteria) server via
  `docker compose`).
- An `xray` binary on `PATH`, or set `XRAY_BIN` to point at one. The directory
  containing that binary must also contain `geoip.dat` and `geosite.dat`,
  because lazyray's generated routing references `geoip:private`.

The suite imports a `hysteria2://` link with a `pinSHA256` certificate pin,
starts the xray client, and asserts that traffic egresses through the tunnel —
exercising the same certificate-pinning path that production profiles use.

## Security

Do not file security vulnerabilities as public issues. See
[SECURITY.md](SECURITY.md) for the private reporting process.
