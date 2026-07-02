## lzr start

Start the xray proxy supervisor in the background

### Synopsis

Start the xray-core supervisor as a background process using the default proxy profile from your settings.

This is the headless counterpart to launching the TUI: it spawns the resident supervisor, waits for it to take the lock, and prints the local SOCKS5/HTTP listen addresses. Use --proxy or --no-proxy to override whether the system proxy is enabled for this session. Running 'lzr start' when lazyray is already running is a no-op.

```
lzr start [flags]
```

### Examples

```
  # Start in the background using your default profile and settings
  lzr start

  # Start and force the system proxy on for this session
  lzr start --proxy

  # Start without touching system proxy settings
  lzr start --no-proxy
```

### Options

```
  -h, --help       help for start
      --no-proxy   leave the system proxy untouched for this session, overriding the profile store default
      --proxy      enable the system proxy for this session, overriding the profile store default
```

### SEE ALSO

* [lzr](lzr.md)	 - Terminal UI for managing Xray-core proxy configurations

