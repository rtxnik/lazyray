## lzr config delete

Delete a profile

### Synopsis

Remove the named proxy profile from the profile store. If you delete the default profile, the first remaining profile becomes the new default. Run 'lzr config list' to confirm the exact name.

```
lzr config delete <name> [flags]
```

### Examples

```
  lzr config delete old-server
  lzr config delete "work vpn"
```

### Options

```
  -h, --help   help for delete
```

### SEE ALSO

* [lzr config](lzr_config.md)	 - Manage configuration

