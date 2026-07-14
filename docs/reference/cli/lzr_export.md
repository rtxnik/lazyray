## lzr export

Export profile as proxy URL

### Synopsis

Export a proxy profile as its protocol URL (VLESS, VMess, or Trojan). Without arguments, exports the default profile.

```
lzr export [name] [flags]
```

### Examples

```
  lzr export
  lzr export home
  lzr export --all
  lzr export --qr
  lzr export --encrypt
```

### Options

```
      --all                      Export all profiles
      --encrypt                  Export all profiles encrypted (passphrase from --passphrase-file, LAZYRAY_PASSPHRASE, or prompt)
  -h, --help                     help for export
      --passphrase-file string   Read the encryption passphrase from the first line of this file
      --qr                       Display proxy URL as QR code in terminal
```

### SEE ALSO

* [lzr](lzr.md)	 - Terminal UI for managing Xray-core proxy configurations

