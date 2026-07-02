//go:build !windows

package core

import (
	"os"
	"syscall"
	"time"
)

// gracefulKill sends SIGTERM to the process and waits up to 5 seconds
// before falling back to SIGKILL.
func gracefulKill(proc *os.Process) error {
	if err := proc.Signal(syscall.SIGTERM); err != nil {
		// Process may have already exited
		return proc.Kill()
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

// isProcessAlive checks whether a process with the given PID is running.
func isProcessAlive(pid int) bool {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	err = proc.Signal(syscall.Signal(0))
	return err == nil
}

// detachedProcAttr returns SysProcAttr for detaching the child process.
func detachedProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{
		Setsid: true,
	}
}
