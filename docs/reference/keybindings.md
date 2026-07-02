# Keybindings Reference

Default keybindings for the lazyray TUI. Override any of them in `keys.yaml` in your config directory (see the configuration reference). The **Palette** column marks commands launchable from the command palette (`:`).

## General & Navigation

| Key | Action | Scope | Palette |
|-----|--------|-------|----------|
| `q` | quit | Global | yes |
| `?` | help | Global | yes |
| `n` | activity log | Global | yes |
| `:` | command palette | Global |  |
| `tab` | next panel | Global |  |
| `shift+tab` | prev panel | Global |  |
| `up/k` | move up / scroll | Global |  |
| `down/j` | move down / scroll | Global |  |
| `enter` | select / activate | Profiles |  |
| `esc` | close / cancel | Global |  |
| `space` | collapse / expand group | Profiles |  |
| `m` | toggle dashboard metric | Status | yes |

## Connection

| Key | Action | Scope | Palette |
|-----|--------|-------|----------|
| `s` | start / stop | Global | yes |
| `r` | restart | Global | yes |
| `t` | SSH tunnel | Global | yes |
| `T` | test all latency | Profiles | yes |

## Profiles

| Key | Action | Scope | Palette |
|-----|--------|-------|----------|
| `i` | import config | Global | yes |
| `S` | subscriptions | Global | yes |
| `Y` | duplicate profile | Profiles | yes |
| `R` | rename profile | Profiles | yes |
| `d` | delete profile | Profiles | yes |
| `g` | filter group | Profiles | yes |
| `shift+up/K` | move profile up | Profiles |  |
| `shift+down/J` | move profile down | Profiles |  |

## Inspect & Export

| Key | Action | Scope | Palette |
|-----|--------|-------|----------|
| `h` | diagnostics | Global | yes |
| `x` | export VLESS URL | Global | yes |
| `Q` | QR code export | Profiles | yes |
| `D` | config diff | Profiles | yes |
| `W` | routing rules | Profiles | yes |

## Config & Logs

| Key | Action | Scope | Palette |
|-----|--------|-------|----------|
| `c` | edit config | Global | yes |
| `u` | update xray | Global | yes |
| `e` | edit profile / toggle log | Profiles | yes |
| `f` | filter logs | Logs | yes |
| `/` | search | Logs | yes |

## Rebinding keys

Set any of these fields in `keys.yaml` to override the default binding:

- `quit`
- `start`
- `restart`
- `doctor`
- `tunnel`
- `import`
- `update`
- `editConfig`
- `export`
- `delete`
- `rename`
- `qrExport`
- `configDiff`
- `toggleLog`
- `filterLog`
- `searchLog`
- `help`
- `subscriptions`
- `filterGroup`
- `testAll`
- `duplicate`
- `routingEdit`
- `activity`
- `palette`
- `toggleMetric`

`health` is a backward-compatible alias for `doctor`.
