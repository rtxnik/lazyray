## lzr import

Import proxy configuration URL or subscription

### Synopsis

Import a single proxy URL (VLESS, VMess, Trojan, Shadowsocks, Hysteria2) into a new proxy profile, or import all profiles from a subscription URL with --sub. The first imported profile becomes the default.

```
lzr import [url] [flags]
```

### Examples

```
  lzr import vless://uuid@host:port?params#name
  lzr import vmess://base64...
  lzr import trojan://pass@host:port?params#name
  lzr import --sub https://example.com/sub
```

### Options

```
      --allow-routing            Honor routing/DNS overrides carried by an encrypted import (validated against an allowlist)
      --decrypt                  Decrypt an encrypted export (passphrase from --passphrase-file, LAZYRAY_PASSPHRASE, or prompt)
  -f, --force                    Import even if UUID already exists
  -h, --help                     help for import
  -n, --name string              Profile name (default: from URL fragment)
      --passphrase-file string   Read the decryption passphrase from the first line of this file
      --sub string               Import from subscription URL ('-' reads the URL from stdin)
```

### SEE ALSO

* [lzr](lzr.md)	 - Terminal UI for managing Xray-core proxy configurations

