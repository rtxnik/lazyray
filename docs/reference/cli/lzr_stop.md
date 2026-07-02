## lzr stop

Stop the xray proxy and revert system routing

### Synopsis

Stop the running xray-core supervisor and revert any system proxy changes it made.

This asks the supervisor to shut down gracefully so it can tear down the SSH tunnel and restore your OS-level system proxy settings; if it does not respond it is hard-killed and the session is self-healed. Running 'lzr stop' when lazyray is not running is a no-op.

```
lzr stop [flags]
```

### Examples

```
  # Stop the supervisor and restore system routing
  lzr stop
```

### Options

```
  -h, --help   help for stop
```

### SEE ALSO

* [lzr](lzr.md)	 - Terminal UI for managing Xray-core proxy configurations

