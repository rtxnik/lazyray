package app

import (
	"errors"

	"github.com/rtxnik/lazyray/internal/config"
	"github.com/rtxnik/lazyray/internal/core"
)

// ErrSupervisorRunning signals that a live supervised session owns xray; the
// update must not fight it. Callers surface the stop→update→start guidance.
var ErrSupervisorRunning = errors.New(
	"lazyray is running; stop it first (`lzr stop`), then `lzr update apply`, then `lzr start`")

// ApplyXrayUpdate installs an xray-core update safely with respect to the
// lifecycle supervisor. It self-heals a crash-orphaned session, then HOLDS the
// lifecycle exclusion lock across the whole apply: a failed acquire means a live
// supervisor (or a concurrent update) owns the session → refuse; holding it
// makes the check-and-apply atomic so no concurrent `lzr start`/service relaunch
// can spawn a supervised xray into the swap window.
func (s *Service) ApplyXrayUpdate(xray *core.XrayProcess, release *core.ReleaseInfo,
	downloadURL string, settings *config.Settings, allowUnverified, allowDowngrade bool) error {

	// Self-heal a dead orphan BEFORE taking the lock — Reconcile no-ops under a
	// held lock (its internal SupervisorAlive would see ours). It kills a dead
	// orphan by its state-recorded PID so applyUpdate sees a stopped engine.
	_ = s.reconcile(s.currentProxy())

	lock, err := s.acquireLock()
	if err != nil { // ErrLocked / probe error → treat as owned, fail closed
		return ErrSupervisorRunning
	}
	defer func() { _ = lock.Release() }()

	err = s.applyUpdate(xray, release, downloadURL, settings.Update.BackupBefore, allowUnverified, allowDowngrade)
	if err == nil && settings.Update.BackupBefore {
		s.pruneEngineBackups(settings.Backup.MaxFiles)
	}
	return err
}
