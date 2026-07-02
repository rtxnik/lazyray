//go:build windows

package core

// On Windows there is no cheap command-line introspection, so identity is
// best-effort (liveness only), matching the deferred Windows graceful-stop
// parity. isProcessAlive lives in process_windows.go.
func isOurXray(pid int) bool   { return pid > 0 && isProcessAlive(pid) }
func isOurTunnel(pid int) bool { return pid > 0 && isProcessAlive(pid) }
