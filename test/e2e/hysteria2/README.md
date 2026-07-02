# Hysteria2 e2e harness

Stands up a real apernet/hysteria server (salamander obfs, self-signed cert) plus
a `whoami` target, drives lazyray's xray client through the tunnel, and asserts
egress. The client trusts the self-signed cert via **`pinSHA256`** (xray-core
>= v26 removed `allowInsecure`, so pinning is the supported path).

## Run locally

```sh
./gen-cert.sh                       # writes certs/ + prints the client pinSHA256
docker compose up -d
XRAY_BIN=$(command -v xray) go test -tags e2e ./test/e2e/hysteria2/ -v
docker compose down
```

Requires: `docker`, `openssl`, an `xray` binary (`XRAY_BIN`), and a built `lzr`
(`LZR_BIN`, or the test builds it). Certs are ephemeral and git-ignored.

### xray geo data

The configs lazyray generates contain routing rules that reference `geoip:private`,
so xray needs `geoip.dat` + `geosite.dat` next to the binary (they ship inside the
XTLS/Xray-core release zip — extract them alongside `xray`). Without them xray
fails to load the config with `failed to open file: geoip.dat`.

## Port-hopping

The client side (xray `finalmask.quicParams.udpHop.ports`) is covered by the unit
tests and the structural `xray run -test` check. A full live port-hopping tunnel
additionally requires server-side firewall plumbing — an `iptables`/`nftables`
redirect of the whole UDP range to the listen port, which is an operator
deployment concern — so it is not run as a hermetic CI test here.

## Hardening

Pin images by digest before merge:

```sh
docker compose pull && docker images --digests | grep -E 'hysteria|whoami'
```

Replace the `:tag` refs in the compose files with `@sha256:<digest>`.
