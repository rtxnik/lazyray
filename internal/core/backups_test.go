//go:build !windows

package core

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/rtxnik/lazyray/internal/config"
)

func TestPruneEngineBackups_KeepsNewestSets_SparesConfigArchives(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	if err := config.EnsureDirs(); err != nil {
		t.Fatalf("EnsureDirs: %v", err)
	}
	dir := config.BackupDir()
	write := func(name string) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	// Three engine sets (oldest→newest ts) + one config archive.
	for _, ts := range []string{"20260101-000000", "20260102-000000", "20260103-000000"} {
		write("xray." + ts + ".bak")
		write("geoip.dat." + ts + ".bak")
		write("geosite.dat." + ts + ".bak")
	}
	write("lazyray-backup-20250101-000000.tar.gz.enc")

	PruneEngineBackups(1) // keep only the newest set

	exists := func(name string) bool {
		_, err := os.Stat(filepath.Join(dir, name))
		return err == nil
	}
	if !exists("xray.20260103-000000.bak") || !exists("geoip.dat.20260103-000000.bak") {
		t.Error("newest engine set was pruned")
	}
	if exists("xray.20260101-000000.bak") || exists("xray.20260102-000000.bak") {
		t.Error("older engine sets not pruned")
	}
	if !exists("lazyray-backup-20250101-000000.tar.gz.enc") {
		t.Error("config archive was deleted (must never happen)")
	}
}
