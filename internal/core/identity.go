package core

// IsOurXray reports whether pid belongs to lazyray's managed xray process. This
// is the single source of truth for xray identity, shared by the supervisor
// self-heal path (internal/lifecycle, which delegates here) and the direct stop
// path in this package. Unix: the process command line must reference the
// managed xray binary. Windows: best-effort (liveness only).
func IsOurXray(pid int) bool { return isOurXray(pid) }

// IsOurTunnel reports whether pid looks like one of lazyray's SSH port-forward
// tunnels. lazyray always spawns tunnels as `ssh -L <fwd> … -N …`, so on unix an
// `ssh` invocation carrying both -N and -L is treated as ours; this guards the
// cross-session tunnel-kill paths against signalling a reused or foreign PID.
// Windows: best-effort (liveness only).
func IsOurTunnel(pid int) bool { return isOurTunnel(pid) }
