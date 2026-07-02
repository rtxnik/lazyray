//go:build !windows

package lifecycle

import (
	"testing"

	"github.com/rtxnik/lazyray/internal/core"
)

func TestReconcile_SkipsKillForForeignPID(t *testing.T) {
	withTempData(t)
	restore := core.SetProcessCmdlineForTest(func(int) (string, error) {
		return "/usr/bin/something-else", nil
	})
	t.Cleanup(restore)

	_ = WriteState(&State{XrayPID: 999999, Routing: Routing{SystemProxy: true}})
	fp := &fakeProxy{}
	if err := Reconcile(fp); err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	// Routing still reverted and state removed, but the foreign PID is NOT killed.
	if !contains(fp.calls, "disable") {
		t.Error("expected routing revert during self-heal")
	}
	if st, _ := ReadState(); st != nil {
		t.Error("stale state not removed")
	}
}
