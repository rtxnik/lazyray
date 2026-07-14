//go:build !windows

package core

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/rtxnik/lazyray/internal/config"
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

// spawnOrphanIgnoringSIGTERM starts a SIGTERM-ignoring process that is NOT the
// test's child (a launcher backgrounds it and exits, so it is re-parented to
// init/launchd) and returns its pid. It uses no `setsid` (absent on macOS), so
// the process is a non-group-leader — which additionally exercises
// procutil.GracefulKill's single-pid fallback (kill(-pid) -> ESRCH -> kill(pid)),
// the exact shape produced by XrayProcess.Start(). The OLD Wait-based
// gracefulKill could not confirm a non-child's death; the new primitive
// escalates SIGTERM -> SIGKILL and confirms it. t.Cleanup hard-kills it if the
// test leaves it alive.
func spawnOrphanIgnoringSIGTERM(t *testing.T) int {
	t.Helper()
	pidPath := filepath.Join(t.TempDir(), "victim.pid")
	// The launcher backgrounds a child that traps SIGTERM, records its own pid,
	// then execs sleep, and exits immediately — orphaning the child so this test
	// process is not its parent. SIG_IGN survives the exec, so the sleep ignores
	// SIGTERM too.
	launcher := exec.Command("sh", "-c",
		`sh -c 'trap "" TERM; echo $$ > "`+pidPath+`"; exec sleep 300' >/dev/null 2>&1 &`)
	if err := launcher.Run(); err != nil {
		t.Fatalf("launch victim: %v", err)
	}
	var pid int
	for i := 0; i < 200; i++ {
		if b, err := os.ReadFile(pidPath); err == nil {
			if p, err := strconv.Atoi(strings.TrimSpace(string(b))); err == nil && p > 0 {
				pid = p
				break
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	if pid == 0 {
		t.Fatal("victim did not report its pid")
	}
	t.Cleanup(func() { _ = syscall.Kill(pid, syscall.SIGKILL) })
	return pid
}

func TestStopLocked_NonChildXray_EscalatesToKill(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	if err := config.EnsureDirs(); err != nil {
		t.Fatalf("EnsureDirs: %v", err)
	}
	pid := spawnOrphanIgnoringSIGTERM(t)

	// Self-record the PID (the path F1 preserves), NOT findXrayPID.
	if err := writePIDFile(pid); err != nil {
		t.Fatalf("writePIDFile: %v", err)
	}
	restore := SetProcessCmdlineForTest(func(int) (string, error) {
		return config.XrayBinaryPath() + " run -c cfg", nil
	})
	t.Cleanup(restore)

	x := &XrayProcess{} // cmd == nil → readPIDFile path (non-child)
	if err := x.stopLocked(); err != nil {
		t.Fatalf("stopLocked() = %v, want nil", err)
	}
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if !isProcessAlive(pid) {
			return // GREEN: escalated to SIGKILL on the readPIDFile path
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Error("stopLocked left a SIGTERM-ignoring non-child xray alive (no SIGKILL escalation)")
}

// After F1, a pgrep-discovered xray (no self-recorded PID file) is NOT a kill
// target: stopLocked must refuse and leave the process alive. RED on pre-F1
// code (which pgrep-kills it), GREEN after F1.
func TestStopLocked_PgrepSourced_Refused(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	victim := exec.Command("sleep", "30")
	if err := victim.Start(); err != nil {
		t.Fatalf("start victim: %v", err)
	}
	t.Cleanup(func() { _ = victim.Process.Kill(); _, _ = victim.Process.Wait() })
	pid := victim.Process.Pid

	origFind := findXrayPID
	findXrayPID = func() int { return pid } // pgrep would "find" the victim
	t.Cleanup(func() { findXrayPID = origFind })

	// Make identity treat the victim as OUR xray, so pre-F1 code proceeds to kill.
	restore := SetProcessCmdlineForTest(func(int) (string, error) {
		return config.XrayBinaryPath() + " run -c cfg", nil
	})
	t.Cleanup(restore)

	x := &XrayProcess{} // cmd == nil, no PID file written → pgrep fallback path
	err := x.stopLocked()
	if !errors.Is(err, errXrayNotOwned) {
		t.Fatalf("stopLocked() err = %v, want errXrayNotOwned", err)
	}
	if !isProcessAlive(pid) {
		t.Error("stopLocked killed a pgrep-sourced (non-owned) process")
	}
}
