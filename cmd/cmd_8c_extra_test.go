package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/rtxnik/lazyray/internal/config"
	"github.com/rtxnik/lazyray/internal/core"
)

// --- PAC generate command ---

func TestPACGenerateCmd_Use(t *testing.T) {
	if pacGenerateCmd.Use != "generate" {
		t.Errorf("pacGenerateCmd.Use = %q, want generate", pacGenerateCmd.Use)
	}
}

func TestPACGenerateCmd_Short(t *testing.T) {
	if pacGenerateCmd.Short == "" {
		t.Error("pac generate should have Short description")
	}
}

func TestPACGenerateCmd_Flags(t *testing.T) {
	flags := pacGenerateCmd.Flags()
	output := flags.Lookup("output")
	if output == nil {
		t.Error("pac generate should have --output flag")
	}
}

func TestPACServeCmd_Use(t *testing.T) {
	if pacServeCmd.Use != "serve" {
		t.Errorf("pacServeCmd.Use = %q, want serve", pacServeCmd.Use)
	}
}

func TestPACServeCmd_Short(t *testing.T) {
	if pacServeCmd.Short == "" {
		t.Error("pac serve should have Short description")
	}
}

func TestPACServeCmd_PortFlag(t *testing.T) {
	flags := pacServeCmd.Flags()
	port := flags.Lookup("port")
	if port == nil {
		t.Error("pac serve should have --port flag")
	}
	if port != nil && port.DefValue != "10810" {
		t.Errorf("pac serve --port default = %q, want 10810", port.DefValue)
	}
}

// --- Proxy command RunE ---

func TestProxyOnCmd_HasRunE(t *testing.T) {
	if proxyOnCmd.RunE == nil {
		t.Error("proxy on should have RunE handler")
	}
}

func TestProxyOffCmd_HasRunE(t *testing.T) {
	if proxyOffCmd.RunE == nil {
		t.Error("proxy off should have RunE handler")
	}
}

func TestProxyStatusCmd_HasRunE(t *testing.T) {
	if proxyStatusCmd.RunE == nil {
		t.Error("proxy status should have RunE handler")
	}
}

// --- Speedtest command ---

func TestSpeedtestCmd_Use(t *testing.T) {
	if speedtestCmd.Use != "speedtest" {
		t.Errorf("speedtestCmd.Use = %q, want speedtest", speedtestCmd.Use)
	}
}

func TestSpeedtestCmd_HasRunE(t *testing.T) {
	if speedtestCmd.RunE == nil {
		t.Error("speedtest command should have RunE handler")
	}
}

// --- Stats command ---

func TestStatsCmd_Use(t *testing.T) {
	if statsCmd.Use != "stats" {
		t.Errorf("statsCmd.Use = %q, want stats", statsCmd.Use)
	}
}

func TestStatsCmd_HasRunE(t *testing.T) {
	if statsCmd.RunE == nil {
		t.Error("stats command should have RunE handler")
	}
}

// --- Health command ---

func TestHealthCmd_HasRunE(t *testing.T) {
	if healthCmd.RunE == nil {
		t.Error("health command should have RunE handler")
	}
}

// --- Stop command ---

func TestStopCmd_HasRunE(t *testing.T) {
	if stopCmd.RunE == nil {
		t.Error("stop command should have RunE handler")
	}
}

// --- Restart command ---

func TestRestartCmd_HasRunE(t *testing.T) {
	if restartCmd.RunE == nil {
		t.Error("restart command should have RunE handler")
	}
}

// --- All new commands registered ---

func TestRootCmd_AllCommandsRegistered(t *testing.T) {
	expected := []string{
		"status", "start", "stop", "restart", "health",
		"import", "export", "config", "update", "tunnel",
		"logs", "ip", "service", "proxy", "pac",
		"speedtest", "stats", "self-update", "test",
	}

	cmds := rootCmd.Commands()
	cmdMap := make(map[string]bool)
	for _, c := range cmds {
		cmdMap[c.Name()] = true
	}

	for _, name := range expected {
		if !cmdMap[name] {
			t.Errorf("missing registered command: %s", name)
		}
	}
}

// --- PAC Long descriptions ---

func TestPACCmd_Long(t *testing.T) {
	if pacCmd.Long == "" {
		t.Error("pac command should have Long description")
	}
}

func TestPACServeCmd_Long(t *testing.T) {
	if pacServeCmd.Long == "" {
		t.Error("pac serve should have Long description")
	}
}

// --- Proxy Long descriptions ---

func TestProxyOnCmd_Long(t *testing.T) {
	if proxyOnCmd.Long == "" {
		t.Error("proxy on should have Long description")
	}
}

func TestProxyOffCmd_Long(t *testing.T) {
	if proxyOffCmd.Long == "" {
		t.Error("proxy off should have Long description")
	}
}

// --- Config restore tests ---

func TestConfigRestoreCmd_RunE(t *testing.T) {
	cleanup := setupTestHome(t)
	defer cleanup()
	writeTestServers(t)
	writeTestSettings(t)
	t.Setenv(passphraseEnvVar, "test-backup-pass")

	// Create backup first
	backupPath := filepath.Join(t.TempDir(), "backup.tar.gz")
	err := configBackupCmd.RunE(configBackupCmd, []string{backupPath})
	if err != nil {
		t.Fatalf("config backup error: %v", err)
	}

	// Delete config files
	os.Remove(config.ServersPath())
	os.Remove(config.SettingsPath())

	// Restore from backup
	err = configRestoreCmd.RunE(configRestoreCmd, []string{backupPath})
	if err != nil {
		t.Fatalf("config restore error: %v", err)
	}

	// Verify files were restored
	if _, err := os.Stat(config.ServersPath()); err != nil {
		t.Error("servers.yaml should be restored")
	}
	if _, err := os.Stat(config.SettingsPath()); err != nil {
		t.Error("settings should be restored")
	}
}

func TestConfigRestoreCmd_RunE_InvalidFile(t *testing.T) {
	cleanup := setupTestHome(t)
	defer cleanup()

	err := configRestoreCmd.RunE(configRestoreCmd, []string{"/nonexistent/backup.tar.gz"})
	if err == nil {
		t.Error("restore with nonexistent file should return error")
	}
}

func TestConfigRestoreCmd_RunE_InvalidGzip(t *testing.T) {
	cleanup := setupTestHome(t)
	defer cleanup()

	badFile := filepath.Join(t.TempDir(), "bad.tar.gz")
	os.WriteFile(badFile, []byte("not a gzip file"), 0644)

	err := configRestoreCmd.RunE(configRestoreCmd, []string{badFile})
	if err == nil {
		t.Error("restore with invalid gzip should return error")
	}
}

// --- Import encrypted tests ---

func TestImportEncrypted_NoPassword(t *testing.T) {
	cleanup := setupTestHome(t)
	defer cleanup()

	importDecrypt = false
	fakeTerminal(t, false)
	clearPassphraseEnv(t)
	err := importEncrypted(importCmd, "LZRENC1:somedata")
	if err == nil {
		t.Error("importEncrypted with no password should return error")
	}
}

func TestImportEncrypted_InvalidData(t *testing.T) {
	cleanup := setupTestHome(t)
	defer cleanup()

	importDecrypt = true
	t.Setenv("LAZYRAY_PASSPHRASE", "mypassword")
	err := importEncrypted(importCmd, "LZRENC1:invaliddata")
	importDecrypt = false
	if err == nil {
		t.Error("importEncrypted with invalid data should return error")
	}
}

func TestImportEncrypted_ValidData(t *testing.T) {
	cleanup := setupTestHome(t)
	defer cleanup()
	writeTestServers(t)
	writeTestSettings(t)

	// Create encrypted export from existing profiles
	servers, _ := config.LoadServers()
	encrypted, err := core.ExportEncrypted(servers.Profiles, "testpass123")
	if err != nil {
		t.Fatalf("ExportEncrypted error: %v", err)
	}

	// Clear profiles and re-import
	servers.Profiles = nil
	config.SaveServers(servers)

	importDecrypt = true
	t.Setenv("LAZYRAY_PASSPHRASE", "testpass123")
	importForce = true
	err = importEncrypted(importCmd, encrypted)
	importDecrypt = false
	importForce = false
	if err != nil {
		t.Fatalf("importEncrypted error: %v", err)
	}

	// Verify profiles were imported
	servers, _ = config.LoadServers()
	if len(servers.Profiles) == 0 {
		t.Error("should have imported profiles")
	}
}

// --- Export encrypted tests ---

func TestExportCmd_RunE_Encrypted(t *testing.T) {
	cleanup := setupTestHome(t)
	defer cleanup()
	writeTestServers(t)

	exportAll = false
	exportQR = false
	exportEncrypt = true
	t.Setenv("LAZYRAY_PASSPHRASE", "mypassword")
	err := exportCmd.RunE(exportCmd, []string{})
	exportEncrypt = false
	if err != nil {
		t.Fatalf("export --encrypt RunE error: %v", err)
	}
}

// --- Export QR tests ---

func TestExportCmd_RunE_QR(t *testing.T) {
	cleanup := setupTestHome(t)
	defer cleanup()
	writeTestServers(t)

	exportAll = false
	exportQR = true
	exportEncrypt = false
	err := exportCmd.RunE(exportCmd, []string{})
	exportQR = false
	if err != nil {
		t.Fatalf("export --qr RunE error: %v", err)
	}
}

// --- Test all profiles ---

func TestTestAllProfiles_8C(t *testing.T) {
	cleanup := setupTestHome(t)
	defer cleanup()
	writeTestServers(t)

	servers, err := config.LoadServers()
	if err != nil {
		t.Fatalf("LoadServers error: %v", err)
	}

	// testAllProfiles will try connecting and fail (no real server)
	// but it should exercise the sorting/formatting code
	err = testAllProfiles(servers)
	// Error is OK since servers are unreachable
	_ = err
}

func TestTestAllProfiles_Empty(t *testing.T) {
	servers := &config.ServersConfig{}
	err := testAllProfiles(servers)
	if err == nil {
		t.Error("testAllProfiles with no profiles should return error")
	}
}

func TestTestCmd_RunE_AllFlag(t *testing.T) {
	cleanup := setupTestHome(t)
	defer cleanup()
	writeTestServers(t)

	testAllFlag = true
	err := testCmd.RunE(testCmd, []string{})
	testAllFlag = false
	// May error due to unreachable servers, but exercises the code path
	_ = err
}

// --- Test cmd default profile ---

func TestTestCmd_RunE_DefaultProfile(t *testing.T) {
	cleanup := setupTestHome(t)
	defer cleanup()
	writeTestServers(t)

	testAllFlag = false
	err := testCmd.RunE(testCmd, []string{})
	// Will fail trying to connect but exercises the code path
	_ = err
}

func TestTestCmd_RunE_NamedProfile(t *testing.T) {
	cleanup := setupTestHome(t)
	defer cleanup()
	writeTestServers(t)

	testAllFlag = false
	err := testCmd.RunE(testCmd, []string{"Test Server 1"})
	_ = err
}

// --- Status cmd with no settings ---

func TestStatusCmd_RunE_NoServers(t *testing.T) {
	cleanup := setupTestHome(t)
	defer cleanup()
	// No servers or settings written

	statusJSON = false
	err := statusCmd.RunE(statusCmd, []string{})
	if err != nil {
		t.Fatalf("status RunE with no servers error: %v", err)
	}
}

// --- Import cmd with valid VLESS URL ---

func TestImportCmd_RunE_VLESS(t *testing.T) {
	cleanup := setupTestHome(t)
	defer cleanup()
	writeTestSettings(t)

	importSub = ""
	importName = ""
	importForce = false
	importDecrypt = false

	url := "vless://test-uuid@1.2.3.4:443?type=tcp&security=none#TestProfile"
	err := importCmd.RunE(importCmd, []string{url})
	if err != nil {
		t.Fatalf("import VLESS URL error: %v", err)
	}

	// Verify profile was imported
	servers, _ := config.LoadServers()
	if len(servers.Profiles) == 0 {
		t.Error("should have imported profile")
	}
}

// --- Import cmd with --decrypt and no encrypted data ---

func TestImportCmd_RunE_DecryptNonEncrypted(t *testing.T) {
	cleanup := setupTestHome(t)
	defer cleanup()

	importSub = ""
	importDecrypt = true
	t.Setenv("LAZYRAY_PASSPHRASE", "password")
	err := importCmd.RunE(importCmd, []string{"vless://test@1.2.3.4:443"})
	importDecrypt = false
	if err == nil {
		t.Error("import --decrypt with non-encrypted data should return error")
	}
}

// --- Config rename (switch + old name verification) ---

func TestConfigSwitchCmd_RunE_8C(t *testing.T) {
	cleanup := setupTestHome(t)
	defer cleanup()
	writeTestServers(t)
	writeTestSettings(t)

	err := configSwitchCmd.RunE(configSwitchCmd, []string{"Test Server 2"})
	if err != nil {
		t.Fatalf("switch error: %v", err)
	}

	// Verify new default
	servers, _ := config.LoadServers()
	def := servers.DefaultProfile()
	if def == nil {
		t.Fatal("should have a default profile")
	}
	if def.Name != "Test Server 2" {
		t.Errorf("default = %q, want Test Server 2", def.Name)
	}
}

// --- Root Execute ---

func TestRootCmd_Execute_8C(t *testing.T) {
	cleanup := setupTestHome(t)
	defer cleanup()

	// Set version for root cmd
	SetVersion("test")

	// Exercise Execute with help flag (won't error)
	rootCmd.SetArgs([]string{"--help"})
	err := rootCmd.Execute()
	rootCmd.SetArgs(nil)
	if err != nil {
		t.Fatalf("root --help Execute error: %v", err)
	}
}

func TestRootCmd_Execute_UnknownCmd(t *testing.T) {
	cleanup := setupTestHome(t)
	defer cleanup()

	rootCmd.SetArgs([]string{"nonexistent-cmd-xyz"})
	err := rootCmd.Execute()
	rootCmd.SetArgs(nil)
	if err == nil {
		t.Error("root with unknown command should return error")
	}
}
