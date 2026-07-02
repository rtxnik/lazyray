package doctor

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/rtxnik/lazyray/internal/config"
)

func TestCheckServersParse(t *testing.T) {
	valid := &config.ServersConfig{Profiles: []config.Profile{
		{Name: "p", Default: true, Server: config.ServerConfig{Address: "1.2.3.4", Port: 443}},
	}}
	tests := []struct {
		name    string
		servers *config.ServersConfig
		loadErr error
		want    Severity
	}{
		{"valid", valid, nil, SeverityOK},
		{"parse-error", nil, errors.New("parsing servers config: bad yaml"), SeverityFail},
		{"no-default-profile", &config.ServersConfig{}, nil, SeverityFail},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			env := &Env{LoadServers: func() (*config.ServersConfig, error) { return tc.servers, tc.loadErr }}
			got := checkServersParse(context.Background(), env)
			if got.Severity != tc.want {
				t.Errorf("severity = %v, want %v (detail=%q)", got.Severity, tc.want, got.Detail)
			}
		})
	}
}

func TestCheckPorts(t *testing.T) {
	mk := func(socks, http int) *config.Settings {
		s := config.DefaultSettings()
		s.Local.SocksPort = socks
		s.Local.HTTPPort = http
		return s
	}
	tests := []struct {
		name  string
		socks int
		http  int
		want  Severity
	}{
		{"ok", 10808, 10809, SeverityOK},
		{"same-port", 10808, 10808, SeverityFail},
		{"socks-out-of-range", 0, 10809, SeverityFail},
		{"http-out-of-range", 10808, 70000, SeverityFail},
		{"privileged-warn", 80, 81, SeverityWarn},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			env := &Env{LoadSettings: func() (*config.Settings, error) { return mk(tc.socks, tc.http), nil }}
			got := checkPorts(context.Background(), env)
			if got.Severity != tc.want {
				t.Errorf("severity = %v, want %v (detail=%q)", got.Severity, tc.want, got.Detail)
			}
		})
	}
}

func TestCheckFilePerms(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX permission semantics are not enforced on Windows")
	}
	state := "/data/state.json"
	cfg := "/data/config.json"
	tests := []struct {
		name  string
		files map[string]os.FileInfo
		want  Severity
	}{
		{"both-0600", map[string]os.FileInfo{
			state: fakeFileInfo{name: "state.json", mode: 0600},
			cfg:   fakeFileInfo{name: "config.json", mode: 0600},
		}, SeverityOK},
		{"loose-perms", map[string]os.FileInfo{
			state: fakeFileInfo{name: "state.json", mode: 0644},
			cfg:   fakeFileInfo{name: "config.json", mode: 0600},
		}, SeverityWarn},
		{"absent-files-ok", map[string]os.FileInfo{}, SeverityOK},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			env := &Env{
				StatePath:      state,
				XrayConfigPath: cfg,
				Stat:           statMap(tc.files),
			}
			got := checkFilePerms(context.Background(), env)
			if got.Severity != tc.want {
				t.Errorf("severity = %v, want %v (detail=%q)", got.Severity, tc.want, got.Detail)
			}
		})
	}
}

func TestCheckFilePerms_FlagsLooseServersYAML(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX perms not checked on Windows")
	}
	dir := t.TempDir()
	servers := filepath.Join(dir, "servers.yaml")
	if err := os.WriteFile(servers, []byte("profiles: []\n"), 0o644); err != nil {
		t.Fatalf("seed servers.yaml: %v", err)
	}
	env := &Env{
		GOOS:        runtime.GOOS,
		Stat:        os.Stat,
		ServersPath: servers,
		// StatePath / XrayConfigPath left empty → absent → ignored by the check.
	}
	r := checkFilePerms(context.Background(), env)
	if r.Severity != SeverityWarn {
		t.Fatalf("severity = %v, want Warn for a world-readable servers.yaml", r.Severity)
	}
	if !strings.Contains(r.Detail, "servers.yaml") {
		t.Errorf("detail %q does not mention servers.yaml", r.Detail)
	}
}

func TestCheckFilePermsSkippedOnWindows(t *testing.T) {
	env := &Env{
		StatePath:      "/data/state.json",
		XrayConfigPath: "/data/config.json",
		Stat: statMap(map[string]os.FileInfo{
			"/data/state.json": fakeFileInfo{name: "state.json", mode: 0644},
		}),
	}
	got := checkFilePerms(context.Background(), env)
	if runtime.GOOS == "windows" && got.Severity != SeverityInfo {
		t.Errorf("on windows severity = %v, want INFO", got.Severity)
	}
}
