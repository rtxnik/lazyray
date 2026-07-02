## lzr update apply

Download and install pinned xray-core

### Synopsis

Download and install the pinned xray-core release for this platform, replacing the current xray-core binary. Use --version to install a specific release tag instead of the one pinned in settings. Run this when 'lzr doctor' reports xray-core is missing or too old.

```
lzr update apply [flags]
```

### Examples

```
  lzr update apply
  lzr update apply --version v26.3.27
```

### Options

```
  -h, --help             help for apply
      --version string   xray-core release tag to install (overrides settings.update.xrayVersion)
```

### SEE ALSO

* [lzr update](lzr_update.md)	 - Manage xray-core updates

