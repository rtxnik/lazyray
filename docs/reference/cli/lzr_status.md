## lzr status

Show xray proxy status

### Synopsis

Show whether the xray-core supervisor is running and the health of its local listeners.

Reports the supervisor state and PID, uptime, the active proxy profile, the xray-core version, and the local SOCKS5/HTTP listen addresses with an up/down probe of each. Use --json for a machine-readable snapshot; for a full diagnostic sweep use 'lzr doctor' instead.

```
lzr status [flags]
```

### Examples

```
  # Human-readable status
  lzr status

  # Machine-readable snapshot
  lzr status --json
```

### Options

```
  -h, --help   help for status
      --json   output a machine-readable JSON status snapshot
```

### SEE ALSO

* [lzr](lzr.md)	 - Terminal UI for managing Xray-core proxy configurations

