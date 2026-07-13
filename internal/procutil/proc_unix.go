//go:build !windows

// Package procutil is the single home for lazyray's process-lifecycle
// primitives (liveness, graceful group termination, TCP reachability, atomic
// pid-file writes), shared by internal/core and internal/lifecycle. It is a
// leaf: it imports only stdlib and internal/fsutil.
package procutil

import (
	"fmt"
	"syscall"
	"time"
)

const pollInterval = 100 * time.Millisecond

// Seams (overridable in tests), following the repo's existing mock-var pattern.
var (
	nowFn   = time.Now
	sleepFn = time.Sleep

	// killSignal sends sig to pid. Callers pass a negative pid to target the
	// whole process group.
	killSignal = func(pid int, sig syscall.Signal) error {
		return syscall.Kill(pid, sig)
	}

	// processAlive reports whether pid exists (signal-0 probe). It probes with
	// syscall.Kill directly (NOT killSignal) so tests that record delivered
	// signals via the killSignal seam do not also capture liveness probes.
	processAlive = func(pid int) bool {
		if pid <= 0 {
			return false
		}
		err := syscall.Kill(pid, 0)
		return err == nil || err == syscall.EPERM
	}
)

// Alive reports whether a process with the given PID exists. pid<=0 is never
// alive; EPERM (process exists but owned by another user) counts as alive.
func Alive(pid int) bool { return processAlive(pid) }

// GracefulKill terminates the process group led by pid: SIGTERM, poll up to
// timeout, then SIGKILL, confirming death. No-op if already dead. Works whether
// or not the caller is the parent (it polls liveness, never proc.Wait()).
func GracefulKill(pid int, timeout time.Duration) error {
	if pid <= 0 || !processAlive(pid) {
		return nil
	}
	_ = killSignal(-pid, syscall.SIGTERM)
	if waitDeath(pid, timeout) {
		return nil
	}
	_ = killSignal(-pid, syscall.SIGKILL)
	if waitDeath(pid, 2*time.Second) {
		return nil
	}
	return fmt.Errorf("process group %d still alive after SIGKILL", pid)
}

// waitDeath polls until pid is gone or the timeout elapses.
func waitDeath(pid int, timeout time.Duration) bool {
	deadline := nowFn().Add(timeout)
	for nowFn().Before(deadline) {
		if !processAlive(pid) {
			return true
		}
		sleepFn(pollInterval)
	}
	return !processAlive(pid)
}
