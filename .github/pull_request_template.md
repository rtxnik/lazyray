## Summary

<!-- What does this change do, and why? -->

## Change type

- [ ] Bug fix
- [ ] Feature
- [ ] Refactor / internal cleanup
- [ ] Documentation
- [ ] Build / CI

## Testing

<!-- How did you verify this? Commands run, platforms exercised. -->

## Checklist

- [ ] PR title follows [Conventional Commits](https://www.conventionalcommits.org/) (`type: subject`) — it becomes the squash commit subject
- [ ] `make lint` passes (gofmt + go vet + golangci-lint)
- [ ] `make test` passes
- [ ] If a command, flag, or keybinding changed: ran `make docs` and committed the regenerated reference
- [ ] Updated `CHANGELOG.md` under `[Unreleased]` if user-facing, or the `no-changelog` label applies (maintainer sets it if you cannot)
- [ ] Exactly one `type/*` label set (release notes group by it) — maintainer applies it if you lack permission
- [ ] Bug fix: linked the issue with `Closes #N`
- [ ] Read `CONTRIBUTING.md`
