## lzr restart

Restart the xray proxy supervisor

### Synopsis

Stop the running xray-core supervisor (if any) and start a fresh one using the current settings.

Use this after editing the profile store or switching the default proxy profile so the new configuration takes effect. The system proxy default for the new session comes from your settings, the same as a plain 'lzr start'.

```
lzr restart [flags]
```

### Examples

```
  # Apply new settings by restarting the supervisor
  lzr restart
```

### Options

```
  -h, --help   help for restart
```

### SEE ALSO

* [lzr](lzr.md)	 - Terminal UI for managing Xray-core proxy configurations

