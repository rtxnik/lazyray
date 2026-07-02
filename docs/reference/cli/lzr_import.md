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
      --decrypt string   Decrypt encrypted export with password
  -f, --force            Import even if UUID already exists
  -h, --help             help for import
  -n, --name string      Profile name (default: from URL fragment)
      --sub string       Import from subscription URL
```

### SEE ALSO

* [lzr](lzr.md)	 - Terminal UI for managing Xray-core proxy configurations

