//go:build windows

package procutil

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

// Alive reports whether pid exists, via the Windows OpenProcess API.
func Alive(pid int) bool {
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

// GracefulKill on Windows is a best-effort hard kill (no SIGTERM). Windows
// graceful-stop parity is deferred to a seed.
func GracefulKill(pid int, _ time.Duration) error {
	if !Alive(pid) {
		return nil
	}
	p, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	defer func() { _ = p.Release() }() // free the process handle after Kill
	return p.Kill()
}
