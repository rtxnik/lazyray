// internal/lifecycle/proc.go
package lifecycle

import (
	"time"

	"github.com/rtxnik/lazyray/internal/procutil"
)

// GracefulKill terminates pid's process group gracefully (SIGTERM, poll, then
// SIGKILL; safe for non-children). Thin wrapper over the shared procutil
// primitive, kept so lifecycle's kill API and its DefaultKill/guardedKill
// callers are undisturbed by the procutil relocation.
func GracefulKill(pid int, timeout time.Duration) error {
	return procutil.GracefulKill(pid, timeout)
}
