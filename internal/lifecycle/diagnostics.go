package lifecycle

import "github.com/rtxnik/lazyray/internal/core"

// IsOurXray reports whether pid belongs to lazyray's managed xray process. Thin
// exported wrapper delegating to core (the single source of truth) for
// diagnostics use by internal/doctor.
func IsOurXray(pid int) bool {
	return core.IsOurXray(pid)
}
