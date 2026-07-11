package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/rtxnik/lazyray/internal/config"
)

// setupTestHome sets HOME to a temp dir and creates necessary config dirs and files.
// Returns a cleanup function.
func setupTestHome(t *testing.T) func() {
	t.Helper()
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)

	// Also set APPDATA/LOCALAPPDATA for windows compatibility
	origAppData := os.Getenv("APPDATA")
	origLocalAppData := os.Getenv("LOCALAPPDATA")
	if runtime.GOOS == "windows" {
		os.Setenv("APPDATA", filepath.Join(tmpDir, "AppData", "Roaming"))
		os.Setenv("LOCALAPPDATA", filepath.Join(tmpDir, "AppData", "Local"))
	}

	// Create config and data dirs
	if err := config.EnsureDirs(); err != nil {
		t.Fatalf("failed to create dirs: %v", err)
	}

	return func() {
		os.Setenv("HOME", origHome)
		if runtime.GOOS == "windows" {
			os.Setenv("APPDATA", origAppData)
			os.Setenv("LOCALAPPDATA", origLocalAppData)
		}
	}
}

func writeTestServers(t *testing.T) {
	t.Helper()
	serversYAML := `profiles:
  - name: "Test Server 1"
    default: true
    server:
      address: "1.2.3.4"
      port: 443
      uuid: "test-uuid-1"
      encryption: "none"
      transport:
        network: "xhttp"
        path: "/test"
        mode: "auto"
      security:
        type: "reality"
        sni: "example.com"
        fingerprint: "chrome"
        publicKey: "TEST_KEY"
        shortId: "abc123"
  - name: "Test Server 2"
    server:
      address: "5.6.7.8"
      port: 443
      uuid: "test-uuid-2"
      encryption: "none"
      transport:
        network: "tcp"
      security:
        type: "none"
`
	if err := os.WriteFile(config.ServersPath(), []byte(serversYAML), 0600); err != nil {
		t.Fatalf("failed to write servers.yaml: %v", err)
	}
}

func writeTestSettings(t *testing.T) {
	t.Helper()
	settingsYAML := `local:
  socksPort: 10808
  httpPort: 10809
  listen: "127.0.0.1"
  dns:
    - "1.1.1.1"
    - "8.8.8.8"
xray:
  logLevel: "warning"
  autoRestart: false
  maxLogSize: 10
health:
  timeout: 5
notifications:
  enabled: true
`
	if err := os.WriteFile(config.SettingsPath(), []byte(settingsYAML), 0600); err != nil {
		t.Fatalf("failed to write settings: %v", err)
	}
}

// --- Config list tests ---

func TestConfigListCmd_RunE_Text(t *testing.T) {
	cleanup := setupTestHome(t)
	defer cleanup()
	writeTestServers(t)

	var buf bytes.Buffer
	configListCmd.SetOut(&buf)
	configListCmd.SetErr(&buf)

	err := configListCmd.RunE(configListCmd, []string{})
	if err != nil {
		t.Fatalf("config list RunE error: %v", err)
	}
}

func TestConfigListCmd_RunE_JSON(t *testing.T) {
	cleanup := setupTestHome(t)
	defer cleanup()
	writeTestServers(t)

	// Set the json flag
	if err := configListCmd.Flags().Set("json", "true"); err != nil {
		t.Fatalf("Set flag: %v", err)
	}
	defer func() { _ = configListCmd.Flags().Set("json", "false") }()

	var buf bytes.Buffer
	configListCmd.SetOut(&buf)
	configListCmd.SetErr(&buf)

	err := configListCmd.RunE(configListCmd, []string{})
	if err != nil {
		t.Fatalf("config list --json RunE error: %v", err)
	}
}

func TestConfigListCmd_RunE_Empty(t *testing.T) {
	cleanup := setupTestHome(t)
	defer cleanup()
	// Don't write any servers

	var buf bytes.Buffer
	configListCmd.SetOut(&buf)
	configListCmd.SetErr(&buf)

	err := configListCmd.RunE(configListCmd, []string{})
	if err != nil {
		t.Fatalf("config list RunE with no profiles error: %v", err)
	}
}

// --- Config switch tests ---

func TestConfigSwitchCmd_RunE(t *testing.T) {
	cleanup := setupTestHome(t)
	defer cleanup()
	writeTestServers(t)

	err := configSwitchCmd.RunE(configSwitchCmd, []string{"Test Server 2"})
	if err != nil {
		t.Fatalf("config switch RunE error: %v", err)
	}

	// Verify the switch happened
	servers, err := config.LoadServers()
	if err != nil {
		t.Fatalf("LoadServers error: %v", err)
	}
	for _, p := range servers.Profiles {
		if p.Name == "Test Server 2" && !p.Default {
			t.Error("Test Server 2 should be default after switch")
		}
		if p.Name == "Test Server 1" && p.Default {
			t.Error("Test Server 1 should not be default after switch")
		}
	}
}

func TestConfigSwitchCmd_RunE_NotFound(t *testing.T) {
	cleanup := setupTestHome(t)
	defer cleanup()
	writeTestServers(t)

	err := configSwitchCmd.RunE(configSwitchCmd, []string{"Nonexistent"})
	if err == nil {
		t.Error("config switch with nonexistent profile should return error")
	}
}

// --- Config delete tests ---

func TestConfigDeleteCmd_RunE(t *testing.T) {
	cleanup := setupTestHome(t)
	defer cleanup()
	writeTestServers(t)

	err := configDeleteCmd.RunE(configDeleteCmd, []string{"Test Server 2"})
	if err != nil {
		t.Fatalf("config delete RunE error: %v", err)
	}

	// Verify deletion
	servers, err := config.LoadServers()
	if err != nil {
		t.Fatalf("LoadServers error: %v", err)
	}
	if len(servers.Profiles) != 1 {
		t.Errorf("expected 1 profile after delete, got %d", len(servers.Profiles))
	}
}

func TestConfigDeleteCmd_RunE_NotFound(t *testing.T) {
	cleanup := setupTestHome(t)
	defer cleanup()
	writeTestServers(t)

	err := configDeleteCmd.RunE(configDeleteCmd, []string{"Nonexistent"})
	if err == nil {
		t.Error("config delete with nonexistent profile should return error")
	}
}

func TestConfigDeleteCmd_RunE_DefaultReassignment(t *testing.T) {
	cleanup := setupTestHome(t)
	defer cleanup()
	writeTestServers(t)

	// Delete the default profile
	err := configDeleteCmd.RunE(configDeleteCmd, []string{"Test Server 1"})
	if err != nil {
		t.Fatalf("config delete RunE error: %v", err)
	}

	// Verify Test Server 2 became the new default
	servers, err := config.LoadServers()
	if err != nil {
		t.Fatalf("LoadServers error: %v", err)
	}
	if len(servers.Profiles) != 1 {
		t.Fatalf("expected 1 profile after delete, got %d", len(servers.Profiles))
	}
	if !servers.Profiles[0].Default {
		t.Error("remaining profile should become default after deleting the default")
	}
}

// --- Config duplicate tests ---

func TestConfigDuplicateCmd_RunE(t *testing.T) {
	cleanup := setupTestHome(t)
	defer cleanup()
	writeTestServers(t)

	err := configDuplicateCmd.RunE(configDuplicateCmd, []string{"Test Server 1"})
	if err != nil {
		t.Fatalf("config duplicate RunE error: %v", err)
	}

	// Verify duplication
	servers, err := config.LoadServers()
	if err != nil {
		t.Fatalf("LoadServers error: %v", err)
	}
	if len(servers.Profiles) != 3 {
		t.Fatalf("expected 3 profiles after duplicate, got %d", len(servers.Profiles))
	}

	// Find the duplicated profile
	found := false
	for _, p := range servers.Profiles {
		if p.Name == "Test Server 1 (copy)" {
			found = true
			if p.Default {
				t.Error("duplicated profile should not be default")
			}
			if p.Server.UUID != "test-uuid-1" {
				t.Error("duplicated profile should have same UUID as original")
			}
		}
	}
	if !found {
		t.Error("duplicated profile 'Test Server 1 (copy)' not found")
	}
}

func TestConfigDuplicateCmd_RunE_NotFound(t *testing.T) {
	cleanup := setupTestHome(t)
	defer cleanup()
	writeTestServers(t)

	err := configDuplicateCmd.RunE(configDuplicateCmd, []string{"Nonexistent"})
	if err == nil {
		t.Error("config duplicate with nonexistent profile should return error")
	}
}

// --- Config show tests ---

func TestConfigShowCmd_RunE_NoConfig(t *testing.T) {
	cleanup := setupTestHome(t)
	defer cleanup()
	// No xray config.json written

	err := configShowCmd.RunE(configShowCmd, []string{})
	if err == nil {
		t.Error("config show with no config.json should return error")
	}
}

func TestConfigShowCmd_RunE_WithConfig(t *testing.T) {
	cleanup := setupTestHome(t)
	defer cleanup()

	// Write a minimal config.json
	configJSON := `{"log":{"loglevel":"warning"},"inbounds":[],"outbounds":[]}`
	if err := os.WriteFile(config.XrayConfigPath(), []byte(configJSON), 0644); err != nil {
		t.Fatalf("failed to write config.json: %v", err)
	}

	err := configShowCmd.RunE(configShowCmd, []string{})
	if err != nil {
		t.Fatalf("config show RunE error: %v", err)
	}
}

// --- Config backup tests ---

func TestConfigBackupCmd_RunE(t *testing.T) {
	cleanup := setupTestHome(t)
	defer cleanup()
	writeTestServers(t)
	writeTestSettings(t)
	t.Setenv(passphraseEnvVar, "test-backup-pass")

	// Backup to a specific file
	outPath := filepath.Join(t.TempDir(), "backup.tar.gz")
	err := configBackupCmd.RunE(configBackupCmd, []string{outPath})
	if err != nil {
		t.Fatalf("config backup RunE error: %v", err)
	}

	// Verify backup file exists
	if _, err := os.Stat(outPath); err != nil {
		t.Errorf("backup file should exist at %s", outPath)
	}
}

func TestConfigBackupCmd_RunE_DefaultPath(t *testing.T) {
	cleanup := setupTestHome(t)
	defer cleanup()
	writeTestServers(t)
	t.Setenv(passphraseEnvVar, "test-backup-pass")

	err := configBackupCmd.RunE(configBackupCmd, []string{})
	if err != nil {
		t.Fatalf("config backup RunE with default path error: %v", err)
	}
}

// --- Test command ---

func TestTestCmd_RunE_NoProfiles(t *testing.T) {
	cleanup := setupTestHome(t)
	defer cleanup()
	// No servers written

	// Reset test all flag
	testAllFlag = false
	err := testCmd.RunE(testCmd, []string{})
	if err == nil {
		t.Error("test with no profiles should return error")
	}
}

func TestTestCmd_RunE_ProfileNotFound(t *testing.T) {
	cleanup := setupTestHome(t)
	defer cleanup()
	writeTestServers(t)

	testAllFlag = false
	err := testCmd.RunE(testCmd, []string{"Nonexistent"})
	if err == nil {
		t.Error("test with nonexistent profile should return error")
	}
}

// --- Status command ---

func TestStatusCmd_RunE_Text(t *testing.T) {
	cleanup := setupTestHome(t)
	defer cleanup()
	writeTestServers(t)
	writeTestSettings(t)

	// Reset json flag
	statusJSON = false

	err := statusCmd.RunE(statusCmd, []string{})
	if err != nil {
		t.Fatalf("status RunE error: %v", err)
	}
}

// --- Import command validation ---

func TestImportCmd_RunE_NoArgs_NoSub(t *testing.T) {
	cleanup := setupTestHome(t)
	defer cleanup()

	importSub = ""
	err := importCmd.RunE(importCmd, []string{})
	if err == nil {
		t.Error("import with no args and no --sub should return error")
	}
}

func TestImportCmd_RunE_InvalidURL(t *testing.T) {
	cleanup := setupTestHome(t)
	defer cleanup()
	writeTestServers(t)
	writeTestSettings(t)

	importSub = ""
	importName = ""
	importForce = false
	err := importSingleProfile(importCmd, "not-a-vless-url")
	if err == nil {
		t.Error("import with invalid URL should return error")
	}
}

// --- Export command ---

func TestExportCmd_RunE_NoProfiles(t *testing.T) {
	cleanup := setupTestHome(t)
	defer cleanup()

	exportAll = false
	exportQR = false
	err := exportCmd.RunE(exportCmd, []string{})
	if err == nil {
		t.Error("export with no profiles should return error")
	}
}

func TestExportCmd_RunE_Default(t *testing.T) {
	cleanup := setupTestHome(t)
	defer cleanup()
	writeTestServers(t)

	exportAll = false
	exportQR = false
	err := exportCmd.RunE(exportCmd, []string{})
	if err != nil {
		t.Fatalf("export RunE error: %v", err)
	}
}

func TestExportCmd_RunE_All(t *testing.T) {
	cleanup := setupTestHome(t)
	defer cleanup()
	writeTestServers(t)

	exportAll = true
	exportQR = false
	err := exportCmd.RunE(exportCmd, []string{})
	if err != nil {
		t.Fatalf("export --all RunE error: %v", err)
	}
	exportAll = false
}

func TestExportCmd_RunE_ByName(t *testing.T) {
	cleanup := setupTestHome(t)
	defer cleanup()
	writeTestServers(t)

	exportAll = false
	exportQR = false
	err := exportCmd.RunE(exportCmd, []string{"Test Server 2"})
	if err != nil {
		t.Fatalf("export by name RunE error: %v", err)
	}
}

func TestExportCmd_RunE_NotFound(t *testing.T) {
	cleanup := setupTestHome(t)
	defer cleanup()
	writeTestServers(t)

	exportAll = false
	exportQR = false
	err := exportCmd.RunE(exportCmd, []string{"Nonexistent"})
	if err == nil {
		t.Error("export with nonexistent profile should return error")
	}
}
