//go:build windows

package core

import (
	"os"
	"syscall"
)

var (
	modkernel32  = syscall.NewLazyDLL("kernel32.dll")
	procOpenProc = modkernel32.NewProc("OpenProcess")
)

const processQueryLimitedInfo = 0x1000

// isProcessAlive checks whether a process with the given PID is running
// using the Windows OpenProcess API.
func isProcessAlive(pid int) bool {
	h, _, err := procOpenProc.Call(
		uintptr(processQueryLimitedInfo),
		0,
		uintptr(pid),
	)
	if h == 0 {
		_ = err
		return false
	}
	_ = syscall.CloseHandle(syscall.Handle(h))
	return true
}

// gracefulKill terminates the process. Windows does not support SIGTERM,
// so we fall back to Kill() directly.
func gracefulKill(proc *os.Process) error {
	return proc.Kill()
}

// detachedProcAttr returns SysProcAttr for detaching the child process on Windows.
func detachedProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{
		CreationFlags: 0x00000008, // CREATE_NO_WINDOW
	}
}
