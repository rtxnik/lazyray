// internal/lifecycle/proc_windows.go
//go:build windows

package lifecycle

import "syscall"

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
