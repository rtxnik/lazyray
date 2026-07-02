package config

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// setupTestHome redirects HOME (and Windows equivalents) to a temp dir.
func setupTestHome(t *testing.T) func() {
	t.Helper()
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)

	origAppData := os.Getenv("APPDATA")
	origLocalAppData := os.Getenv("LOCALAPPDATA")
	if runtime.GOOS == "windows" {
		os.Setenv("APPDATA", filepath.Join(tmpDir, "AppData", "Roaming"))
		os.Setenv("LOCALAPPDATA", filepath.Join(tmpDir, "AppData", "Local"))
	}

	return func() {
		os.Setenv("HOME", origHome)
		if runtime.GOOS == "windows" {
			os.Setenv("APPDATA", origAppData)
			os.Setenv("LOCALAPPDATA", origLocalAppData)
		}
	}
}

// --- Path tests ---

func TestConfigDir(t *testing.T) {
	cleanup := setupTestHome(t)
	defer cleanup()

	dir := ConfigDir()
	if dir == "" {
		t.Fatal("ConfigDir() returned empty string")
	}
	if !strings.Contains(dir, "lazyray") {
		t.Errorf("ConfigDir() = %q, should contain 'lazyray'", dir)
	}
}

func TestDataDir(t *testing.T) {
	cleanup := setupTestHome(t)
	defer cleanup()

	dir := DataDir()
	if dir == "" {
		t.Fatal("DataDir() returned empty string")
	}
	if !strings.Contains(dir, "lazyray") {
		t.Errorf("DataDir() = %q, should contain 'lazyray'", dir)
	}
}

func TestServersPath(t *testing.T) {
	cleanup := setupTestHome(t)
	defer cleanup()

	path := ServersPath()
	if !strings.HasSuffix(path, "servers.yaml") {
		t.Errorf("ServersPath() = %q, should end with servers.yaml", path)
	}
}

func TestSettingsPath(t *testing.T) {
	cleanup := setupTestHome(t)
	defer cleanup()

	path := SettingsPath()
	if !strings.HasSuffix(path, "lazyray.yaml") {
		t.Errorf("SettingsPath() = %q, should end with lazyray.yaml", path)
	}
}

func TestXrayBinaryPath(t *testing.T) {
	cleanup := setupTestHome(t)
	defer cleanup()

	path := XrayBinaryPath()
	if runtime.GOOS == "windows" {
		if !strings.HasSuffix(path, "xray.exe") {
			t.Errorf("XrayBinaryPath() = %q, should end with xray.exe on windows", path)
		}
	} else {
		if !strings.HasSuffix(path, "xray") {
			t.Errorf("XrayBinaryPath() = %q, should end with xray", path)
		}
	}
}

func TestXrayConfigPath(t *testing.T) {
	cleanup := setupTestHome(t)
	defer cleanup()

	path := XrayConfigPath()
	if !strings.HasSuffix(path, "config.json") {
		t.Errorf("XrayConfigPath() = %q, should end with config.json", path)
	}
}

func TestLogDir(t *testing.T) {
	cleanup := setupTestHome(t)
	defer cleanup()

	dir := LogDir()
	if !strings.Contains(dir, "logs") {
		t.Errorf("LogDir() = %q, should contain 'logs'", dir)
	}
}

func TestAccessLogPath(t *testing.T) {
	cleanup := setupTestHome(t)
	defer cleanup()

	path := AccessLogPath()
	if !strings.HasSuffix(path, "access.log") {
		t.Errorf("AccessLogPath() = %q, should end with access.log", path)
	}
}

func TestErrorLogPath(t *testing.T) {
	cleanup := setupTestHome(t)
	defer cleanup()

	path := ErrorLogPath()
	if !strings.HasSuffix(path, "error.log") {
		t.Errorf("ErrorLogPath() = %q, should end with error.log", path)
	}
}

func TestBackupDir(t *testing.T) {
	cleanup := setupTestHome(t)
	defer cleanup()

	dir := BackupDir()
	if !strings.Contains(dir, "backup") {
		t.Errorf("BackupDir() = %q, should contain 'backup'", dir)
	}
}

func TestPIDFilePath(t *testing.T) {
	cleanup := setupTestHome(t)
	defer cleanup()

	path := PIDFilePath()
	if !strings.HasSuffix(path, "xray.pid") {
		t.Errorf("PIDFilePath() = %q, should end with xray.pid", path)
	}
}

func TestTunnelPIDPath(t *testing.T) {
	cleanup := setupTestHome(t)
	defer cleanup()

	path := TunnelPIDPath("my-server")
	if !strings.Contains(path, "tunnel-my-server.pid") {
		t.Errorf("TunnelPIDPath() = %q, should contain tunnel-my-server.pid", path)
	}
}

func TestTunnelPIDPath_Sanitized(t *testing.T) {
	cleanup := setupTestHome(t)
	defer cleanup()

	path := TunnelPIDPath("my server / special")
	if strings.Contains(path, " ") || strings.Contains(path, "/special") {
		t.Errorf("TunnelPIDPath() = %q, should sanitize unsafe characters", path)
	}
}

func TestTunnelPIDGlob_Pattern(t *testing.T) {
	cleanup := setupTestHome(t)
	defer cleanup()

	glob := TunnelPIDGlob()
	if !strings.Contains(glob, "tunnel-*.pid") {
		t.Errorf("TunnelPIDGlob() = %q, should contain tunnel-*.pid", glob)
	}
}

// --- EnsureDirs ---

func TestEnsureDirs(t *testing.T) {
	cleanup := setupTestHome(t)
	defer cleanup()

	err := EnsureDirs()
	if err != nil {
		t.Fatalf("EnsureDirs() error = %v", err)
	}

	// Verify directories were created
	for _, dir := range []string{ConfigDir(), DataDir(), LogDir(), BackupDir()} {
		info, err := os.Stat(dir)
		if err != nil {
			t.Errorf("directory %q should exist after EnsureDirs: %v", dir, err)
			continue
		}
		if !info.IsDir() {
			t.Errorf("%q should be a directory", dir)
		}
	}
}

// --- LoadServers / SaveServers ---

func TestLoadServers_NoFile(t *testing.T) {
	cleanup := setupTestHome(t)
	defer cleanup()
	if err := EnsureDirs(); err != nil {
		t.Fatalf("EnsureDirs: %v", err)
	}

	servers, err := LoadServers()
	if err != nil {
		t.Fatalf("LoadServers() with no file error = %v", err)
	}
	if len(servers.Profiles) != 0 {
		t.Errorf("LoadServers() with no file should return empty profiles, got %d", len(servers.Profiles))
	}
}

func TestLoadServers_WithFile(t *testing.T) {
	cleanup := setupTestHome(t)
	defer cleanup()
	if err := EnsureDirs(); err != nil {
		t.Fatalf("EnsureDirs: %v", err)
	}

	yaml := `profiles:
  - name: "Test"
    default: true
    server:
      address: "1.2.3.4"
      port: 443
      uuid: "test-uuid"
      encryption: "none"
      transport:
        network: "tcp"
      security:
        type: "none"
`
	if err := os.WriteFile(ServersPath(), []byte(yaml), 0600); err != nil {
		t.Fatalf("failed to write servers.yaml: %v", err)
	}

	servers, err := LoadServers()
	if err != nil {
		t.Fatalf("LoadServers() error = %v", err)
	}
	if len(servers.Profiles) != 1 {
		t.Fatalf("expected 1 profile, got %d", len(servers.Profiles))
	}
	if servers.Profiles[0].Name != "Test" {
		t.Errorf("profile name = %q, want Test", servers.Profiles[0].Name)
	}
}

func TestSaveServers(t *testing.T) {
	cleanup := setupTestHome(t)
	defer cleanup()

	servers := &ServersConfig{
		Profiles: []Profile{
			{
				Name:    "SaveTest",
				Default: true,
				Server: ServerConfig{
					Address:    "1.2.3.4",
					Port:       443,
					UUID:       "save-uuid",
					Encryption: "none",
					Transport:  TransportConfig{Network: "tcp"},
					Security:   SecurityConfig{Type: "none"},
				},
			},
		},
	}

	if err := SaveServers(servers); err != nil {
		t.Fatalf("SaveServers() error = %v", err)
	}

	// Re-load and verify
	loaded, err := LoadServers()
	if err != nil {
		t.Fatalf("LoadServers() after save error = %v", err)
	}
	if len(loaded.Profiles) != 1 {
		t.Fatalf("expected 1 profile after save, got %d", len(loaded.Profiles))
	}
	if loaded.Profiles[0].Name != "SaveTest" {
		t.Errorf("loaded name = %q, want SaveTest", loaded.Profiles[0].Name)
	}
}

// --- DefaultProfile ---

func TestDefaultProfile_WithDefault(t *testing.T) {
	servers := &ServersConfig{
		Profiles: []Profile{
			{Name: "A"},
			{Name: "B", Default: true},
			{Name: "C"},
		},
	}

	p := servers.DefaultProfile()
	if p == nil {
		t.Fatal("DefaultProfile() returned nil")
	}
	if p.Name != "B" {
		t.Errorf("DefaultProfile() = %q, want B", p.Name)
	}
}

func TestDefaultProfile_NoDefault(t *testing.T) {
	servers := &ServersConfig{
		Profiles: []Profile{
			{Name: "A"},
			{Name: "B"},
		},
	}

	p := servers.DefaultProfile()
	if p == nil {
		t.Fatal("DefaultProfile() returned nil")
	}
	if p.Name != "A" {
		t.Errorf("DefaultProfile() with no default should return first, got %q", p.Name)
	}
}

func TestDefaultProfile_Empty(t *testing.T) {
	servers := &ServersConfig{}
	p := servers.DefaultProfile()
	if p != nil {
		t.Error("DefaultProfile() with no profiles should return nil")
	}
}

// --- SetDefault ---

func TestSetDefault(t *testing.T) {
	servers := &ServersConfig{
		Profiles: []Profile{
			{Name: "A", Default: true},
			{Name: "B"},
			{Name: "C"},
		},
	}

	if err := servers.SetDefault(2); err != nil {
		t.Fatalf("SetDefault(2) error = %v", err)
	}

	if servers.Profiles[0].Default {
		t.Error("A should not be default")
	}
	if servers.Profiles[1].Default {
		t.Error("B should not be default")
	}
	if !servers.Profiles[2].Default {
		t.Error("C should be default")
	}
}

func TestSetDefault_OutOfRange(t *testing.T) {
	servers := &ServersConfig{
		Profiles: []Profile{{Name: "A"}},
	}

	if err := servers.SetDefault(5); err == nil {
		t.Error("SetDefault(5) should return error for out of range")
	}
	if err := servers.SetDefault(-1); err == nil {
		t.Error("SetDefault(-1) should return error for negative index")
	}
}

// --- HasUUID ---

func TestHasUUID_Found(t *testing.T) {
	servers := &ServersConfig{
		Profiles: []Profile{
			{Name: "A", Server: ServerConfig{UUID: "uuid-a"}},
			{Name: "B", Server: ServerConfig{UUID: "uuid-b"}},
		},
	}

	name, found := servers.HasUUID("uuid-b")
	if !found {
		t.Error("HasUUID should find uuid-b")
	}
	if name != "B" {
		t.Errorf("HasUUID name = %q, want B", name)
	}
}

func TestHasUUID_NotFound(t *testing.T) {
	servers := &ServersConfig{
		Profiles: []Profile{
			{Name: "A", Server: ServerConfig{UUID: "uuid-a"}},
		},
	}

	_, found := servers.HasUUID("nonexistent")
	if found {
		t.Error("HasUUID should not find nonexistent uuid")
	}
}

// --- MaskUUID ---

func TestMaskUUID_Variants(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"123e4567-e89b-42d3-a456-426614174000", "123e4567-****-****-****-************"},
		{"short", "****"},
		{"12345678", "****"},
		{"123456789", "12345678-****-****-****-************"},
	}

	for _, tc := range tests {
		got := MaskUUID(tc.input)
		if got != tc.want {
			t.Errorf("MaskUUID(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

// --- IsChained ---

func TestIsChained_No(t *testing.T) {
	p := Profile{Name: "test", Server: ServerConfig{Address: "1.2.3.4"}}
	if p.IsChained() {
		t.Error("profile with no chain should not be chained")
	}
}

func TestIsChained_Yes(t *testing.T) {
	p := Profile{
		Name:   "test",
		Server: ServerConfig{Address: "1.2.3.4"},
		Chain:  []ServerConfig{{Address: "5.6.7.8"}},
	}
	if !p.IsChained() {
		t.Error("profile with chain should be chained")
	}
}

// --- ChainServers ---

func TestChainServers_Single(t *testing.T) {
	p := Profile{Server: ServerConfig{Address: "1.2.3.4"}}
	servers := p.ChainServers()
	if len(servers) != 1 {
		t.Fatalf("ChainServers single = %d, want 1", len(servers))
	}
	if servers[0].Address != "1.2.3.4" {
		t.Errorf("ChainServers[0].Address = %q, want 1.2.3.4", servers[0].Address)
	}
}

func TestChainServers_Multi(t *testing.T) {
	p := Profile{
		Server: ServerConfig{Address: "entry"},
		Chain: []ServerConfig{
			{Address: "hop1"},
			{Address: "exit"},
		},
	}
	servers := p.ChainServers()
	if len(servers) != 3 {
		t.Fatalf("ChainServers multi = %d, want 3", len(servers))
	}
	if servers[0].Address != "entry" {
		t.Errorf("servers[0] = %q, want entry", servers[0].Address)
	}
	if servers[1].Address != "hop1" {
		t.Errorf("servers[1] = %q, want hop1", servers[1].Address)
	}
	if servers[2].Address != "exit" {
		t.Errorf("servers[2] = %q, want exit", servers[2].Address)
	}
}

// --- LoadSettings / SaveSettings ---

func TestLoadSettings_NoFile(t *testing.T) {
	cleanup := setupTestHome(t)
	defer cleanup()
	if err := EnsureDirs(); err != nil {
		t.Fatalf("EnsureDirs: %v", err)
	}

	settings, err := LoadSettings()
	if err != nil {
		t.Fatalf("LoadSettings() with no file error = %v", err)
	}
	// Should return defaults
	if settings.Local.SocksPort != 10808 {
		t.Errorf("default SocksPort = %d, want 10808", settings.Local.SocksPort)
	}
}

func TestLoadSettings_WithFile(t *testing.T) {
	cleanup := setupTestHome(t)
	defer cleanup()
	if err := EnsureDirs(); err != nil {
		t.Fatalf("EnsureDirs: %v", err)
	}

	yaml := `local:
  socksPort: 1080
  httpPort: 1081
  listen: "0.0.0.0"
  dns:
    - "9.9.9.9"
`
	if err := os.WriteFile(SettingsPath(), []byte(yaml), 0644); err != nil {
		t.Fatalf("failed to write settings: %v", err)
	}

	settings, err := LoadSettings()
	if err != nil {
		t.Fatalf("LoadSettings() error = %v", err)
	}
	if settings.Local.SocksPort != 1080 {
		t.Errorf("SocksPort = %d, want 1080", settings.Local.SocksPort)
	}
	if settings.Local.Listen != "0.0.0.0" {
		t.Errorf("Listen = %q, want 0.0.0.0", settings.Local.Listen)
	}
}

func TestSaveSettings(t *testing.T) {
	cleanup := setupTestHome(t)
	defer cleanup()

	settings := DefaultSettings()
	settings.Local.SocksPort = 9999

	if err := SaveSettings(settings); err != nil {
		t.Fatalf("SaveSettings() error = %v", err)
	}

	loaded, err := LoadSettings()
	if err != nil {
		t.Fatalf("LoadSettings() after save error = %v", err)
	}
	if loaded.Local.SocksPort != 9999 {
		t.Errorf("loaded SocksPort = %d, want 9999", loaded.Local.SocksPort)
	}
}

// --- DefaultSettings ---

func TestDefaultSettings_Complete(t *testing.T) {
	s := DefaultSettings()

	if s.Local.SocksPort != 10808 {
		t.Errorf("SocksPort = %d, want 10808", s.Local.SocksPort)
	}
	if s.Local.HTTPPort != 10809 {
		t.Errorf("HTTPPort = %d, want 10809", s.Local.HTTPPort)
	}
	if s.Local.Listen != "127.0.0.1" {
		t.Errorf("Listen = %q, want 127.0.0.1", s.Local.Listen)
	}
	if len(s.Local.DNS) != 2 {
		t.Errorf("DNS count = %d, want 2", len(s.Local.DNS))
	}
	if s.Xray.LogLevel != "warning" {
		t.Errorf("LogLevel = %q, want warning", s.Xray.LogLevel)
	}
	if !s.Xray.AutoRestart {
		t.Error("AutoRestart should be true by default")
	}
	if s.Xray.MaxLogSize != 10 {
		t.Errorf("MaxLogSize = %d, want 10", s.Xray.MaxLogSize)
	}
	if s.Health.Timeout != 5 {
		t.Errorf("Health.Timeout = %d, want 5", s.Health.Timeout)
	}
	if !s.Health.AlertOnFailure {
		t.Error("AlertOnFailure should be true by default")
	}
	if s.Update.Channel != "stable" {
		t.Errorf("Channel = %q, want stable", s.Update.Channel)
	}
	if !s.Update.AutoCheck {
		t.Error("AutoCheck should be true by default")
	}
	if !s.Update.BackupBefore {
		t.Error("BackupBefore should be true by default")
	}
	if s.UI.Theme != "dark" {
		t.Errorf("Theme = %q, want dark", s.UI.Theme)
	}
	if s.UI.RefreshInterval != 5 {
		t.Errorf("RefreshInterval = %d, want 5", s.UI.RefreshInterval)
	}
	if s.UI.LogLines != 100 {
		t.Errorf("LogLines = %d, want 100", s.UI.LogLines)
	}
}
