## lzr speedtest

Run a download speed test through the proxy

### Synopsis

Measure download throughput through the active system proxy. Downloads data from a
test server for the given duration and reports the observed speed. Requires a running
connection — start one with 'lzr start' first. Use --duration to lengthen the sample
and --url to point at your own test file.

```
lzr speedtest [flags]
```

### Examples

```
  # Run the default 10-second speed test
  lzr speedtest

  # Sample for 15 seconds
  lzr speedtest --duration 15

  # Use a custom download URL
  lzr speedtest --url http://example.com/testfile
```

### Options

```
      --duration int   Test duration in seconds (default 10)
  -h, --help           help for speedtest
      --url string     Custom download test URL
```

### SEE ALSO

* [lzr](lzr.md)	 - Terminal UI for managing Xray-core proxy configurations

