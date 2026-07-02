## lzr config duplicate

Duplicate a profile

### Synopsis

Copy the named proxy profile to a new "(copy)" profile that is not the default. Use it as a starting point when you want a variant of an existing proxy server without re-importing it. Run 'lzr config list' to confirm the source name.

```
lzr config duplicate <name> [flags]
```

### Examples

```
  lzr config duplicate home
  lzr config duplicate "work vpn"
```

### Options

```
  -h, --help   help for duplicate
```

### SEE ALSO

* [lzr config](lzr_config.md)	 - Manage configuration

