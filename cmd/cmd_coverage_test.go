package cmd

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/rtxnik/lazyray/internal/status"
)

// --- Command tree structure tests ---

func TestUpdateCmd_HasSubcommands(t *testing.T) {
	commands := updateCmd.Commands()
	if len(commands) == 0 {
		t.Fatal("update command should have subcommands")
	}

	expected := map[string]bool{
		"check": false,
		"apply": false,
	}

	for _, cmd := range commands {
		if _, ok := expected[cmd.Name()]; ok {
			expected[cmd.Name()] = true
		}
	}

	for name, found := range expected {
		if !found {
			t.Errorf("missing update subcommand: %s", name)
		}
	}
}

func TestServiceCmd_HasSubcommands(t *testing.T) {
	commands := serviceCmd.Commands()
	if len(commands) == 0 {
		t.Fatal("service command should have subcommands")
	}

	expected := map[string]bool{
		"install":   false,
		"uninstall": false,
		"status":    false,
	}

	for _, cmd := range commands {
		if _, ok := expected[cmd.Name()]; ok {
			expected[cmd.Name()] = true
		}
	}

	for name, found := range expected {
		if !found {
			t.Errorf("missing service subcommand: %s", name)
		}
	}
}

func TestTunnelCmd_HasSubcommands(t *testing.T) {
	commands := tunnelCmd.Commands()
	if len(commands) == 0 {
		t.Fatal("tunnel command should have subcommands")
	}

	found := false
	for _, cmd := range commands {
		if cmd.Name() == "close" {
			found = true
		}
	}
	if !found {
		t.Error("tunnel should have 'close' subcommand")
	}
}

// --- Export command flags ---

func TestExportCmd_Flags(t *testing.T) {
	flags := exportCmd.Flags()

	allFlag := flags.Lookup("all")
	if allFlag == nil {
		t.Error("export command should have --all flag")
	}
	if allFlag != nil && allFlag.DefValue != "false" {
		t.Errorf("--all default = %q, want false", allFlag.DefValue)
	}

	qrFlag := flags.Lookup("qr")
	if qrFlag == nil {
		t.Error("export command should have --qr flag")
	}
	if qrFlag != nil && qrFlag.DefValue != "false" {
		t.Errorf("--qr default = %q, want false", qrFlag.DefValue)
	}
}

func TestExportCmd_Args(t *testing.T) {
	// Export accepts 0 or 1 args
	err := exportCmd.Args(exportCmd, []string{})
	if err != nil {
		t.Errorf("export with 0 args should succeed: %v", err)
	}

	err = exportCmd.Args(exportCmd, []string{"name"})
	if err != nil {
		t.Errorf("export with 1 arg should succeed: %v", err)
	}

	err = exportCmd.Args(exportCmd, []string{"a", "b"})
	if err == nil {
		t.Error("export with 2 args should fail")
	}
}

// --- Status command flags ---

func TestStatusCmd_Flags(t *testing.T) {
	flags := statusCmd.Flags()

	jsonFlag := flags.Lookup("json")
	if jsonFlag == nil {
		t.Error("status command should have --json flag")
	}
}

// --- Health command flags ---

func TestHealthCmd_Flags(t *testing.T) {
	flags := healthCmd.Flags()

	jsonFlag := flags.Lookup("json")
	if jsonFlag == nil {
		t.Error("health command should have --json flag")
	}
}

// --- Logs command flags ---

func TestLogsCmd_Flags(t *testing.T) {
	flags := logsCmd.Flags()

	errorFlag := flags.Lookup("error")
	if errorFlag == nil {
		t.Error("logs command should have --error flag")
	}

	linesFlag := flags.Lookup("lines")
	if linesFlag == nil {
		t.Error("logs command should have --lines flag")
	}
}

// --- Config list --json flag ---

func TestConfigListCmd_Flags(t *testing.T) {
	flags := configListCmd.Flags()

	jsonFlag := flags.Lookup("json")
	if jsonFlag == nil {
		t.Error("config list command should have --json flag")
	}
}

// --- Command Use strings ---

func TestCommandUseStrings(t *testing.T) {
	tests := []struct {
		name string
		use  string
		want string
	}{
		{"status", statusCmd.Use, "status"},
		{"start", startCmd.Use, "start"},
		{"stop", stopCmd.Use, "stop"},
		{"restart", restartCmd.Use, "restart"},
		{"health", healthCmd.Use, "health"},
		{"config", configCmd.Use, "config"},
		{"export", exportCmd.Use, "export"},
		{"logs", logsCmd.Use, "logs"},
		{"tunnel", tunnelCmd.Use, "tunnel"},
		{"update", updateCmd.Use, "update"},
		{"service", serviceCmd.Use, "service"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Use should start with the expected command name
			parts := strings.Fields(tc.use)
			if len(parts) == 0 || parts[0] != tc.want {
				t.Errorf("cmd.Use = %q, should start with %q", tc.use, tc.want)
			}
		})
	}
}

// --- Command Short descriptions ---

func TestCommandShortDescriptions(t *testing.T) {
	commands := []struct {
		name  string
		short string
	}{
		{"root", rootCmd.Short},
		{"status", statusCmd.Short},
		{"start", startCmd.Short},
		{"stop", stopCmd.Short},
		{"restart", restartCmd.Short},
		{"health", healthCmd.Short},
		{"config", configCmd.Short},
		{"import", importCmd.Short},
		{"export", exportCmd.Short},
		{"logs", logsCmd.Short},
		{"tunnel", tunnelCmd.Short},
		{"update", updateCmd.Short},
		{"service", serviceCmd.Short},
	}

	for _, c := range commands {
		t.Run(c.name, func(t *testing.T) {
			if c.short == "" {
				t.Errorf("%s command should have a Short description", c.name)
			}
		})
	}
}

// --- Config subcommand Short descriptions ---

func TestConfigSubcommandDescriptions(t *testing.T) {
	commands := configCmd.Commands()
	for _, cmd := range commands {
		if cmd.Short == "" {
			t.Errorf("config subcommand %q should have a Short description", cmd.Name())
		}
	}
}

// --- Import command validation ---

func TestImportCmd_Args_MaxOne(t *testing.T) {
	// Import accepts 0 or 1 args
	err := importCmd.Args(importCmd, []string{})
	if err != nil {
		t.Errorf("import with 0 args should succeed: %v", err)
	}

	err = importCmd.Args(importCmd, []string{"vless://..."})
	if err != nil {
		t.Errorf("import with 1 arg should succeed: %v", err)
	}

	err = importCmd.Args(importCmd, []string{"a", "b"})
	if err == nil {
		t.Error("import with 2 args should fail")
	}
}

// --- Snapshot comprehensive test ---

func TestStatusOutput_AllFields(t *testing.T) {
	out := status.Snapshot{
		Running:       true,
		PID:           9999,
		Uptime:        "2h 30m",
		UptimeSeconds: 9000,
		SocksOK:       true,
		HTTPOK:        true,
		SocksAddr:     "0.0.0.0:1080",
		HTTPAddr:      "0.0.0.0:1081",
		Profile:       "my-server",
		XrayVersion:   "v1.8.24",
		ExitIP:        "1.2.3.4",
		PIDFile:       "/tmp/xray.pid",
		ConfigPath:    "/tmp/config.json",
	}

	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	// Verify JSON contains all expected fields
	jsonStr := string(data)
	for _, field := range []string{"running", "pid", "uptime", "uptimeSeconds", "socksOK", "httpOK", "socksAddr", "httpAddr", "profile", "xrayVersion", "exitIP", "pidFile", "configPath"} {
		if !strings.Contains(jsonStr, field) {
			t.Errorf("JSON output missing field: %s", field)
		}
	}
}

// --- Version setter/getter ---

func TestSetVersion_Multiple(t *testing.T) {
	versions := []string{"v0.1.0", "v0.5.0", "v1.0.0", "dev"}
	for _, v := range versions {
		SetVersion(v)
		if got := Version(); got != v {
			t.Errorf("after SetVersion(%q), Version() = %q", v, got)
		}
		if rootCmd.Version != v {
			t.Errorf("after SetVersion(%q), rootCmd.Version = %q", v, rootCmd.Version)
		}
	}
}

// --- Config backup/restore command args ---

func TestConfigBackupCmd_Args(t *testing.T) {
	// Backup accepts 0 or 1 args
	err := configBackupCmd.Args(configBackupCmd, []string{})
	if err != nil {
		t.Errorf("backup with 0 args should succeed: %v", err)
	}

	err = configBackupCmd.Args(configBackupCmd, []string{"file.tar.gz"})
	if err != nil {
		t.Errorf("backup with 1 arg should succeed: %v", err)
	}

	err = configBackupCmd.Args(configBackupCmd, []string{"a", "b"})
	if err == nil {
		t.Error("backup with 2 args should fail")
	}
}

func TestConfigRestoreCmd_Args(t *testing.T) {
	// Restore requires exactly 1 arg
	err := configRestoreCmd.Args(configRestoreCmd, []string{})
	if err == nil {
		t.Error("restore with 0 args should fail")
	}

	err = configRestoreCmd.Args(configRestoreCmd, []string{"file.tar.gz"})
	if err != nil {
		t.Errorf("restore with 1 arg should succeed: %v", err)
	}
}

// --- Root command ---

func TestRootCmd_Long(t *testing.T) {
	if rootCmd.Long == "" {
		t.Error("root command should have a Long description")
	}
}

func TestRootCmd_HasVersion(t *testing.T) {
	if rootCmd.Version == "" {
		t.Error("rootCmd.Version should not be empty")
	}
}

// --- Subcommand count ---

func TestRootCmd_MinimumSubcommands(t *testing.T) {
	commands := rootCmd.Commands()
	// Should have at least 10 subcommands
	if len(commands) < 10 {
		t.Errorf("root command has %d subcommands, want at least 10", len(commands))
	}
}

func TestConfigCmd_MinimumSubcommands(t *testing.T) {
	commands := configCmd.Commands()
	// Should have at least 7 subcommands
	if len(commands) < 7 {
		t.Errorf("config command has %d subcommands, want at least 7", len(commands))
	}
}
