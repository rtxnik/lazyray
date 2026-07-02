## lzr pac serve

Serve PAC file over HTTP

### Synopsis

Start a local HTTP server that serves the PAC file for the default proxy profile. Point a browser's automatic proxy-configuration URL at it, or pass --system to also set that PAC URL as the system proxy (auto-rolled-back when the server stops).

```
lzr pac serve [flags]
```

### Examples

```
  lzr pac serve
  lzr pac serve --port 8080
  lzr pac serve --system
```

### Options

```
  -h, --help       help for serve
  -p, --port int   Port to serve PAC file on (default 10810)
      --system     Set PAC URL as system proxy (auto-rollback on stop)
```

### SEE ALSO

* [lzr pac](lzr_pac.md)	 - PAC file generation and serving

