package config

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestSupervisorLogPath_UnderLogDir(t *testing.T) {
	got := SupervisorLogPath()
	want := filepath.Join(LogDir(), "supervisor.log")
	if got != want {
		t.Errorf("SupervisorLogPath() = %q, want %q", got, want)
	}
}

func TestLastErrorPath_UnderDataDir(t *testing.T) {
	got := LastErrorPath()
	want := filepath.Join(DataDir(), "last-error.json")
	if got != want {
		t.Errorf("LastErrorPath() = %q, want %q", got, want)
	}
}

func TestTunnelPIDPath_Sanitize(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantFile string
	}{
		{
			name:     "simple name",
			input:    "my-tunnel",
			wantFile: "tunnel-my-tunnel.pid",
		},
		{
			name:     "with spaces",
			input:    "Alpha Beta Cascade",
			wantFile: "tunnel-Alpha_Beta_Cascade.pid",
		},
		{
			name:     "with arrow unicode",
			input:    "Alpha→Beta",
			wantFile: "tunnel-Alpha_Beta.pid",
		},
		{
			name:     "with special chars",
			input:    "test/path:name@host",
			wantFile: "tunnel-test_path_name_host.pid",
		},
		{
			name:     "allowed chars preserved",
			input:    "abc-123_XYZ",
			wantFile: "tunnel-abc-123_XYZ.pid",
		},
		{
			name:     "dots replaced",
			input:    "server.example.com",
			wantFile: "tunnel-server_example_com.pid",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := TunnelPIDPath(tc.input)
			base := filepath.Base(result)
			if base != tc.wantFile {
				t.Errorf("TunnelPIDPath(%q) base = %q, want %q", tc.input, base, tc.wantFile)
			}
			// Result should be under DataDir
			if !strings.HasPrefix(result, DataDir()) {
				t.Errorf("TunnelPIDPath(%q) = %q, not under DataDir %q", tc.input, result, DataDir())
			}
		})
	}
}

func TestTunnelPIDGlob(t *testing.T) {
	result := TunnelPIDGlob()
	base := filepath.Base(result)
	if base != "tunnel-*.pid" {
		t.Errorf("TunnelPIDGlob() base = %q, want %q", base, "tunnel-*.pid")
	}
	if !strings.HasPrefix(result, DataDir()) {
		t.Errorf("TunnelPIDGlob() = %q, not under DataDir %q", result, DataDir())
	}
}
