//go:build !windows

package core

import (
	"os"
	"syscall"
	"time"

	"github.com/rtxnik/lazyray/internal/procutil"
)

// gracefulKill sends SIGTERM to a CHILD process and waits up to 5 seconds
// before falling back to SIGKILL. Child-only: the caller is the parent and
// proc.Wait() reaps it (a just-exited unreaped child answers signal-0 as alive,
// so the non-child poll-based path lives in procutil.GracefulKill instead).
func gracefulKill(proc *os.Process) error {
	if err := proc.Signal(syscall.SIGTERM); err != nil {
		return proc.Kill() // already exited
	}
	done := make(chan error, 1)
	go func() {
		_, err := proc.Wait()
		done <- err
	}()
	select {
	case <-done:
		return nil
	case <-time.After(5 * time.Second):
		return proc.Kill()
	}
}

// isProcessAlive reports whether a process with the given PID is running.
// Single source of truth: procutil.Alive (pid>0 guard, EPERM→alive).
func isProcessAlive(pid int) bool { return procutil.Alive(pid) }

// detachedProcAttr returns SysProcAttr for detaching the child process.
func detachedProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{Setsid: true}
}
