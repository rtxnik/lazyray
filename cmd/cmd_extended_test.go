package cmd

import (
	"testing"
)

func TestRootCmd_VersionFlag(t *testing.T) {
	SetVersion("v0.6.0-test")
	if rootCmd.Version != "v0.6.0-test" {
		t.Errorf("rootCmd.Version = %q, want %q", rootCmd.Version, "v0.6.0-test")
	}
	SetVersion("dev")
}

func TestConfigCmd_HasSubcommands(t *testing.T) {
	commands := configCmd.Commands()
	if len(commands) == 0 {
		t.Fatal("config command should have subcommands")
	}

	expected := map[string]bool{
		"show":      false,
		"list":      false,
		"switch":    false,
		"edit":      false,
		"delete":    false,
		"backup":    false,
		"restore":   false,
		"duplicate": false,
	}

	for _, cmd := range commands {
		if _, ok := expected[cmd.Name()]; ok {
			expected[cmd.Name()] = true
		}
	}

	for name, found := range expected {
		if !found {
			t.Errorf("missing config subcommand: %s", name)
		}
	}
}

func TestImportCmd_Flags(t *testing.T) {
	// Verify import command has expected flags
	flags := importCmd.Flags()

	nameFlag := flags.Lookup("name")
	if nameFlag == nil {
		t.Error("import command should have --name flag")
	}

	forceFlag := flags.Lookup("force")
	if forceFlag == nil {
		t.Error("import command should have --force flag")
	}

	subFlag := flags.Lookup("sub")
	if subFlag == nil {
		t.Error("import command should have --sub flag")
	}
}

func TestTestCmd_Flags(t *testing.T) {
	flags := testCmd.Flags()

	allFlag := flags.Lookup("all")
	if allFlag == nil {
		t.Error("test command should have --all flag")
	}
}

func TestRootCmd_Use(t *testing.T) {
	if rootCmd.Use != "lzr" {
		t.Errorf("rootCmd.Use = %q, want %q", rootCmd.Use, "lzr")
	}
}

func TestImportCmd_Use(t *testing.T) {
	if importCmd.Use != "import [url]" {
		t.Errorf("importCmd.Use = %q, want expected value", importCmd.Use)
	}
}

func TestConfigDuplicateCmd_Args(t *testing.T) {
	// Duplicate requires exactly 1 arg
	err := configDuplicateCmd.Args(configDuplicateCmd, []string{})
	if err == nil {
		t.Error("duplicate command should require exactly 1 arg")
	}

	err = configDuplicateCmd.Args(configDuplicateCmd, []string{"name"})
	if err != nil {
		t.Errorf("duplicate with 1 arg should succeed: %v", err)
	}
}

func TestConfigSwitchCmd_Args(t *testing.T) {
	err := configSwitchCmd.Args(configSwitchCmd, []string{})
	if err == nil {
		t.Error("switch command should require exactly 1 arg")
	}

	err = configSwitchCmd.Args(configSwitchCmd, []string{"name"})
	if err != nil {
		t.Errorf("switch with 1 arg should succeed: %v", err)
	}
}

func TestConfigDeleteCmd_Args(t *testing.T) {
	err := configDeleteCmd.Args(configDeleteCmd, []string{})
	if err == nil {
		t.Error("delete command should require exactly 1 arg")
	}
}

func TestTestCmd_Args(t *testing.T) {
	// Test accepts 0 or 1 args
	err := testCmd.Args(testCmd, []string{})
	if err != nil {
		t.Errorf("test with 0 args should succeed: %v", err)
	}

	err = testCmd.Args(testCmd, []string{"name"})
	if err != nil {
		t.Errorf("test with 1 arg should succeed: %v", err)
	}

	err = testCmd.Args(testCmd, []string{"a", "b"})
	if err == nil {
		t.Error("test with 2 args should fail")
	}
}
