//go:build e2e

package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/rtxnik/lazyray/internal/config"
)

// realHome captures HOME at package-load time, before any test redirects it via
// t.Setenv. buildLZR uses it so `go build` keeps the real Go module cache /
// build cache instead of redownloading modules (read-only) into the test's
// temp HOME, which would then break TempDir cleanup.
var realHome = os.Getenv("HOME")

// writeFakeXray writes an executable shell script at path that stands in for the
// real xray binary during the e2e. For `version` it prints a plausible version
// banner; for `run` it exec's a real, long-lived `sleep` so the supervised
// process is a genuine, SIGTERM-killable child.
func writeFakeXray(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("writeFakeXray: mkdir: %v", err)
	}
	script := `#!/bin/sh
case "$1" in
  version)
    echo "Xray 26.3.0 (Xray, Penetrates Everything.) custom (go1.24.7 linux/amd64)"
    echo "A unified platform for anti-censorship."
    exit 0
    ;;
  run)
    # Become a real long-lived process the supervisor can SIGTERM/SIGKILL.
    exec sleep 600
    ;;
  *)
    exec sleep 600
    ;;
esac
`
	if err := os.WriteFile(path, []byte(script), 0755); err != nil {
		t.Fatalf("writeFakeXray: write: %v", err)
	}
	// Ensure the exec bit survived umask.
	if err := os.Chmod(path, 0755); err != nil {
		t.Fatalf("writeFakeXray: chmod: %v", err)
	}
}

// seedMinimalProfile persists a minimal but valid VLESS profile and default
// settings under the temp HOME, so the supervisor's core.WriteXrayConfig (which
// runs ValidateProfile + GenerateXrayConfig) succeeds. We use the config package
// writers rather than hand-writing YAML.
//
// Minimal valid VLESS (non-reality) per internal/core ValidateProfile:
//   - Server.UUID set
//   - Server.Address set
//   - Server.Port in 1-65535
//   - Server.Transport.Network non-empty
//   - Security.Type != "reality" (so no reality publicKey/SNI/fingerprint needed)
func seedMinimalProfile(t *testing.T) {
	t.Helper()

	servers := &config.ServersConfig{
		ConfigVersion: config.CurrentConfigVersion,
		Profiles: []config.Profile{
			{
				Name:    "e2e",
				Default: true,
				Server: config.ServerConfig{
					Address:    "203.0.113.10",
					Port:       8443,
					UUID:       "123e4567-e89b-42d3-a456-426614174000",
					Protocol:   "vless",
					Encryption: "none",
					Transport: config.TransportConfig{
						Network: "tcp",
					},
					Security: config.SecurityConfig{
						Type: "none",
					},
				},
			},
		},
	}
	if err := config.SaveServers(servers); err != nil {
		t.Fatalf("seedMinimalProfile: SaveServers: %v", err)
	}
	settings := config.DefaultSettings()
	settings.AutoSystemProxy = false
	if err := config.SaveSettings(settings); err != nil {
		t.Fatalf("seedMinimalProfile: SaveSettings: %v", err)
	}
}

// buildLZR compiles the lzr binary from the repo root into a temp dir and
// returns its path. The test's working dir is the cmd package dir, so the repo
// root is one level up.
func buildLZR(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "lzr")
	cmd := exec.Command("go", "build", "-o", bin, ".")
	cmd.Dir = ".."
	// Pin HOME to the real one so go build resolves the normal GOPATH/module
	// cache instead of populating a read-only cache inside the test's temp HOME
	// (which TempDir cleanup cannot then remove).
	cmd.Env = append(os.Environ(), "HOME="+realHome)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("buildLZR: go build: %v\n%s", err, out)
	}
	return bin
}

// syscallKillBestEffort sends SIGKILL to a supervisor PID (and its process
// group) for test-teardown cleanup. Errors are ignored.
func syscallKillBestEffort(pid int) error {
	if pid <= 0 {
		return nil
	}
	_ = syscall.Kill(-pid, syscall.SIGKILL)
	_ = syscall.Kill(pid, syscall.SIGKILL)
	return nil
}
