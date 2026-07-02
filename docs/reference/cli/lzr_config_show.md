## lzr config show

Show current xray configuration

### Synopsis

Print the generated xray config (the xray-core JSON that lazyray builds from your active proxy profile) to stdout. Use it to inspect exactly what xray-core will run, or to copy the config elsewhere.

```
lzr config show [flags]
```

### Examples

```
  lzr config show
  lzr config show | jq .
```

### Options

```
  -h, --help   help for show
```

### SEE ALSO

* [lzr config](lzr_config.md)	 - Manage configuration

