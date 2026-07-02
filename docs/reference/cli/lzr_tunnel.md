## lzr tunnel

Manage SSH tunnels to server panels

### Synopsis

Open or inspect an SSH tunnel to a profile's management panel.

This is an SSH tunnel to the server's admin panel — separate from the proxy itself: it does not route your traffic through xray-core and does not change the active proxy profile. With no argument it lists SSH-capable profiles and their tunnel state; with a profile name it opens a persistent tunnel to that profile's panel. Tear tunnels down with 'lzr tunnel close'.

```
lzr tunnel [name] [flags]
```

### Examples

```
  lzr tunnel
  lzr tunnel ru
  lzr tunnel close
```

### Options

```
  -h, --help   help for tunnel
```

### SEE ALSO

* [lzr](lzr.md)	 - Terminal UI for managing Xray-core proxy configurations
* [lzr tunnel close](lzr_tunnel_close.md)	 - Close all SSH tunnels

