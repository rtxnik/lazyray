//go:build !windows

package lifecycle

import (
	"os/exec"
	"testing"
	"time"
)

func TestSignalSupervisor_TerminatesLiveProcess(t *testing.T) {
	cmd := exec.Command("sleep", "60")
	if err := cmd.Start(); err != nil {
		t.Fatalf("start sleep: %v", err)
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	if err := SignalSupervisor(cmd.Process.Pid); err != nil {
		t.Fatalf("SignalSupervisor() error = %v", err)
	}
	select {
	case <-done: // exited due to SIGTERM
	case <-time.After(5 * time.Second):
		_ = cmd.Process.Kill()
		t.Fatal("process was not terminated by SignalSupervisor within 5s")
	}
}

func TestSignalSupervisor_NoSuchProcessErrors(t *testing.T) {
	// 1<<30 (~1.07e9) is above any real PID (pid_max); ESRCH expected.
	if err := SignalSupervisor(1 << 30); err == nil {
		t.Error("SignalSupervisor(nonexistent) = nil, want error")
	}
}
