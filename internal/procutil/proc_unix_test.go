//go:build !windows

package procutil

import (
	"syscall"
	"testing"
	"time"
)

// fakeClock lets waitDeath advance deterministically without real sleeps.
type fakeClock struct{ t time.Time }

func (c *fakeClock) now() time.Time        { return c.t }
func (c *fakeClock) sleep(d time.Duration) { c.t = c.t.Add(d) }

func swapClock(t *testing.T) *fakeClock {
	t.Helper()
	c := &fakeClock{t: time.Unix(0, 0)}
	origNow, origSleep := nowFn, sleepFn
	nowFn, sleepFn = c.now, c.sleep
	t.Cleanup(func() { nowFn, sleepFn = origNow, origSleep })
	return c
}

func swapSignals(t *testing.T) *[]syscall.Signal {
	t.Helper()
	var sent []syscall.Signal
	orig := killSignal
	killSignal = func(pid int, sig syscall.Signal) error {
		sent = append(sent, sig)
		return nil
	}
	t.Cleanup(func() { killSignal = orig })
	return &sent
}

func TestGracefulKill_AlreadyDead_NoSignals(t *testing.T) {
	swapClock(t)
	sent := swapSignals(t)
	orig := processAlive
	processAlive = func(pid int) bool { return false }
	t.Cleanup(func() { processAlive = orig })

	if err := GracefulKill(123, time.Second); err != nil {
		t.Fatalf("GracefulKill() = %v", err)
	}
	if len(*sent) != 0 {
		t.Errorf("sent %v signals to a dead process, want 0", *sent)
	}
}

func TestGracefulKill_DiesOnSIGTERM_NoSIGKILL(t *testing.T) {
	swapClock(t)
	sent := swapSignals(t)
	calls := 0
	orig := processAlive
	processAlive = func(pid int) bool {
		calls++
		return calls <= 1 // alive for the initial check, dead on first poll
	}
	t.Cleanup(func() { processAlive = orig })

	if err := GracefulKill(123, time.Second); err != nil {
		t.Fatalf("GracefulKill() = %v", err)
	}
	if len(*sent) != 1 || (*sent)[0] != syscall.SIGTERM {
		t.Errorf("signals = %v, want [SIGTERM] only", *sent)
	}
}

func TestGracefulKill_SurvivesSIGTERM_EscalatesToSIGKILL(t *testing.T) {
	swapClock(t)
	sent := swapSignals(t)
	sigkilled := false
	origKill := killSignal
	killSignal = func(pid int, sig syscall.Signal) error {
		*sent = append(*sent, sig)
		if sig == syscall.SIGKILL {
			sigkilled = true
		}
		return nil
	}
	t.Cleanup(func() { killSignal = origKill })
	orig := processAlive
	processAlive = func(pid int) bool { return !sigkilled }
	t.Cleanup(func() { processAlive = orig })

	if err := GracefulKill(123, 500*time.Millisecond); err != nil {
		t.Fatalf("GracefulKill() = %v", err)
	}
	if len(*sent) < 2 || (*sent)[0] != syscall.SIGTERM || (*sent)[len(*sent)-1] != syscall.SIGKILL {
		t.Errorf("signals = %v, want SIGTERM…SIGKILL", *sent)
	}
}

func TestAlive_NonPositivePID(t *testing.T) {
	if Alive(0) || Alive(-1) {
		t.Error("Alive(<=0) must be false")
	}
}

func TestAlive_EPERMCountsAsAlive(t *testing.T) {
	orig := processAlive
	t.Cleanup(func() { processAlive = orig })
	// processAlive is the real probe; here we assert the exported wrapper
	// forwards it. Substitute to prove the wiring (EPERM handling lives in the
	// real processAlive body).
	processAlive = func(pid int) bool { return pid == 4242 }
	if !Alive(4242) || Alive(4243) {
		t.Error("Alive must forward to processAlive")
	}
}
