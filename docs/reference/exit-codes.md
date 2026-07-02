# Exit Codes

`lzr` returns the following process exit codes. Scripts should treat any non-zero
code as failure; only `0`, `1`, and `5` are currently emitted.

| Code | Constant | Meaning | When |
|------|----------|---------|------|
| `0` | `ExitOK` | Success | the command completed successfully |
| `1` | `ExitGeneric` | Generic failure | any error without a more specific code, including `lzr doctor` check failures and warnings promoted by `--strict` |
| `3` | `ExitNotRunning` | Reserved | defined for scripting forward-compatibility; **not currently emitted** |
| `4` | `ExitConflict` | Reserved | defined for scripting forward-compatibility; **not currently emitted** |
| `5` | `ExitConfig` | Configuration / startup error | `lzr start` / `lzr restart` could not load settings or spawn the background supervisor |

The mapping is centralized: command errors flow through a single funnel
(`cmd.Execute`), and an error carrying a specific code (`ExitError`) sets that
code; everything else maps to `ExitGeneric` (1).
