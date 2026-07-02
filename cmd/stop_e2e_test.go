//go:build e2e

package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/rtxnik/lazyray/internal/config"
	"github.com/rtxnik/lazyray/internal/lifecycle"
)

// This e2e builds the real lzr binary, installs a fake "xray" that sleeps,
// starts the supervisor, asserts state, then stops and asserts full cleanup.
func TestStartStop_RoundTrip(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	// Fake xray binary that responds to "version" and, for "run", exec's a
	// real long-lived (SIGTERM-killable) sleep.
	xrayPath := config.XrayBinaryPath()
	if err := os.MkdirAll(filepath.Dir(xrayPath), 0755); err != nil {
		t.Fatalf("mkdir xray dir: %v", err)
	}
	writeFakeXray(t, xrayPath)

	// Minimal profile + config so the supervisor can launch.
	seedMinimalProfile(t)

	lzr := buildLZR(t)

	// Best-effort cleanup: if the test fails mid-way, do not leak the
	// supervisor or its child sleep process.
	t.Cleanup(func() {
		if st, _ := lifecycle.ReadState(); st != nil {
			if st.SupervisorPID > 0 {
				_ = exec.Command(lzr, "stop").Run()
			}
		}
		if lifecycle.SupervisorAlive() {
			if st, _ := lifecycle.ReadState(); st != nil && st.SupervisorPID > 0 {
				_ = syscallKillBestEffort(st.SupervisorPID)
			}
		}
	})

	if out, err := exec.Command(lzr, "start").CombinedOutput(); err != nil {
		t.Fatalf("start: %v\n%s", err, out)
	}
	if !lifecycle.SupervisorAlive() {
		t.Fatal("supervisor not alive after start")
	}

	if out, err := exec.Command(lzr, "stop").CombinedOutput(); err != nil {
		t.Fatalf("stop: %v\n%s", err, out)
	}
	// Allow the lock to release.
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) && lifecycle.SupervisorAlive() {
		time.Sleep(50 * time.Millisecond)
	}
	if lifecycle.SupervisorAlive() {
		t.Error("supervisor still alive after stop")
	}
	if st, _ := lifecycle.ReadState(); st != nil {
		t.Error("state file not removed after stop")
	}
}
