package core

import (
	"os"
	"runtime"
	"testing"

	"github.com/rtxnik/lazyray/internal/config"
)

func TestWriteXrayConfig_Perms0600(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX file permissions are not honored on Windows")
	}
	t.Setenv("HOME", t.TempDir())
	_ = config.EnsureDirs()
	if err := WriteXrayConfig(testProfile(), testSettings()); err != nil {
		t.Fatalf("WriteXrayConfig() = %v", err)
	}
	info, err := os.Stat(config.XrayConfigPath())
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if info.Mode().Perm() != 0600 {
		t.Errorf("config.json perm = %v, want -rw------- (embeds credentials)", info.Mode().Perm())
	}
}

func TestStatsSave_Perms0600(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX file permissions are not honored on Windows")
	}
	t.Setenv("HOME", t.TempDir())
	_ = config.EnsureDirs()
	sm := &StatsManager{history: &StatsHistory{}}
	if err := sm.Save(); err != nil {
		t.Fatalf("Save() = %v", err)
	}
	info, err := os.Stat(config.StatsPath())
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if info.Mode().Perm() != 0600 {
		t.Errorf("stats.json perm = %v, want -rw------- (usage metadata)", info.Mode().Perm())
	}
}
