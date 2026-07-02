// internal/lifecycle/supervisor_test.go
package lifecycle

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestSupervisor_Run_LockedReturnsErrLocked(t *testing.T) {
	withTempData(t)
	l, _ := AcquireLock()
	defer func() { _ = l.Release() }()

	s := &Supervisor{Owner: OwnerDaemon, Settings: testSettings(), sysproxy: &fakeProxy{}}
	err := s.Run(context.Background())
	if !errors.Is(err, ErrLocked) {
		t.Errorf("Run() = %v, want ErrLocked", err)
	}
}

func TestSupervisor_Run_AppliesRoutingThenTearsDownOnCancel(t *testing.T) {
	withTempData(t)
	fp := &fakeProxy{}
	s := &Supervisor{
		Owner:    OwnerDaemon,
		Settings: testSettings(),
		Proxy:    ProxyForceOn,
		sysproxy: fp,
	}
	// Seam: pretend xray started at PID 4242 and never exits on its own.
	s.startXray = func() (int, error) { return 4242, nil }
	s.superviseXray = func(ctx context.Context, pid int) error {
		<-ctx.Done() // block until canceled, like a live process
		return nil
	}
	// Seam: record kill instead of signalling a real group.
	killed := 0
	s.killXray = func(pid int) error { killed = pid; return nil }

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- s.Run(ctx) }()

	// Wait until state is written (routing applied).
	waitFor(t, func() bool { st, _ := ReadState(); return st != nil })
	if !contains(fp.calls, "enableHTTP") {
		t.Errorf("routing not applied: calls=%v", fp.calls)
	}

	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Run() = %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not return after cancel")
	}
	if killed != 4242 {
		t.Errorf("killXray pid = %d, want 4242", killed)
	}
	if !contains(fp.calls, "disable") {
		t.Error("routing not reverted on teardown")
	}
	if st, _ := ReadState(); st != nil {
		t.Error("state not removed after teardown")
	}
}

func waitFor(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("condition not met within timeout")
}
