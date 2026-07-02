package cmd

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/rtxnik/lazyray/internal/status"
)

func TestStatusOutput_JSON(t *testing.T) {
	tmpDir := t.TempDir()
	out := status.Snapshot{
		Running:       true,
		PID:           1234,
		Uptime:        "1h 30m",
		UptimeSeconds: 5400,
		SocksOK:       true,
		HTTPOK:        true,
		SocksAddr:     "127.0.0.1:10808",
		HTTPAddr:      "127.0.0.1:10809",
		Profile:       "test-server",
		XrayVersion:   "1.8.7",
		ExitIP:        "5.6.7.8",
		PIDFile:       filepath.Join(tmpDir, "xray.pid"),
		ConfigPath:    filepath.Join(tmpDir, "config.json"),
	}

	data, err := json.Marshal(out)
	if err != nil {
		t.Fatalf("failed to marshal Snapshot: %v", err)
	}

	var parsed status.Snapshot
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal Snapshot: %v", err)
	}

	if parsed.Running != out.Running {
		t.Errorf("Running = %v, want %v", parsed.Running, out.Running)
	}
	if parsed.PID != out.PID {
		t.Errorf("PID = %d, want %d", parsed.PID, out.PID)
	}
	if parsed.Profile != out.Profile {
		t.Errorf("Profile = %q, want %q", parsed.Profile, out.Profile)
	}
	if parsed.ExitIP != out.ExitIP {
		t.Errorf("ExitIP = %q, want %q", parsed.ExitIP, out.ExitIP)
	}
}

func TestStatusOutput_JSON_OmitsEmptyExitIP(t *testing.T) {
	out := status.Snapshot{
		Running: false,
		Profile: "test",
		ExitIP:  "",
	}

	data, err := json.Marshal(out)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	// ExitIP has omitempty, so it should not appear
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatal(err)
	}
	if _, exists := raw["exitIP"]; exists {
		t.Error("exitIP should be omitted when empty")
	}
}

func TestVersion(t *testing.T) {
	SetVersion("v0.5.0-test")
	if got := Version(); got != "v0.5.0-test" {
		t.Errorf("Version() = %q, want %q", got, "v0.5.0-test")
	}
	// Reset
	SetVersion("dev")
}

func TestRootCmd_HasSubcommands(t *testing.T) {
	commands := rootCmd.Commands()
	if len(commands) == 0 {
		t.Fatal("root command should have subcommands")
	}

	// Check key subcommands exist
	expected := map[string]bool{
		"status":  false,
		"start":   false,
		"stop":    false,
		"restart": false,
		"health":  false,
		"import":  false,
		"export":  false,
		"config":  false,
		"update":  false,
		"tunnel":  false,
		"logs":    false,
		"ip":      false,
		"service": false,
	}

	for _, cmd := range commands {
		if _, ok := expected[cmd.Name()]; ok {
			expected[cmd.Name()] = true
		}
	}

	for name, found := range expected {
		if !found {
			t.Errorf("missing subcommand: %s", name)
		}
	}
}
