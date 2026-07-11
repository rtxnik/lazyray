## lzr config backup

Backup configuration to an encrypted archive

### Synopsis

Write an archive of the profile store and the generated xray config (servers.yaml, lazyray.yaml, config.json). Archives are encrypted by default because they bundle proxy credentials; the passphrase comes from --passphrase-file, the LAZYRAY_PASSPHRASE environment variable, or an interactive prompt. Pass --no-encrypt for a plaintext tar.gz. Without a path, the archive lands in the backup directory with a timestamped name and old backups are rotated.

```
lzr config backup [file] [flags]
```

### Examples

```
  lzr config backup
  lzr config backup ~/lazyray-backup.tar.gz.enc
  lzr config backup --no-encrypt ~/lazyray-backup.tar.gz
  LAZYRAY_PASSPHRASE=secret lzr config backup
```

### Options

```
  -h, --help                     help for backup
      --no-encrypt               Write a plaintext archive instead of an encrypted one
      --passphrase-file string   Read the encryption passphrase from the first line of this file
```

### SEE ALSO

* [lzr config](lzr_config.md)	 - Manage configuration

