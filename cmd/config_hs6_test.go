package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rtxnik/lazyray/internal/config"
)

func TestBackupRefusesSymlinkedSource(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := config.EnsureDirs(); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(home, "secret")
	if err := os.WriteFile(outside, []byte("PRIVATE"), 0o600); err != nil {
		t.Fatal(err)
	}
	_ = os.Remove(config.ServersPath())
	if err := os.Symlink(outside, config.ServersPath()); err != nil {
		t.Skipf("symlink unsupported: %v", err)
	}
	backupNoEncrypt = true
	t.Cleanup(func() { backupNoEncrypt = false })

	out := filepath.Join(home, "b.tar.gz")
	err := configBackupCmd.RunE(configBackupCmd, []string{out})
	if err == nil {
		t.Fatal("backup followed a symlinked source")
	}
	// The failure must be the refusal to open the symlinked source, not an
	// unrelated backup error masking a regression.
	if !strings.Contains(err.Error(), "servers.yaml") {
		t.Fatalf("unexpected error (not the symlink refusal): %v", err)
	}
}
