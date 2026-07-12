## lzr tunnel trust

Pin or re-pin a profile's SSH host key

### Synopsis

Capture the SSH host key of a profile's server, show its SHA256 fingerprint, and pin it into the profile after explicit confirmation.

Verify the fingerprint out-of-band before confirming (on the server: ssh-keygen -lf /etc/ssh/ssh_host_*.pub). With --fingerprint the command is non-interactive: only captured keys whose fingerprints match the provided values are pinned, and a value that matches nothing is an error. Re-running the command replaces a previous pin (shown as "old") after the same confirmation. Pinning via a dedicated known_hosts disables OpenSSH's automatic host-key rotation, so re-run this command when the server legitimately rotates its keys.

```
lzr tunnel trust <name> [flags]
```

### Examples

```
  lzr tunnel trust ru
  lzr tunnel trust ru --fingerprint SHA256:mVN1EX9nGiimZzXFqXTZHrpx5RCasCMEEyBGavrfBFo
```

### Options

```
      --fingerprint stringArray   pin only captured keys matching this SHA256 fingerprint (repeatable; enables non-interactive pinning)
  -h, --help                      help for trust
```

### SEE ALSO

* [lzr tunnel](lzr_tunnel.md)	 - Manage SSH tunnels to server panels

