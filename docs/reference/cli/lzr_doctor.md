## lzr doctor

Diagnose the lazyray installation, config, and connectivity

### Synopsis

Diagnose the lazyray installation, profile store, and connectivity.

doctor runs grouped diagnostics over your environment — the xray-core binary and
its geoip/geosite data, the profile store and default proxy profile, the
supervisor and system proxy, and basic connectivity — and prints each finding
with a hint. It is the home for diagnostics; most error hints from other
commands point here. A non-zero exit means problems were found, which is the
normal outcome of a failing gate rather than a usage error.

```
lzr doctor [flags]
```

### Examples

```
  # Run all diagnostics
  lzr doctor

  # Fail (non-zero exit) on warnings too, for use in CI
  lzr doctor --strict

  # Machine-readable report for scripts
  lzr doctor --json
```

### Options

```
  -h, --help     help for doctor
      --json     Print the diagnostics report as JSON instead of human-readable text
      --strict   Treat warnings as failures, so any warning yields a non-zero exit
```

### SEE ALSO

* [lzr](lzr.md)	 - Terminal UI for managing Xray-core proxy configurations

