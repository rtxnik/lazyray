## lzr config switch

Switch active profile

### Synopsis

Make the named proxy profile the default, so subsequent connects and exports use it. Run 'lzr config list' first to see the available profile names.

```
lzr config switch <name> [flags]
```

### Examples

```
  lzr config switch home
  lzr config switch "work vpn"
```

### Options

```
  -h, --help   help for switch
```

### SEE ALSO

* [lzr config](lzr_config.md)	 - Manage configuration

