// internal/lifecycle/proc_windows.go
//go:build windows

package lifecycle

import (
	"os"
	"syscall"
	"time"
)

var (
	modkernel32  = syscall.NewLazyDLL("kernel32.dll")
	procOpenProc = modkernel32.NewProc("OpenProcess")
)

const processQueryLimitedInfo = 0x1000

// isProcessAlive checks PID existence via OpenProcess (also used by lock_windows).
func isProcessAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	h, _, _ := procOpenProc.Call(uintptr(processQueryLimitedInfo), 0, uintptr(pid))
	if h == 0 {
		return false
	}
	_ = syscall.CloseHandle(syscall.Handle(h))
	return true
}

// GracefulKill on Windows is a best-effort hard kill (no SIGTERM). Per the
// design, Windows graceful-stop parity is deferred to a seed.
func GracefulKill(pid int, _ time.Duration) error {
	if !isProcessAlive(pid) {
		return nil
	}
	p, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	return p.Kill()
}
