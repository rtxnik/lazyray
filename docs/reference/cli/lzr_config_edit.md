## lzr config edit

Open configuration in editor

### Synopsis

Open the generated xray config (the xray-core JSON) in your editor, using $EDITOR (falling back to vi). Use it for manual tweaks; lazyray rewrites this file whenever you import or switch a proxy profile, so durable changes belong in the profile store.

```
lzr config edit [flags]
```

### Examples

```
  lzr config edit
  EDITOR=nano lzr config edit
```

### Options

```
  -h, --help   help for edit
```

### SEE ALSO

* [lzr config](lzr_config.md)	 - Manage configuration

