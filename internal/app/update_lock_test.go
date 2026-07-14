//go:build !windows

package app

import (
	"errors"
	"testing"

	"github.com/rtxnik/lazyray/internal/config"
	"github.com/rtxnik/lazyray/internal/lifecycle"
	"github.com/rtxnik/lazyray/internal/platform"
)

// TestApplyXrayUpdate_RealLockHeld_Refuses proves the held lifecycle lock — not
// a bare SupervisorAlive() probe — is the real fence (codex-H1 / spec §4): with
// a real lifecycle.AcquireLock held in-process, the flow wired through
// NewService's real acquireLock seam must refuse with ErrSupervisorRunning
// before ever calling applyUpdate. reconcile/currentProxy are stubbed to keep
// the test off the platform proxy and focused on the lock fence.
func TestApplyXrayUpdate_RealLockHeld_Refuses(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	lk, err := lifecycle.AcquireLock()
	if err != nil {
		t.Fatalf("AcquireLock: %v", err)
	}
	t.Cleanup(func() { _ = lk.Release() })

	s := NewService() // real acquireLock -> lifecycle.AcquireLock
	s.reconcile = func(platform.SystemProxy) error { return nil }
	s.currentProxy = func() platform.SystemProxy { return nil }

	err = s.ApplyXrayUpdate(nil, nil, "", config.DefaultSettings(), false, false)
	if !errors.Is(err, ErrSupervisorRunning) {
		t.Fatalf("ApplyXrayUpdate with a real lock held = %v, want ErrSupervisorRunning", err)
	}
}
