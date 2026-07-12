package cmd

import (
	"io"
	"os"
	"strings"
	"testing"

	"github.com/rtxnik/lazyray/internal/config"
)

// captureStdout runs f with os.Stdout redirected to a pipe and returns what was written.
func captureStdout(t *testing.T, f func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	f()
	_ = w.Close()
	os.Stdout = old
	out, _ := io.ReadAll(r)
	return string(out)
}

func TestConfigList_StripsControlFromStoredName(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	servers := &config.ServersConfig{Profiles: []config.Profile{{
		Name:    "e\x1b[31mvil",
		Default: true,
		Server:  config.ServerConfig{Address: "h.example", Port: 443, UUID: "11111111-1111-1111-1111-111111111111", Transport: config.TransportConfig{Network: "tcp"}},
	}}}
	if err := config.SaveServers(servers); err != nil {
		t.Fatal(err)
	}
	out := captureStdout(t, func() {
		if err := configListCmd.RunE(configListCmd, nil); err != nil {
			t.Fatalf("config list: %v", err)
		}
	})
	if strings.ContainsRune(out, 0x1b) {
		t.Fatalf("ESC reached terminal output: %q", out)
	}
}
