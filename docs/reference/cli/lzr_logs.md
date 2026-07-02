## lzr logs

Show xray log output

### Synopsis

Show the tail of the xray-core log files.

logs prints the most recent lines of the xray-core access log, or the error log
with --error. Use it to see what the engine recorded for the active proxy
connection when a session misbehaves; pair it with 'lzr doctor' for a full
diagnosis. If no log file exists yet, logs says so rather than failing.

```
lzr logs [flags]
```

### Examples

```
  # Last 50 lines of the access log
  lzr logs

  # Last 200 lines of the error log
  lzr logs --error --lines 200
```

### Options

```
  -e, --error       Show the xray-core error log instead of the access log
  -h, --help        help for logs
  -n, --lines int   Number of trailing log lines to show (0 shows all) (default 50)
```

### SEE ALSO

* [lzr](lzr.md)	 - Terminal UI for managing Xray-core proxy configurations

