//go:build !windows

package cmd

import "syscall"

// forceKillSupervisor hard-kills an unresponsive supervisor. (Graceful SIGTERM
// now lives in lifecycle.SignalSupervisor; hard-kill escalation stays CLI-only.)
func forceKillSupervisor(pid int) error { return syscall.Kill(pid, syscall.SIGKILL) }
