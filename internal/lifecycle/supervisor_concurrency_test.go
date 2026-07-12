//go:build !windows

package lifecycle

import (
	"context"
	"os/exec"
	"sync/atomic"
	"syscall"
	"testing"
	"time"
)

// startSleeper starts a real child in its own process group as an xray stand-in.
func startSleeper(t *testing.T, secs string) *exec.Cmd {
	t.Helper()
	c := exec.Command("sleep", secs)
	c.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := c.Start(); err != nil {
		t.Fatalf("start sleeper: %v", err)
	}
	return c
}

func TestSuperviseXrayReal_CancelTerminatesChild(t *testing.T) {
	withTempData(t)
	c := startSleeper(t, "30")
	pid := c.Process.Pid
	s := &Supervisor{Settings: testSettings(), cmd: c}
	s.Settings.Xray.AutoRestart = false

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	st := &State{XrayPID: pid}
	go func() { _ = s.superviseXrayReal(ctx, st); close(done) }()

	cancel()
	select {
	case <-done:
	case <-time.After(8 * time.Second):
		t.Fatal("superviseXrayReal did not return after cancel")
	}
	time.Sleep(100 * time.Millisecond)
	if syscall.Kill(pid, 0) == nil {
		t.Error("child still alive after cancel teardown")
	}
}

func TestSuperviseXrayReal_AutoRestartsOnUnexpectedExit(t *testing.T) {
	withTempData(t)
	first := startSleeper(t, "0.2") // exits quickly → triggers a restart
	s := &Supervisor{Settings: testSettings(), cmd: first}
	s.Settings.Xray.AutoRestart = true

	var restarts atomic.Int64
	s.startXray = func() (int, error) {
		restarts.Add(1)
		c := startSleeper(t, "30")
		s.cmd = c
		return c.Process.Pid, nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	st := &State{XrayPID: first.Process.Pid}
	go func() { _ = s.superviseXrayReal(ctx, st); close(done) }()

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) && restarts.Load() < 1 {
		time.Sleep(20 * time.Millisecond)
	}
	cancel()
	<-done
	if restarts.Load() < 1 {
		t.Errorf("expected >=1 restart, got %d", restarts.Load())
	}
}
