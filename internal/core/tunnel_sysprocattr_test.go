//go:build !windows

package core

import (
	"syscall"
	"testing"
)

// Asserts the REAL default ssh command carries the detach attr, exercised
// through newSSHCmd (the seam-swap tests replace startSSHProcess and never run
// the default spawn). No t.Parallel: touches package-level construction only.
func TestNewSSHCmd_SetsDetachedProcAttr(t *testing.T) {
	cmd := newSSHCmd([]string{"-N", "user@host"})
	if cmd.SysProcAttr == nil {
		t.Fatal("newSSHCmd did not set SysProcAttr (tunnel would not detach)")
	}
	if !cmd.SysProcAttr.Setsid {
		t.Errorf("SysProcAttr.Setsid = false, want true (detached session) — got %+v", cmd.SysProcAttr)
	}
	_ = syscall.SIGTERM // keep syscall import if trimmed elsewhere
}
