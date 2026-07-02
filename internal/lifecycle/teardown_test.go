// internal/lifecycle/teardown_test.go
package lifecycle

import (
	"testing"
)

func TestTeardown_RevertsBeforeKill(t *testing.T) {
	withTempData(t)
	var order []string
	fp := &fakeProxy{}
	// Wrap Disable to record ordering relative to kill.
	origDisable := fp.Disable
	_ = origDisable
	fp2 := &recordingProxy{fakeProxy: fp, order: &order}

	_ = WriteState(&State{Owner: OwnerDaemon, XrayPID: 4242, Routing: Routing{SystemProxy: true}})
	st, _ := ReadState()

	kill := func(pid int) error {
		order = append(order, "kill")
		return nil
	}
	if err := Teardown(st, fp2, kill); err != nil {
		t.Fatalf("Teardown() = %v", err)
	}
	if len(order) != 2 || order[0] != "revert" || order[1] != "kill" {
		t.Errorf("order = %v, want [revert kill]", order)
	}
	if got, _ := ReadState(); got != nil {
		t.Error("state file not removed after Teardown")
	}
}

func TestReconcile_StaleStateFreeLock_CleansUp(t *testing.T) {
	withTempData(t)
	fp := &fakeProxy{}
	_ = WriteState(&State{Owner: OwnerDaemon, XrayPID: 0, Routing: Routing{SystemProxy: true}})

	// No lock held → supervisor considered dead → reconcile must revert+remove.
	if err := Reconcile(fp); err != nil {
		t.Fatalf("Reconcile() = %v", err)
	}
	if !contains(fp.calls, "disable") {
		t.Errorf("calls = %v, want disable during self-heal", fp.calls)
	}
	if got, _ := ReadState(); got != nil {
		t.Error("stale state not removed by Reconcile")
	}
}

func TestReconcile_LiveLock_NoOp(t *testing.T) {
	withTempData(t)
	fp := &fakeProxy{}
	_ = WriteState(&State{Owner: OwnerDaemon, Routing: Routing{SystemProxy: true}})
	l, _ := AcquireLock() // simulate a live supervisor holding the lock
	defer func() { _ = l.Release() }()

	if err := Reconcile(fp); err != nil {
		t.Fatalf("Reconcile() = %v", err)
	}
	if contains(fp.calls, "disable") {
		t.Error("Reconcile reverted routing while supervisor is alive")
	}
	if got, _ := ReadState(); got == nil {
		t.Error("Reconcile removed state while supervisor is alive")
	}
}

// recordingProxy records the relative order of Disable() for the ordering test.
type recordingProxy struct {
	*fakeProxy
	order *[]string
}

func (r *recordingProxy) Disable() error {
	*r.order = append(*r.order, "revert")
	return r.fakeProxy.Disable()
}
