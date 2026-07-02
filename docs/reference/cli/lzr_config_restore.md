## lzr config restore

Restore configuration from tar.gz archive

### Synopsis

Restore the profile store and the generated xray config from a tar.gz archive made by 'lzr config backup'. Recognized members (servers.yaml, lazyray.yaml, config.json) overwrite the current files. Use it to recover a setup or replicate it on another machine.

```
lzr config restore <file> [flags]
```

### Examples

```
  lzr config restore ~/lazyray-backup.tar.gz
  lzr config restore ./lazyray-backup-20260101-120000.tar.gz
```

### Options

```
  -h, --help   help for restore
```

### SEE ALSO

* [lzr config](lzr_config.md)	 - Manage configuration

