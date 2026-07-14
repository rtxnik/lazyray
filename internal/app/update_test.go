package app

import (
	"errors"
	"testing"

	"github.com/rtxnik/lazyray/internal/config"
	"github.com/rtxnik/lazyray/internal/core"
	"github.com/rtxnik/lazyray/internal/lifecycle"
	"github.com/rtxnik/lazyray/internal/platform"
)

type fakeLock struct{ released bool }

func (f *fakeLock) Release() error { f.released = true; return nil }

func newTestService() (*Service, *[]string) {
	var calls []string
	s := &Service{
		currentProxy: func() platform.SystemProxy { return nil },
		reconcile: func(platform.SystemProxy) error {
			calls = append(calls, "reconcile")
			return nil
		},
	}
	return s, &calls
}

func TestApplyXrayUpdate_RefusesWhenLocked(t *testing.T) {
	s, calls := newTestService()
	s.acquireLock = func() (locker, error) { return nil, lifecycle.ErrLocked }
	applied := false
	s.applyUpdate = func(*core.XrayProcess, *core.ReleaseInfo, string, bool, bool, bool) error {
		applied = true
		return nil
	}
	s.pruneEngineBackups = func(int) {}

	err := s.ApplyXrayUpdate(nil, nil, "", config.DefaultSettings(), false, false)
	if !errors.Is(err, ErrSupervisorRunning) {
		t.Fatalf("err = %v, want ErrSupervisorRunning", err)
	}
	if applied {
		t.Error("applyUpdate ran while a supervisor held the lock")
	}
	if len(*calls) == 0 || (*calls)[0] != "reconcile" {
		t.Error("reconcile must run before the lock attempt")
	}
}

func TestApplyXrayUpdate_HoldsLockAcrossApply_ThenPrunes(t *testing.T) {
	s, calls := newTestService()
	lk := &fakeLock{}
	s.acquireLock = func() (locker, error) { *calls = append(*calls, "acquire"); return lk, nil }
	s.applyUpdate = func(*core.XrayProcess, *core.ReleaseInfo, string, bool, bool, bool) error {
		if lk.released {
			t.Error("lock released before applyUpdate finished")
		}
		*calls = append(*calls, "apply")
		return nil
	}
	s.pruneEngineBackups = func(int) { *calls = append(*calls, "prune") }

	settings := config.DefaultSettings()
	settings.Update.BackupBefore = true
	if err := s.ApplyXrayUpdate(nil, nil, "", settings, false, false); err != nil {
		t.Fatalf("err = %v", err)
	}
	if !lk.released {
		t.Error("lock not released")
	}
	want := []string{"reconcile", "acquire", "apply", "prune"}
	if got := *calls; len(got) != 4 || got[0] != want[0] || got[1] != want[1] || got[2] != want[2] || got[3] != want[3] {
		t.Fatalf("order = %v, want %v", got, want)
	}
}

func TestApplyXrayUpdate_NoPruneWhenBackupDisabledOrError(t *testing.T) {
	s, _ := newTestService()
	s.acquireLock = func() (locker, error) { return &fakeLock{}, nil }
	pruned := false
	s.pruneEngineBackups = func(int) { pruned = true }

	// BackupBefore=false ⇒ no prune.
	noBackup := config.DefaultSettings()
	noBackup.Update.BackupBefore = false
	s.applyUpdate = func(*core.XrayProcess, *core.ReleaseInfo, string, bool, bool, bool) error { return nil }
	_ = s.ApplyXrayUpdate(nil, nil, "", noBackup, false, false)
	if pruned {
		t.Error("pruned with BackupBefore=false")
	}
	// error ⇒ no prune.
	pruned = false
	settings := config.DefaultSettings()
	settings.Update.BackupBefore = true
	s.applyUpdate = func(*core.XrayProcess, *core.ReleaseInfo, string, bool, bool, bool) error { return errors.New("boom") }
	if err := s.ApplyXrayUpdate(nil, nil, "", settings, false, false); err == nil {
		t.Error("want error")
	}
	if pruned {
		t.Error("pruned on applyUpdate error")
	}
}
