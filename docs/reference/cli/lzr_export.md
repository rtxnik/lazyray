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
  lzr export --encrypt "passphrase"
```

### Options

```
      --all              Export all profiles
      --encrypt string   Export all profiles encrypted with password
  -h, --help             help for export
      --qr               Display proxy URL as QR code in terminal
```

### SEE ALSO

* [lzr](lzr.md)	 - Terminal UI for managing Xray-core proxy configurations

