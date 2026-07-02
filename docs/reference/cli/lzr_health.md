## lzr health

Run health check on proxy connection

### Synopsis

Run a connectivity probe against the default proxy profile.

health runs a sequence of checks (xray-core startup, handshake, and exit-IP
reachability) through the proxy server the profile points to and reports which
passed. It is a fast connectivity probe — for full installation and config
diagnostics use 'lzr doctor' instead.

```
lzr health [flags]
```

### Examples

```
  # Probe the default proxy profile
  lzr health

  # Machine-readable output for scripts
  lzr health --json
```

### Options

```
  -h, --help   help for health
      --json   Print the health report as JSON instead of human-readable text
```

### SEE ALSO

* [lzr](lzr.md)	 - Terminal UI for managing Xray-core proxy configurations

