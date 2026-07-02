//go:build !windows

package lifecycle

import "syscall"

// SignalSupervisor asks the supervisor to shut down gracefully (SIGTERM on
// Unix). Both shells and the app service route their graceful-stop signal
// through this one primitive.
func SignalSupervisor(pid int) error { return syscall.Kill(pid, syscall.SIGTERM) }
