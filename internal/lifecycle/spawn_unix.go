// internal/lifecycle/spawn_unix.go
//go:build !windows

package lifecycle

import (
	"os"
	"os/exec"
	"syscall"
	"time"

	"github.com/rtxnik/lazyray/internal/config"
)

// xrayProcAttr launches xray in its own process group so GracefulKill(-pid)
// can reap the whole group (xray plus any grandchildren).
func xrayProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{Setpgid: true}
}

// terminateSignals are the signals that trigger an ordered supervisor teardown.
func terminateSignals() []os.Signal {
	return []os.Signal{syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP}
}

// killGrace bounds the SIGTERM window before SIGKILL during in-process teardown.
const killGrace = 5 * time.Second

// terminateChild signals the child's process group and waits for the sole
// reaper (done) to confirm exit, escalating to SIGKILL after killGrace. It never
// polls the pid, so it cannot race the reaper on a recycled pid.
func (s *Supervisor) terminateChild(pid int, done <-chan struct{}) {
	_ = syscall.Kill(-pid, syscall.SIGTERM)
	select {
	case <-done:
	case <-time.After(killGrace):
		_ = syscall.Kill(-pid, syscall.SIGKILL)
		<-done
	}
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

// SpawnDetached re-execs the current binary as `lzr __run` with the given args,
// fully detached (new session) so the supervisor outlives the caller. The
// child's stderr is redirected to the supervisor log; stdin/stdout stay closed.
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
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		if logFile != nil {
			_ = logFile.Close()
		}
		return err
	}
	_ = cmd.Process.Release()
	// The child has inherited the fd; the parent can drop its handle now.
	if logFile != nil {
		_ = logFile.Close()
	}
	return nil
}
