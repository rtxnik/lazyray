package cmd

import (
	"errors"
	"os"
	"testing"

	"github.com/rtxnik/lazyray/internal/config"
	"github.com/rtxnik/lazyray/internal/lifecycle"
)

// A double-start must not record a startup failure: lock contention means
// another supervisor is already live, which is the expected idempotent outcome.
func TestRun_LockContention_DoesNotRecordStartupError(t *testing.T) {
	cleanup := setupTestHome(t)
	defer cleanup()
	writeTestServers(t)
	writeTestSettings(t)

	// Hold the supervisor lock so the run's AcquireLock loses with ErrLocked
	// (before any routing/xray work).
	l, err := lifecycle.AcquireLock()
	if err != nil {
		t.Fatalf("AcquireLock() = %v", err)
	}
	defer func() { _ = l.Release() }()

	runErr := runCmd.RunE(runCmd, []string{})
	if !errors.Is(runErr, lifecycle.ErrLocked) {
		t.Fatalf("RunE() = %v, want ErrLocked", runErr)
	}
	if _, statErr := os.Stat(config.LastErrorPath()); !os.IsNotExist(statErr) {
		t.Errorf("last-error.json exists after lock contention (stat err = %v); contention must not be recorded", statErr)
	}
}
