## lzr config backup

Backup configuration to tar.gz archive

### Synopsis

Write a tar.gz archive of the profile store and the generated xray config (servers.yaml, lazyray.yaml, config.json). Without a path, the archive lands in the backup directory with a timestamped name and old backups are rotated. Use it before risky edits or to move your setup to another machine.

```
lzr config backup [file] [flags]
```

### Examples

```
  lzr config backup
  lzr config backup ~/lazyray-backup.tar.gz
```

### Options

```
  -h, --help   help for backup
```

### SEE ALSO

* [lzr config](lzr_config.md)	 - Manage configuration

