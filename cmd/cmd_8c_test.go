package cmd

import (
	"strings"
	"testing"
)

// --- Proxy command tree tests ---

func TestProxyCmd_HasSubcommands(t *testing.T) {
	commands := proxyCmd.Commands()
	if len(commands) == 0 {
		t.Fatal("proxy command should have subcommands")
	}

	expected := map[string]bool{
		"on":     false,
		"off":    false,
		"status": false,
	}

	for _, cmd := range commands {
		if _, ok := expected[cmd.Name()]; ok {
			expected[cmd.Name()] = true
		}
	}

	for name, found := range expected {
		if !found {
			t.Errorf("missing proxy subcommand: %s", name)
		}
	}
}

func TestProxyCmd_Use(t *testing.T) {
	if proxyCmd.Use != "proxy" {
		t.Errorf("proxyCmd.Use = %q, want %q", proxyCmd.Use, "proxy")
	}
}

func TestProxyOnCmd_Short(t *testing.T) {
	if proxyOnCmd.Short == "" {
		t.Error("proxy on should have Short description")
	}
}

func TestProxyOffCmd_Short(t *testing.T) {
	if proxyOffCmd.Short == "" {
		t.Error("proxy off should have Short description")
	}
}

func TestProxyStatusCmd_Short(t *testing.T) {
	if proxyStatusCmd.Short == "" {
		t.Error("proxy status should have Short description")
	}
}

func TestProxyCmd_Descriptions(t *testing.T) {
	if proxyCmd.Short == "" {
		t.Error("proxy command should have Short description")
	}
	if proxyCmd.Long == "" {
		t.Error("proxy command should have Long description")
	}
}

// --- PAC command tree tests ---

func TestPACCmd_HasSubcommands(t *testing.T) {
	commands := pacCmd.Commands()
	if len(commands) == 0 {
		t.Fatal("pac command should have subcommands")
	}

	expected := map[string]bool{
		"generate": false,
		"serve":    false,
	}

	for _, cmd := range commands {
		if _, ok := expected[cmd.Name()]; ok {
			expected[cmd.Name()] = true
		}
	}

	for name, found := range expected {
		if !found {
			t.Errorf("missing pac subcommand: %s", name)
		}
	}
}

func TestPACCmd_Use(t *testing.T) {
	if pacCmd.Use != "pac" {
		t.Errorf("pacCmd.Use = %q, want %q", pacCmd.Use, "pac")
	}
}

func TestPACServeCmd_Flags(t *testing.T) {
	flags := pacServeCmd.Flags()
	systemFlag := flags.Lookup("system")
	if systemFlag == nil {
		t.Error("pac serve command should have --system flag")
	}
}

// --- Root cmd should include new commands ---

func TestRootCmd_HasProxyCommand(t *testing.T) {
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "proxy" {
			found = true
			break
		}
	}
	if !found {
		t.Error("root command should have proxy subcommand")
	}
}

func TestRootCmd_HasPACCommand(t *testing.T) {
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "pac" {
			found = true
			break
		}
	}
	if !found {
		t.Error("root command should have pac subcommand")
	}
}

func TestRootCmd_HasSpeedtestCommand(t *testing.T) {
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "speedtest" {
			found = true
			break
		}
	}
	if !found {
		t.Error("root command should have speedtest subcommand")
	}
}

func TestRootCmd_HasStatsCommand(t *testing.T) {
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "stats" {
			found = true
			break
		}
	}
	if !found {
		t.Error("root command should have stats subcommand")
	}
}

func TestRootCmd_HasSelfUpdateCommand(t *testing.T) {
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "self-update" {
			found = true
			break
		}
	}
	if !found {
		t.Error("root command should have self-update subcommand")
	}
}

// --- Command Use string patterns ---

func TestNewCommandUseStrings(t *testing.T) {
	tests := []struct {
		name string
		use  string
		want string
	}{
		{"proxy", proxyCmd.Use, "proxy"},
		{"proxy on", proxyOnCmd.Use, "on"},
		{"proxy off", proxyOffCmd.Use, "off"},
		{"proxy status", proxyStatusCmd.Use, "status"},
		{"pac", pacCmd.Use, "pac"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			parts := strings.Fields(tc.use)
			if len(parts) == 0 || parts[0] != tc.want {
				t.Errorf("cmd.Use = %q, should start with %q", tc.use, tc.want)
			}
		})
	}
}

// --- Speedtest command ---

func TestSpeedtestCmd_Short(t *testing.T) {
	if speedtestCmd.Short == "" {
		t.Error("speedtest command should have Short description")
	}
}

// --- Stats command ---

func TestStatsCmd_Short(t *testing.T) {
	if statsCmd.Short == "" {
		t.Error("stats command should have Short description")
	}
}

// --- Self-update command ---

func TestSelfUpdateCmd_HasRunE(t *testing.T) {
	if selfUpdateCmd.RunE == nil {
		t.Error("self-update command should have RunE handler")
	}
}

// --- Start command ---

func TestStartCmd_HasRunE(t *testing.T) {
	if startCmd.RunE == nil {
		t.Error("start command should have RunE handler")
	}
}

// --- IP command flags ---

func TestIPCmd_Flags(t *testing.T) {
	flags := ipCmd.Flags()
	jsonFlag := flags.Lookup("json")
	if jsonFlag == nil {
		t.Error("ip command should have --json flag")
	}
}
