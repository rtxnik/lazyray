//go:build windows

package core

import (
	"os"
	"syscall"

	"github.com/rtxnik/lazyray/internal/procutil"
)

// isProcessAlive reports whether a process with the given PID is running.
// Single source of truth: procutil.Alive (OpenProcess).
func isProcessAlive(pid int) bool { return procutil.Alive(pid) }

// gracefulKill terminates a child process. Windows has no SIGTERM, so this is a
// direct Kill(); Wait() reaps the child so its handle is freed (parity with the
// unix child path, which reaps in a goroutine).
func gracefulKill(proc *os.Process) error {
	err := proc.Kill()
	_, _ = proc.Wait()
	return err
}

// detachedProcAttr returns SysProcAttr for detaching the child process on Windows.
func detachedProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{CreationFlags: 0x00000008} // CREATE_NO_WINDOW
}
