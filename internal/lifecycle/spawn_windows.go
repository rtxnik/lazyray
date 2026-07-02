// internal/lifecycle/spawn_windows.go
//go:build windows

package lifecycle

import (
	"os"
	"os/exec"
	"syscall"

	"github.com/rtxnik/lazyray/internal/config"
)

// xrayProcAttr launches xray in a new process group (best-effort group control;
// Windows has no Setpgid, so CREATE_NEW_PROCESS_GROUP is the nearest analogue).
func xrayProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{CreationFlags: 0x00000200} // CREATE_NEW_PROCESS_GROUP
}

// terminateSignals are the signals that trigger an ordered supervisor teardown.
// Windows only delivers os.Interrupt and syscall.SIGTERM; SIGHUP does not exist.
func terminateSignals() []os.Signal {
	return []os.Signal{os.Interrupt, syscall.SIGTERM}
}

// terminateChild best-effort kills the child and waits for the reaper (Windows
// has no SIGTERM; graceful-stop parity is deferred to a seed).
func (s *Supervisor) terminateChild(_ int, done <-chan struct{}) {
	if s.cmd != nil && s.cmd.Process != nil {
		_ = s.cmd.Process.Kill()
	}
	<-done
}

// openSupervisorLog opens (append/create) the supervisor stderr log with 0600
// perms. It returns nil on any failure so a missing/unwritable log can never
// block the supervisor from starting.
func openSupervisorLog() *os.File {
	f, err := os.OpenFile(config.SupervisorLogPath(), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return nil
	}
	return f
}

// SpawnDetached re-execs `lzr __run` with CREATE_NO_WINDOW (best-effort detach).
// The child's stderr is redirected to the supervisor log; stdin/stdout stay closed.
func SpawnDetached(extraArgs []string) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	args := append([]string{"__run"}, extraArgs...)
	cmd := exec.Command(exe, args...)
	logFile := openSupervisorLog()
	cmd.Stdin, cmd.Stdout = nil, nil
	if logFile != nil {
		cmd.Stderr = logFile
	} else {
		cmd.Stderr = nil
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x00000008} // CREATE_NO_WINDOW
	if err := cmd.Start(); err != nil {
		if logFile != nil {
			_ = logFile.Close()
		}
		return err
	}
	_ = cmd.Process.Release()
	if logFile != nil {
		_ = logFile.Close()
	}
	return nil
}
