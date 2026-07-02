//go:build windows

package lifecycle

import "os"

// SignalSupervisor asks the supervisor to stop. Windows has no SIGTERM, so this
// is a best-effort process kill (matches the prior per-shell helpers).
func SignalSupervisor(pid int) error {
	p, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	return p.Kill()
}
