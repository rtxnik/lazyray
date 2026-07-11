## lzr config restore

Restore configuration from a backup archive

### Synopsis

Restore the profile store and the generated xray config from an archive made by 'lzr config backup'. Encrypted archives are detected automatically; the passphrase comes from --passphrase-file, the LAZYRAY_PASSPHRASE environment variable, or an interactive prompt. Plain tar.gz archives from older versions restore without a passphrase. Recognized members (servers.yaml, lazyray.yaml, config.json) overwrite the current files.

```
lzr config restore <file> [flags]
```

### Examples

```
  lzr config restore ~/lazyray-backup.tar.gz.enc
  lzr config restore ./lazyray-backup-20260101-120000.tar.gz
```

### Options

```
  -h, --help                     help for restore
      --passphrase-file string   Read the decryption passphrase from the first line of this file
```

### SEE ALSO

* [lzr config](lzr_config.md)	 - Manage configuration

