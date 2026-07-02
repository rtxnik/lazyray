## lzr pac generate

Generate PAC file to stdout or file

### Synopsis

Generate a PAC (Proxy Auto-Configuration) file for the default proxy profile and print it to stdout, or write it to a file with --output. Point a browser's auto-config URL at the resulting file to route matching traffic through lazyray.

```
lzr pac generate [flags]
```

### Examples

```
  lzr pac generate
  lzr pac generate --output proxy.pac
```

### Options

```
  -h, --help            help for generate
  -o, --output string   write the PAC file to this path instead of stdout
```

### SEE ALSO

* [lzr pac](lzr_pac.md)	 - PAC file generation and serving

