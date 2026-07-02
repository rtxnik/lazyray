package config

import (
	"os"
	"runtime"
	"testing"
)

func TestDefaultSettings(t *testing.T) {
	s := DefaultSettings()

	if s.Local.SocksPort != 10808 {
		t.Errorf("SocksPort = %d, want 10808", s.Local.SocksPort)
	}
	if s.Local.HTTPPort != 10809 {
		t.Errorf("HTTPPort = %d, want 10809", s.Local.HTTPPort)
	}
	if s.Local.Listen != "127.0.0.1" {
		t.Errorf("Listen = %q, want %q", s.Local.Listen, "127.0.0.1")
	}
	if len(s.Local.DNS) != 2 {
		t.Errorf("DNS count = %d, want 2", len(s.Local.DNS))
	}
}

func TestDefaultSettings_Xray(t *testing.T) {
	s := DefaultSettings()

	if !s.Xray.AutoRestart {
		t.Error("AutoRestart should be true")
	}
	if s.Xray.LogLevel != "warning" {
		t.Errorf("LogLevel = %q, want %q", s.Xray.LogLevel, "warning")
	}
	if s.Xray.MaxLogSize != 10 {
		t.Errorf("MaxLogSize = %d, want 10", s.Xray.MaxLogSize)
	}
}

func TestDefaultSettings_Health(t *testing.T) {
	s := DefaultSettings()

	if s.Health.Timeout != 5 {
		t.Errorf("Timeout = %d, want 5", s.Health.Timeout)
	}
	if !s.Health.AlertOnFailure {
		t.Error("AlertOnFailure should be true")
	}
	if s.Health.IPCheckURL == "" {
		t.Error("IPCheckURL should not be empty")
	}
	if s.Health.LatencyHost == "" {
		t.Error("LatencyHost should not be empty")
	}
	if s.Health.DNSCheckHost == "" {
		t.Error("DNSCheckHost should not be empty")
	}
}

func TestDefaultSettings_Update(t *testing.T) {
	s := DefaultSettings()

	if s.Update.Channel != "stable" {
		t.Errorf("Channel = %q, want %q", s.Update.Channel, "stable")
	}
	if !s.Update.AutoCheck {
		t.Error("AutoCheck should be true")
	}
	if !s.Update.BackupBefore {
		t.Error("BackupBefore should be true")
	}
}

func TestDefaultSettings_UI(t *testing.T) {
	s := DefaultSettings()

	if s.UI.Theme != "dark" {
		t.Errorf("Theme = %q, want %q", s.UI.Theme, "dark")
	}
	if s.UI.RefreshInterval != 5 {
		t.Errorf("RefreshInterval = %d, want 5", s.UI.RefreshInterval)
	}
	if s.UI.LogLines != 100 {
		t.Errorf("LogLines = %d, want 100", s.UI.LogLines)
	}
}

func TestDefaultSettings_Backup(t *testing.T) {
	s := DefaultSettings()

	if s.Backup.MaxFiles != 5 {
		t.Errorf("Backup.MaxFiles = %d, want 5", s.Backup.MaxFiles)
	}
}

func TestDefaultSettings_AutoSystemProxyOn(t *testing.T) {
	if !DefaultSettings().AutoSystemProxy {
		t.Error("DefaultSettings().AutoSystemProxy = false, want true")
	}
}

func TestDefaultSettings_UpdateXrayVersion(t *testing.T) {
	s := DefaultSettings()

	if s.Update.XrayVersion != "v26.3.27" {
		t.Errorf("Update.XrayVersion = %q, want %q", s.Update.XrayVersion, "v26.3.27")
	}
}

func TestSaveSettings_Perm0600(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX permission bits not meaningful on Windows")
	}
	cleanup := setupTestHome(t)
	defer cleanup()

	if err := SaveSettings(DefaultSettings()); err != nil {
		t.Fatalf("SaveSettings: %v", err)
	}
	fi, err := os.Stat(SettingsPath())
	if err != nil {
		t.Fatalf("stat %s: %v", SettingsPath(), err)
	}
	if fi.Mode().Perm() != 0o600 {
		t.Errorf("lazyray.yaml perm = %o, want 0600 (subscription URLs must not be world-readable)", fi.Mode().Perm())
	}
}
