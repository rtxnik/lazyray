//go:build !windows

package core

import (
	"os"
	"os/exec"
	"strconv"
	"syscall"
	"testing"
	"time"

	"github.com/rtxnik/lazyray/internal/config"
)

// writeTunnelPIDFile stores a one-line PID file at the tunnel pidfile path for `name`.
func writeTunnelPIDFile(t *testing.T, name string, pid int) {
	t.Helper()
	if err := config.EnsureDirs(); err != nil {
		t.Fatalf("EnsureDirs: %v", err)
	}
	if err := os.WriteFile(config.TunnelPIDPath(name), []byte(itoa(pid)), 0o600); err != nil {
		t.Fatalf("write pidfile: %v", err)
	}
}

func itoa(n int) string { return strconv.Itoa(n) }

// childTerminated reports whether the direct-child pid was signalled or exited,
// reaping any pending zombie state non-blockingly. Needed because isProcessAlive
// (Signal(0)) reports a still-unreaped, already-killed child as "alive" and
// because SIGKILL delivery is asynchronous — so it is an unreliable witness for
// "did the kill path actually terminate our subprocess?".
func childTerminated(pid int) bool {
	for i := 0; i < 50; i++ {
		var ws syscall.WaitStatus
		wpid, err := syscall.Wait4(pid, &ws, syscall.WNOHANG, nil)
		if err == nil && wpid == pid && (ws.Signaled() || ws.Exited()) {
			return true
		}
		time.Sleep(10 * time.Millisecond)
	}
	return false
}

func TestCloseAllPersistentTunnels_ForeignPID_NotKilled(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	victim := exec.Command("sleep", "30")
	if err := victim.Start(); err != nil {
		t.Fatalf("start victim: %v", err)
	}
	t.Cleanup(func() { _ = victim.Process.Kill(); _, _ = victim.Process.Wait() })
	foreign := victim.Process.Pid

	writeTunnelPIDFile(t, "vps", foreign)

	restore := SetProcessCmdlineForTest(func(int) (string, error) {
		return "/usr/bin/sleep 30", nil // not an ssh -N -L tunnel
	})
	t.Cleanup(restore)

	CloseAllPersistentTunnels()

	if childTerminated(foreign) {
		t.Error("CloseAllPersistentTunnels killed a foreign (non-tunnel) process")
	}
	if _, err := os.Stat(config.TunnelPIDPath("vps")); !os.IsNotExist(err) {
		t.Error("stale tunnel pidfile should be removed even when the PID is foreign")
	}
}

func TestCloseAllPersistentTunnels_OurTunnel_Killed(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	victim := exec.Command("sleep", "30")
	if err := victim.Start(); err != nil {
		t.Fatalf("start victim: %v", err)
	}
	t.Cleanup(func() { _ = victim.Process.Kill(); _, _ = victim.Process.Wait() })
	ours := victim.Process.Pid

	writeTunnelPIDFile(t, "vps", ours)

	restore := SetProcessCmdlineForTest(func(int) (string, error) {
		return "ssh -L 5000:127.0.0.1:443 -N user@host", nil // our tunnel shape
	})
	t.Cleanup(restore)

	CloseAllPersistentTunnels()

	_, _ = victim.Process.Wait()
	if isProcessAlive(ours) {
		t.Error("CloseAllPersistentTunnels should have killed our own tunnel process")
	}
}

func TestStatus_ForeignLivePID_NotConnected(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	victim := exec.Command("sleep", "30")
	if err := victim.Start(); err != nil {
		t.Fatalf("start victim: %v", err)
	}
	t.Cleanup(func() { _ = victim.Process.Kill(); _, _ = victim.Process.Wait() })
	foreign := victim.Process.Pid

	writeTunnelPIDFile(t, "vps", foreign)

	restore := SetProcessCmdlineForTest(func(int) (string, error) {
		return "/usr/bin/sleep 30", nil // NOT an ssh -N -L tunnel
	})
	t.Cleanup(restore)

	tm := NewTunnelManager()
	statuses := tm.Status([]config.Profile{{Name: "vps", SSH: config.SSHConfig{Host: "h"}}})
	if len(statuses) != 1 {
		t.Fatalf("got %d statuses, want 1", len(statuses))
	}
	if statuses[0].Connected {
		t.Error("Status reported Connected=true for a foreign (non-tunnel) live PID")
	}
	if _, err := os.Stat(config.TunnelPIDPath("vps")); !os.IsNotExist(err) {
		t.Error("Status should remove the stale/foreign tunnel pidfile")
	}
}
