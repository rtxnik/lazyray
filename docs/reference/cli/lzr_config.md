## lzr config

Manage configuration

### Synopsis

Manage the profile store and the generated xray config. The "config" group covers two distinct things: the profile store (the YAML at servers.yaml/lazyray.yaml that holds your proxy profiles and settings) and the generated xray config (the xray-core JSON that 'config show' and 'config edit' open). Use the subcommands to list, switch, inspect, duplicate, delete, back up, and restore profiles.

### Options

```
  -h, --help   help for config
```

### SEE ALSO

* [lzr](lzr.md)	 - Terminal UI for managing Xray-core proxy configurations
* [lzr config backup](lzr_config_backup.md)	 - Backup configuration to tar.gz archive
* [lzr config delete](lzr_config_delete.md)	 - Delete a profile
* [lzr config duplicate](lzr_config_duplicate.md)	 - Duplicate a profile
* [lzr config edit](lzr_config_edit.md)	 - Open configuration in editor
* [lzr config list](lzr_config_list.md)	 - List all profiles
* [lzr config restore](lzr_config_restore.md)	 - Restore configuration from tar.gz archive
* [lzr config show](lzr_config_show.md)	 - Show current xray configuration
* [lzr config switch](lzr_config_switch.md)	 - Switch active profile

