## lzr stats

Show traffic consumption statistics

### Synopsis

Show traffic consumption recorded by lazyray — today's usage, the current month,
and the all-time total — as a human-readable report. Pass --json to emit the raw history
for scripting. Stats accrue while a proxy profile is connected; this command reads the
stored history and does not require an active connection.

```
lzr stats [flags]
```

### Examples

```
  # Show the traffic report
  lzr stats

  # Emit the raw history as JSON
  lzr stats --json
```

### Options

```
  -h, --help   help for stats
      --json   Output in JSON format
```

### SEE ALSO

* [lzr](lzr.md)	 - Terminal UI for managing Xray-core proxy configurations

