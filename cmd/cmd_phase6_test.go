package cmd

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/rtxnik/lazyray/internal/clihint"
	"github.com/rtxnik/lazyray/internal/config"
)

func TestMatchesShortName(t *testing.T) {
	tests := []struct {
		profileName string
		target      string
		want        bool
	}{
		{"Alpha→Beta Cascade", "al", true},
		{"Beta Direct", "be", true},
		{"Beta Direct", "beta direct", true},
		{"Gamma Backup", "eu", false},
		{"Alpha→Beta Cascade", "cascade", false},
		{"", "", true},
		{"abc", "abcdef", false},
	}

	for _, tc := range tests {
		got := matchesShortName(tc.profileName, tc.target)
		if got != tc.want {
			t.Errorf("matchesShortName(%q, %q) = %v, want %v", tc.profileName, tc.target, got, tc.want)
		}
	}
}

func TestTestCmd_HasAllFlag(t *testing.T) {
	flags := testCmd.Flags()
	allFlag := flags.Lookup("all")
	if allFlag == nil {
		t.Error("test command should have --all flag")
	}
	if allFlag != nil && allFlag.DefValue != "false" {
		t.Errorf("--all default = %q, want false", allFlag.DefValue)
	}
}

func TestTunnelCmd_ArgsValidation(t *testing.T) {
	// Tunnel accepts 0 or 1 args
	err := tunnelCmd.Args(tunnelCmd, []string{})
	if err != nil {
		t.Errorf("tunnel with 0 args should succeed: %v", err)
	}

	err = tunnelCmd.Args(tunnelCmd, []string{"server-name"})
	if err != nil {
		t.Errorf("tunnel with 1 arg should succeed: %v", err)
	}
}

func TestImportCmd_HasSubFlag(t *testing.T) {
	flags := importCmd.Flags()
	subFlag := flags.Lookup("sub")
	if subFlag == nil {
		t.Error("import command should have --sub flag")
	}
}

func TestStartCmd_Use(t *testing.T) {
	if startCmd.Use != "start" {
		t.Errorf("startCmd.Use = %q, want 'start'", startCmd.Use)
	}
}

func TestSelfUpdateCmd_Use(t *testing.T) {
	if selfUpdateCmd.Use != "self-update" {
		t.Errorf("selfUpdateCmd.Use = %q, want 'self-update'", selfUpdateCmd.Use)
	}
}

func TestIpCmd_HasJsonFlag(t *testing.T) {
	flags := ipCmd.Flags()
	jsonFlag := flags.Lookup("json")
	if jsonFlag == nil {
		t.Error("ip command should have --json flag")
	}
}

func TestTunnelCloseCmd_Use(t *testing.T) {
	if tunnelCloseCmd.Use != "close" {
		t.Errorf("tunnelCloseCmd.Use = %q, want 'close'", tunnelCloseCmd.Use)
	}
}

func TestTestCmd_Short(t *testing.T) {
	if testCmd.Short == "" {
		t.Error("test command should have a Short description")
	}
}

func TestTestCmd_Long(t *testing.T) {
	if testCmd.Long == "" {
		t.Error("test command should have a Long description")
	}
}

func TestImportCmd_Short(t *testing.T) {
	if importCmd.Short == "" {
		t.Error("import command should have a Short description")
	}
}

func TestStartCmd_Short(t *testing.T) {
	if startCmd.Short == "" {
		t.Error("start command should have a Short description")
	}
}

func TestStopCmd_Short(t *testing.T) {
	if stopCmd.Short == "" {
		t.Error("stop command should have a Short description")
	}
}

func TestRestartCmd_Short(t *testing.T) {
	if restartCmd.Short == "" {
		t.Error("restart command should have a Short description")
	}
}

// --- RunE tests for commands not yet covered ---

func writeSSHServers(t *testing.T) {
	t.Helper()
	serversYAML := `profiles:
  - name: "SSH Server"
    default: true
    server:
      address: "1.2.3.4"
      port: 443
      uuid: "ssh-uuid-1"
    ssh:
      host: "1.2.3.4"
      port: 22
      user: "root"
      keyPath: "~/.ssh/id_ed25519"
      panel:
        port: 28080
        path: "/panel/"
`
	if err := os.WriteFile(config.ServersPath(), []byte(serversYAML), 0600); err != nil {
		t.Fatalf("failed to write servers.yaml: %v", err)
	}
}

// --- Tunnel status ---

func TestTunnelStatus_WithSSHProfiles(t *testing.T) {
	cleanup := setupTestHome(t)
	defer cleanup()
	writeSSHServers(t)

	err := tunnelStatus()
	if err != nil {
		t.Fatalf("tunnelStatus() error = %v", err)
	}
}

func TestTunnelStatus_NoSSHProfiles(t *testing.T) {
	cleanup := setupTestHome(t)
	defer cleanup()
	writeTestServers(t) // These don't have SSH config

	err := tunnelStatus()
	if err != nil {
		t.Fatalf("tunnelStatus() error = %v", err)
	}
}

func TestTunnelStatus_NoServers(t *testing.T) {
	cleanup := setupTestHome(t)
	defer cleanup()
	// No servers file -> empty config with no profiles

	err := tunnelStatus()
	// Should handle gracefully (either error or "no profiles")
	if err != nil {
		t.Logf("tunnelStatus() with no servers returned error (expected): %v", err)
	}
}

// --- Tunnel close ---

func TestTunnelCloseCmd_RunE(t *testing.T) {
	cleanup := setupTestHome(t)
	defer cleanup()

	err := tunnelCloseCmd.RunE(tunnelCloseCmd, []string{})
	if err != nil {
		t.Fatalf("tunnel close RunE error: %v", err)
	}
}

// --- Tunnel connect with profile not found ---

func TestTunnelCmd_RunE_NotFound(t *testing.T) {
	cleanup := setupTestHome(t)
	defer cleanup()
	writeTestServers(t)

	err := tunnelCmd.RunE(tunnelCmd, []string{"nonexistent-profile"})
	if err == nil {
		t.Error("tunnel connect with nonexistent profile should return error")
	}
}

// --- Stop command ---

func TestStopCmd_RunE_NotRunning(t *testing.T) {
	cleanup := setupTestHome(t)
	defer cleanup()

	// When xray is not running, should just print message and return nil
	err := stopCmd.RunE(stopCmd, []string{})
	if err != nil {
		t.Fatalf("stop RunE (not running) error: %v", err)
	}
}

// --- Health command ---

func TestHealthCmd_RunE_Text(t *testing.T) {
	cleanup := setupTestHome(t)
	defer cleanup()
	writeTestServers(t)
	writeTestSettings(t)

	healthJSON = false
	err := healthCmd.RunE(healthCmd, []string{})
	// In a CI/test environment xray is not running, so checks fail and
	// RunE returns a non-nil errHealthChecksFailed. Verify that the error
	// is either nil (all passed) or a diagnostics hint error (checks failed).
	if err != nil {
		var he *clihint.Error
		if !errors.As(err, &he) {
			// Unwrap ExitError to reach the clihint.Error underneath.
			var xe *ExitError
			if errors.As(err, &xe) {
				if !errors.As(xe.Err, &he) {
					t.Fatalf("health RunE returned unexpected error type: %v", err)
				}
			} else {
				t.Fatalf("health RunE returned unexpected error type: %v", err)
			}
		}
	}
}

func TestHealthCmd_RunE_JSON(t *testing.T) {
	cleanup := setupTestHome(t)
	defer cleanup()
	writeTestServers(t)
	writeTestSettings(t)

	healthJSON = true
	defer func() { healthJSON = false }()

	err := healthCmd.RunE(healthCmd, []string{})
	if err != nil {
		t.Fatalf("health --json RunE error: %v", err)
	}
}

func TestHealthCmd_RunE_NoProfiles(t *testing.T) {
	cleanup := setupTestHome(t)
	defer cleanup()

	healthJSON = false
	err := healthCmd.RunE(healthCmd, []string{})
	if err == nil {
		t.Error("health with no profiles should return error")
	}
}

// --- Logs command ---

func TestLogsCmd_RunE_NoLogFile(t *testing.T) {
	cleanup := setupTestHome(t)
	defer cleanup()

	logsError = false
	logsLines = 50
	err := logsCmd.RunE(logsCmd, []string{})
	// Should handle gracefully (no log file = "no log file found")
	if err != nil {
		t.Fatalf("logs RunE (no log file) error: %v", err)
	}
}

func TestLogsCmd_RunE_AccessLog(t *testing.T) {
	cleanup := setupTestHome(t)
	defer cleanup()

	// Write a fake access log
	logPath := config.AccessLogPath()
	if err := os.MkdirAll(filepath.Dir(logPath), 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(logPath, []byte("2024/01/01 12:00:00 test log line 1\n2024/01/01 12:00:01 test log line 2\n"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	logsError = false
	logsLines = 50
	err := logsCmd.RunE(logsCmd, []string{})
	if err != nil {
		t.Fatalf("logs RunE error: %v", err)
	}
}

func TestLogsCmd_RunE_ErrorLog(t *testing.T) {
	cleanup := setupTestHome(t)
	defer cleanup()

	// Write a fake error log
	logPath := config.ErrorLogPath()
	if err := os.MkdirAll(filepath.Dir(logPath), 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(logPath, []byte("error line 1\nerror line 2\n"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	logsError = true
	logsLines = 1
	defer func() { logsError = false; logsLines = 50 }()

	err := logsCmd.RunE(logsCmd, []string{})
	if err != nil {
		t.Fatalf("logs --error RunE error: %v", err)
	}
}

// --- Status command with JSON ---

func TestStatusCmd_RunE_JSON(t *testing.T) {
	cleanup := setupTestHome(t)
	defer cleanup()
	writeTestServers(t)
	writeTestSettings(t)

	statusJSON = true
	defer func() { statusJSON = false }()

	err := statusCmd.RunE(statusCmd, []string{})
	if err != nil {
		t.Fatalf("status --json RunE error: %v", err)
	}
}

func TestStatusCmd_RunE_NoSettings(t *testing.T) {
	cleanup := setupTestHome(t)
	defer cleanup()
	// No settings, no servers — should use defaults

	statusJSON = false
	err := statusCmd.RunE(statusCmd, []string{})
	if err != nil {
		t.Fatalf("status RunE (no settings) error: %v", err)
	}
}

// --- Import single profile with valid VLESS URL ---

func TestImportSingleProfile_ValidURL(t *testing.T) {
	cleanup := setupTestHome(t)
	defer cleanup()
	writeTestSettings(t)
	// Write empty servers so loading works
	if err := os.WriteFile(config.ServersPath(), []byte("profiles: []\n"), 0600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	importName = ""
	importForce = false
	importSub = ""

	vlessURL := "vless://aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee@1.2.3.4:443?type=xhttp&security=reality&sni=example.com&fp=chrome&pbk=TEST_KEY&sid=abc123&path=/test&mode=auto#TestServer"

	err := importSingleProfile(importCmd, vlessURL)
	if err != nil {
		t.Fatalf("importSingleProfile error: %v", err)
	}

	// Verify profile was saved
	servers, err := config.LoadServers()
	if err != nil {
		t.Fatalf("LoadServers error: %v", err)
	}
	if len(servers.Profiles) != 1 {
		t.Fatalf("expected 1 profile, got %d", len(servers.Profiles))
	}
	if servers.Profiles[0].Name != "TestServer" {
		t.Errorf("profile name = %q, want 'TestServer'", servers.Profiles[0].Name)
	}
	if !servers.Profiles[0].Default {
		t.Error("first imported profile should be default")
	}
}

func TestImportSingleProfile_WithCustomName(t *testing.T) {
	cleanup := setupTestHome(t)
	defer cleanup()
	writeTestSettings(t)
	if err := os.WriteFile(config.ServersPath(), []byte("profiles: []\n"), 0600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	importName = "My Custom Name"
	importForce = false
	importSub = ""
	defer func() { importName = "" }()

	vlessURL := "vless://aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee@1.2.3.4:443?type=tcp&security=none#Original"

	err := importSingleProfile(importCmd, vlessURL)
	if err != nil {
		t.Fatalf("importSingleProfile error: %v", err)
	}

	servers, _ := config.LoadServers()
	if servers.Profiles[0].Name != "My Custom Name" {
		t.Errorf("profile name = %q, want 'My Custom Name'", servers.Profiles[0].Name)
	}
}

func TestImportSingleProfile_DuplicateUUID(t *testing.T) {
	cleanup := setupTestHome(t)
	defer cleanup()
	writeTestServers(t) // Already has test-uuid-1
	writeTestSettings(t)

	importName = ""
	importForce = false
	importSub = ""

	vlessURL := "vless://test-uuid-1@5.5.5.5:443?type=tcp&security=none#Duplicate"

	err := importSingleProfile(importCmd, vlessURL)
	if err == nil {
		t.Error("importSingleProfile with duplicate UUID should return error")
	}
}

func TestImportSingleProfile_DuplicateUUID_Force(t *testing.T) {
	cleanup := setupTestHome(t)
	defer cleanup()
	writeTestServers(t)
	writeTestSettings(t)

	importName = ""
	importForce = true
	importSub = ""
	defer func() { importForce = false }()

	vlessURL := "vless://test-uuid-1@5.5.5.5:443?type=tcp&security=none#ForceDuplicate"

	err := importSingleProfile(importCmd, vlessURL)
	if err != nil {
		t.Fatalf("importSingleProfile with --force should succeed: %v", err)
	}
}

// --- Import command RunE with --sub flag pointing to invalid URL ---

func TestImportCmd_RunE_SubFlag(t *testing.T) {
	cleanup := setupTestHome(t)
	defer cleanup()
	writeTestServers(t)
	writeTestSettings(t)

	importSub = "https://invalid.local.example/subscription"
	importName = ""
	defer func() { importSub = "" }()

	err := importCmd.RunE(importCmd, []string{})
	// Should fail because the URL is not reachable
	if err == nil {
		t.Error("import with unreachable --sub URL should return error")
	}
}

// --- Self-update command ---

func TestSelfUpdateCmd_Short(t *testing.T) {
	if selfUpdateCmd.Short == "" {
		t.Error("self-update command should have a Short description")
	}
}

func TestSelfUpdateCmd_Long(t *testing.T) {
	if selfUpdateCmd.Long == "" {
		t.Error("self-update command should have a Long description")
	}
}

// --- IP command ---

func TestIpCmd_Use(t *testing.T) {
	if ipCmd.Use != "ip" {
		t.Errorf("ipCmd.Use = %q, want 'ip'", ipCmd.Use)
	}
}

func TestIpCmd_Short(t *testing.T) {
	if ipCmd.Short == "" {
		t.Error("ip command should have a Short description")
	}
}
