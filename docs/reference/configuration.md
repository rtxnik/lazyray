# Configuration Reference

lazyray reads two YAML files from your config directory:

| OS | Config directory |
|----|------------------|
| Linux / macOS | `~/.config/lazyray/` |
| Windows | `%APPDATA%\lazyray\` |

- `lazyray.yaml` — application settings (this file is optional; defaults apply when absent).
- `servers.yaml` — proxy profiles.
- `keys.yaml` — optional keybinding overrides; see the [keybindings reference](keybindings.md).

## `lazyray.yaml` — application settings

### `local` — local proxy listeners

| Field | Type | Default | Notes |
|-------|------|---------|-------|
| `socksPort` | int | `10808` | local SOCKS5 listener port |
| `httpPort` | int | `10809` | local HTTP proxy listener port |
| `listen` | string | `127.0.0.1` | listener bind address |
| `dns` | list | `[1.1.1.1, 8.8.8.8]` | upstream DNS servers |

### `xray` — engine behavior

| Field | Type | Default | Notes |
|-------|------|---------|-------|
| `autoRestart` | bool | `true` | restart xray on unexpected exit |
| `logLevel` | string | `warning` | xray log level |
| `maxLogSize` | int | `10` | log size (MB) that triggers rotation |

### `health` — health checks

| Field | Type | Default | Notes |
|-------|------|---------|-------|
| `timeout` | int | `5` | health-probe timeout (seconds) |
| `alertOnFailure` | bool | `true` | notify on health failure |
| `ipCheckURL` | string | `https://ifconfig.me/ip` | exit-IP probe endpoint |
| `latencyHost` | string | `1.1.1.1:443` | latency probe target |
| `dnsCheckHost` | string | `dns.google:443` | DNS-leak probe target |

### `update` — xray-core updates

| Field | Type | Default | Notes |
|-------|------|---------|-------|
| `channel` | string | `stable` | update channel |
| `autoCheck` | bool | `true` | check for updates automatically |
| `backupBefore` | bool | `true` | back up the binary before updating |
| `xrayVersion` | string | `v26.3.27` | pinned xray-core version fetched by `lzr update apply` |

### `ui` — terminal UI

| Field | Type | Default | Notes |
|-------|------|---------|-------|
| `theme` | string | `dark` | TUI theme (e.g. dark, gruvbox, nord, catppuccin, solarized) |
| `refreshInterval` | int | `5` | status refresh interval (seconds) |
| `logLines` | int | `100` | log lines kept in the Logs panel |

### `notifications`

| Field | Type | Default | Notes |
|-------|------|---------|-------|
| `enabled` | bool | `true` | system notifications on/off |

### `backup`

| Field | Type | Default | Notes |
|-------|------|---------|-------|
| `maxFiles` | int | `5` | number of rotated backup files to keep |

### Top-level settings

| Field | Type | Default | Notes |
|-------|------|---------|-------|
| `autoSystemProxy` | bool | `true` | enable the OS system proxy automatically when xray starts |
| `subscriptions` | list | — | managed subscriptions (see below) |

### `subscriptions[]` — subscription entries

| Field | Type | Default | Notes |
|-------|------|---------|-------|
| `name` | string | — | display name |
| `url` | string | — | subscription URL |
| `interval` | int | — | hours between auto-refresh |

## `servers.yaml` — proxy profiles

| Field | Type | Default | Notes |
|-------|------|---------|-------|
| `configVersion` | int | `2` | schema version; older files are auto-migrated (see Migration) |
| `profiles` | list | — | the proxy profiles |

### `profiles[]` — a profile

| Field | Type | Notes |
|-------|------|-------|
| `name` | string | profile name |
| `default` | bool | the active profile when set |
| `server` | object | the proxy server (see below) |
| `chain` | list | optional multi-hop chain of additional servers (entry → exit) |
| `ssh` | object | optional SSH tunnel settings |
| `expectedExitIP` | string | expected exit IP, checked by health |
| `routing` | object | per-profile routing rules |
| `group` | string | optional group label for filtering |
| `tags` | list | optional tags |
| `latency` | int | last measured latency (ms); `-1` = failed, set by batch test |
| `subscription` | string | source subscription URL, if imported from one |

### `server` / `chain[]` — `ServerConfig`

| Field | Type | Notes |
|-------|------|-------|
| `address` | string | server host/IP |
| `port` | int | server port |
| `uuid` | string | user UUID (VLESS/VMess); password for Trojan/Shadowsocks |
| `encryption` | string | protocol encryption (`none` for VLESS, `auto` for VMess) |
| `flow` | string | VLESS flow; empty for XHTTP |
| `protocol` | string | `vless` (default), `vmess`, `trojan`, `shadowsocks`, `hysteria2` |
| `alterId` | int | VMess only |
| `obfs` | string | Hysteria2 only — obfuscation type (`salamander`) |
| `obfsPassword` | string | Hysteria2 only — obfuscation password |
| `portHopping` | string | Hysteria2 only — port spec, e.g. `443,5000-6000` |
| `portHopInterval` | string | Hysteria2 only — hop interval, e.g. `30` or `10-30` |
| `transport` | object | transport settings (see below) |
| `security` | object | TLS/Reality settings (see below) |

### `transport` — `TransportConfig`

| Field | Type | Notes |
|-------|------|-------|
| `network` | string | `tcp`, `ws`, `grpc`, `xhttp`, `h2`, `httpupgrade`, `splithttp` |
| `path` | string | transport path |
| `mode` | string | transport mode (e.g. xhttp `auto`) |
| `host` | string | host header |
| `maxConcurrentUploads` | int | SplitHTTP only |
| `maxUploadSize` | int | SplitHTTP only |

### `security` — `SecurityConfig`

| Field | Type | Notes |
|-------|------|-------|
| `type` | string | `reality`, `tls`, or empty |
| `sni` | string | server name |
| `fingerprint` | string | TLS fingerprint (e.g. `chrome`) |
| `publicKey` | string | Reality public key |
| `shortId` | string | Reality short ID |
| `spiderX` | string | Reality spiderX path |
| `alpn` | string | optional ALPN list |
| `allowInsecure` | bool | **deprecated** — removed in xray-core ≥ v26; use `pinSHA256` instead |
| `pinSHA256` | string | Hysteria2 cert pin — comma-separated hex SHA-256 of the cert DER |

### `ssh` — `SSHConfig`

| Field | Type | Notes |
|-------|------|-------|
| `host` | string | SSH host |
| `port` | int | SSH port |
| `user` | string | SSH user |
| `keyPath` | string | path to the private key |
| `hostKeys` | list | pinned SSH host public keys (`"<type> <base64>"` entries); empty means TOFU mode |
| `panel` | object | admin-panel target: `port`, `path` |

### `routing` — `ProfileRouting`

| Field | Type | Notes |
|-------|------|-------|
| `bypass` | list | rules routed direct (e.g. `geoip:private`) |
| `block` | list | rules blocked (e.g. `geosite:category-ads-all`) |
| `dnsRules` | list | conditional DNS rules: `server`, `domains`, `expectIPs` |

## Protocol applicability

| Field(s) | Applies to |
|----------|-----------|
| `alterId` | VMess |
| `obfs`, `obfsPassword`, `portHopping`, `portHopInterval`, `pinSHA256` | Hysteria2 |
| `allowInsecure` | deprecated (removed in xray-core ≥ v26 — prefer `pinSHA256`) |

## Migration

`servers.yaml` carries a `configVersion` (current: `2`). When an older file is loaded, lazyray backs it up to `servers.yaml.v<N>.bak` and fills in protocol-appropriate defaults (`protocol`, `encryption`, `transport.network`) before saving the upgraded file.
