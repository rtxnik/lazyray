package core

// ScanXrayPID returns the PID of any running xray process, or 0 when none is
// found. It is DIAGNOSTICS ONLY: it matches xray by name (foreign or managed)
// and MUST NOT be used as a termination target. The authoritative lifecycle
// source of truth is internal/lifecycle (state.json + supervisor.lock).
func ScanXrayPID() int {
	return findXrayPID()
}

// IsProcessAlive reports whether a process with the given PID is currently
// running. Thin exported wrapper over the platform-specific liveness probe,
// for diagnostics use by internal/doctor.
func IsProcessAlive(pid int) bool {
	return isProcessAlive(pid)
}
