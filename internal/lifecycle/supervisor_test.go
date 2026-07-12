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
	s.superviseXray = func(ctx context.Context, st *State) error {
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

// setLoopSeams overrides the supervise-loop timing seams for a test and
// restores them on cleanup. Never call from a t.Parallel test.
func setLoopSeams(t *testing.T, backoff, threshold time.Duration, now func() time.Time) {
	t.Helper()
	ob, ot, on := restartBackoff, healthyRunThreshold, timeNow
	restartBackoff, healthyRunThreshold, timeNow = backoff, threshold, now
	t.Cleanup(func() { restartBackoff, healthyRunThreshold, timeNow = ob, ot, on })
}

// scriptedClock returns successive instants from a script; the last value
// repeats if the script is exhausted (defensive against an unexpected extra
// call). Named scriptedClock (not fakeClock) to avoid colliding with the
// unrelated fakeClock type already declared in proc_unix_test.go.
func scriptedClock(script []time.Time) func() time.Time {
	i := 0
	return func() time.Time {
		v := script[i]
		if i < len(script)-1 {
			i++
		}
		return v
	}
}

// buildClock produces a timeNow script for superviseXrayReal from the intended
// uptime of each incarnation. The loop calls timeNow once for the initial
// startedAt, then per incarnation once to read uptime and once (on restart) to
// reset startedAt; buildClock spaces the instants so incarnation k's
// (uptime read - startedAt) == uptimes[k].
func buildClock(uptimes []time.Duration) func() time.Time {
	base := time.Unix(1_700_000_000, 0)
	script := []time.Time{base} // initial startedAt
	cur := base
	for _, up := range uptimes {
		cur = cur.Add(up)
		script = append(script, cur, cur) // uptime read, then the next startedAt (== now)
	}
	return scriptedClock(script)
}

// L2-02: after auto-restart, both the in-memory State (via Teardown's kill) and
// the on-disk State target the CURRENT xray pid, never the pre-restart pid.
func TestSupervisor_AutoRestart_SyncsPidForTeardown(t *testing.T) {
	withTempData(t)
	setLoopSeams(t, time.Millisecond, time.Hour, time.Now) // all rapid (uptime<1h), tiny backoff
	s := &Supervisor{Owner: OwnerDaemon, Settings: testSettings(), sysproxy: &fakeProxy{}}
	s.Settings.Xray.AutoRestart = true
	s.Settings.Notifications.Enabled = false

	started := []int{}
	s.startXray = func() (int, error) {
		pid := 4242 + len(started)*1111
		started = append(started, pid)
		return pid, nil
	}
	var killedPid, diskPidAtKill int
	s.killXray = func(pid int) error {
		killedPid = pid
		if st, _ := ReadState(); st != nil {
			diskPidAtKill = st.XrayPID
		}
		return nil
	}
	// superviseXray defaults to the real impl (s.cmd stays nil, so each
	// incarnation's exit is instant), driving restarts to the give-up limit.
	if err := s.Run(context.Background()); err != nil {
		t.Fatalf("Run() = %v", err)
	}
	if len(started) < 2 {
		t.Fatalf("expected auto-restarts, got %d starts", len(started))
	}
	last := started[len(started)-1]
	if killedPid != last {
		t.Errorf("Teardown killed pid %d, want current %d (in-memory State not synced)", killedPid, last)
	}
	if diskPidAtKill != last {
		t.Errorf("disk State pid at kill = %d, want %d", diskPidAtKill, last)
	}
}

// L2-03: a healthy run resets the rapid-crash streak AND does not consume a
// retry slot, so a following rapid streak gets the full maxRetries budget.
func TestSupervisor_AutoRestart_ResetsRetriesAfterHealthyRun(t *testing.T) {
	withTempData(t)
	ms := time.Millisecond
	// rapid, rapid, HEALTHY, rapid, rapid, rapid, (give up on the next exit)
	uptimes := []time.Duration{ms, ms, 2 * time.Second, ms, ms, ms, ms}
	setLoopSeams(t, ms, time.Second, buildClock(uptimes))
	s := &Supervisor{Owner: OwnerDaemon, Settings: testSettings(), sysproxy: &fakeProxy{}}
	s.Settings.Xray.AutoRestart = true
	s.Settings.Notifications.Enabled = false

	starts := 0
	s.startXray = func() (int, error) { starts++; return 4000 + starts, nil }
	s.killXray = func(int) error { return nil }

	if err := s.Run(context.Background()); err != nil {
		t.Fatalf("Run() = %v", err)
	}
	// initial + 6 restarts: rapid×2 (retries 1,2) → HEALTHY (reset, free) →
	// rapid×3 (retries 1,2,3) → give up. Without the reset+free-restart fix the
	// loop gives up after only 4 starts.
	if starts != 7 {
		t.Errorf("startXray calls = %d, want 7 (a healthy run must grant a full fresh budget)", starts)
	}
}

// L2-03: rapid respawns are paced by restartBackoff (real timer).
func TestSupervisor_AutoRestart_PacesRapidRespawns(t *testing.T) {
	withTempData(t)
	backoff := 25 * time.Millisecond
	setLoopSeams(t, backoff, time.Hour, time.Now) // all rapid
	s := &Supervisor{Owner: OwnerDaemon, Settings: testSettings(), sysproxy: &fakeProxy{}}
	s.Settings.Xray.AutoRestart = true
	s.Settings.Notifications.Enabled = false
	s.startXray = func() (int, error) { return 4242, nil }
	s.killXray = func(int) error { return nil }

	start := time.Now()
	if err := s.Run(context.Background()); err != nil {
		t.Fatalf("Run() = %v", err)
	}
	// 3 rapid restarts before give-up, each preceded by one backoff.
	if elapsed := time.Since(start); elapsed < 3*backoff {
		t.Errorf("crash-loop elapsed %v, want ≥ %v (respawns must be paced)", elapsed, 3*backoff)
	}
}

// L2-03: a shutdown during a rapid-crash backoff returns promptly.
func TestSupervisor_Shutdown_InterruptsBackoff(t *testing.T) {
	withTempData(t)
	setLoopSeams(t, 10*time.Second, time.Hour, time.Now) // long backoff, all rapid
	s := &Supervisor{Owner: OwnerDaemon, Settings: testSettings(), sysproxy: &fakeProxy{}}
	s.Settings.Xray.AutoRestart = true
	s.Settings.Notifications.Enabled = false
	s.startXray = func() (int, error) { return 4242, nil }
	s.killXray = func(int) error { return nil }

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { _ = s.Run(ctx); close(done) }()
	time.Sleep(50 * time.Millisecond) // let the loop reach its first backoff
	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not return promptly after cancel during backoff")
	}
}

// XREV #2: a cancelled context preempts a respawn even when the exited branch
// wins the select race.
func TestSupervisor_ExitedAfterCancel_DoesNotRespawn(t *testing.T) {
	withTempData(t)
	setLoopSeams(t, time.Millisecond, time.Hour, time.Now)
	s := &Supervisor{Owner: OwnerDaemon, Settings: testSettings(), sysproxy: &fakeProxy{}}
	s.Settings.Xray.AutoRestart = true
	s.Settings.Notifications.Enabled = false
	startCount := 0
	s.startXray = func() (int, error) { startCount++; return 4000 + startCount, nil }
	s.killXray = func(int) error { return nil }

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // pre-cancelled: the exited branch must observe ctx.Err()!=nil
	if err := s.Run(ctx); err != nil {
		t.Fatalf("Run() = %v", err)
	}
	if startCount != 1 {
		t.Errorf("startXray called %d times, want 1 (cancelled ctx must preempt respawn)", startCount)
	}
}

// Plan-review (codex, adjudicated LOW): a persistence failure on restart must NOT
// drop the working proxy. The in-memory State stays the load-bearing source of
// truth, so Teardown still kills the CURRENT pid even when every on-disk write
// failed. The complementary "a resulting stale on-disk pid is never wrong-killed"
// guarantee is already covered by TestReconcile_SkipsKillForForeignPID (guardedKill
// / IsOurXray), so this test asserts only the in-memory invariant.
func TestSupervisor_AutoRestart_WriteStateFailure_KeepsRunningTeardownKillsCurrent(t *testing.T) {
	withTempData(t)
	setLoopSeams(t, time.Millisecond, time.Hour, time.Now) // all rapid, tiny backoff
	ow := writeState
	writeState = func(*State) error { return errors.New("simulated persistence failure") }
	t.Cleanup(func() { writeState = ow })

	s := &Supervisor{Owner: OwnerDaemon, Settings: testSettings(), sysproxy: &fakeProxy{}}
	s.Settings.Xray.AutoRestart = true
	s.Settings.Notifications.Enabled = false

	started := []int{}
	s.startXray = func() (int, error) {
		pid := 5000 + len(started)
		started = append(started, pid)
		return pid, nil
	}
	var killedPid int
	s.killXray = func(pid int) error { killedPid = pid; return nil }

	// The initial Run write (supervisor.go:87) uses WriteState directly and
	// succeeds under withTempData; only the restart-tail writeState seam fails.
	if err := s.Run(context.Background()); err != nil {
		t.Fatalf("Run() = %v", err)
	}
	if len(started) < 2 {
		t.Fatalf("expected auto-restarts despite persistence failure, got %d starts", len(started))
	}
	last := started[len(started)-1]
	if killedPid != last {
		t.Errorf("Teardown killed pid %d, want current %d (in-memory State must stay load-bearing when WriteState fails)", killedPid, last)
	}
}
