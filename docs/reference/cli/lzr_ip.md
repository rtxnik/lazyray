## lzr ip

Show proxy and direct IP addresses

### Synopsis

Show your direct public IP and the IP seen through the proxy.

ip fetches your direct exit IP and the exit IP observed through the active proxy
server, so you can confirm at a glance that traffic is actually leaving via the
proxy and not the local network. Each lookup is reported independently; a failed
lookup is shown inline rather than aborting the command.

```
lzr ip [flags]
```

### Examples

```
  # Compare direct and proxy exit IPs
  lzr ip

  # Machine-readable output for scripts
  lzr ip --json
```

### Options

```
  -h, --help   help for ip
      --json   Print the direct and proxy IPs as JSON instead of human-readable text
```

### SEE ALSO

* [lzr](lzr.md)	 - Terminal UI for managing Xray-core proxy configurations

