//go:build !windows

package core

import (
	"os/exec"
	"strings"
	"testing"
)

func TestStopLocked_ForeignPID_NotKilled(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	// A real, unrelated process that lazyray will be tricked into "finding".
	victim := exec.Command("sleep", "30")
	if err := victim.Start(); err != nil {
		t.Fatalf("start victim: %v", err)
	}
	t.Cleanup(func() { _ = victim.Process.Kill(); _, _ = victim.Process.Wait() })
	foreign := victim.Process.Pid

	origFind := findXrayPID
	findXrayPID = func() int { return foreign }
	t.Cleanup(func() { findXrayPID = origFind })

	restore := SetProcessCmdlineForTest(func(int) (string, error) {
		return "/usr/bin/sleep 30", nil // NOT our xray
	})
	t.Cleanup(restore)

	x := &XrayProcess{} // cmd == nil → falls through to the pidfile/find path
	err := x.stopLocked()
	if err == nil || !strings.Contains(err.Error(), "not running") {
		t.Fatalf("stopLocked should refuse a foreign PID, got err=%v", err)
	}
	if !isProcessAlive(foreign) {
		t.Error("stopLocked killed an unrelated (foreign) process")
	}
}
