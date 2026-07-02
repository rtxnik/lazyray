## lzr test

Test connection to a profile's server

### Synopsis

Test reachability of a proxy profile's server endpoint with a TCP connection probe.
Uses the default profile when no name is given. Pass --all to probe every proxy profile
and rank them by latency (failed profiles sort last). This is a fast liveness check; for a
full connectivity probe through the system proxy use 'lzr health', and for environment
diagnostics use 'lzr doctor'.

```
lzr test [profile-name] [flags]
```

### Examples

```
  # Test the default proxy profile
  lzr test

  # Test a specific proxy profile by name
  lzr test work

  # Test every proxy profile and sort by latency
  lzr test --all
```

### Options

```
  -a, --all    Test all profiles and sort by latency
  -h, --help   help for test
```

### SEE ALSO

* [lzr](lzr.md)	 - Terminal UI for managing Xray-core proxy configurations

