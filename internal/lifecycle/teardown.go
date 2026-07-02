// internal/lifecycle/teardown.go
package lifecycle

import (
	"time"

	"github.com/rtxnik/lazyray/internal/core"
	"github.com/rtxnik/lazyray/internal/platform"
)

// defaultKillTimeout bounds the SIGTERM grace window before SIGKILL.
const defaultKillTimeout = 5 * time.Second

// Teardown performs the ordered shutdown for a session: revert routing first
// (so new traffic stops flowing to a dying proxy), then kill the xray group,
// then remove the state file. kill is injected for testability.
func Teardown(s *State, sp platform.SystemProxy, kill func(pid int) error) error {
	if s == nil {
		return nil
	}
	if s.Routing.SystemProxy || s.Routing.PAC {
		_ = RevertRouting(sp, s.Routing)
	}
	if s.XrayPID > 0 && kill != nil {
		_ = kill(s.XrayPID)
	}
	return RemoveState()
}

// DefaultKill graceful-kills the process group with the standard timeout.
func DefaultKill(pid int) error { return GracefulKill(pid, defaultKillTimeout) }

// guardedKill graceful-kills pid only if it is still lazyray's own xray, so the
// self-heal path never signals a reused or foreign PID.
func guardedKill(pid int) error {
	if !core.IsOurXray(pid) {
		return nil
	}
	return GracefulKill(pid, defaultKillTimeout)
}

// Reconcile is the crash self-heal: if a state file exists but no live
// supervisor holds the lock, tear down the abandoned session (revert dangling
// proxy, kill any orphaned xray, remove state). No-op when a supervisor is alive.
func Reconcile(sp platform.SystemProxy) error {
	s, err := ReadState()
	if err != nil || s == nil {
		return err
	}
	if SupervisorAlive() {
		return nil
	}
	return Teardown(s, sp, guardedKill)
}
