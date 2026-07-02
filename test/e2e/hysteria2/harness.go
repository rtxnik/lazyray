//go:build e2e

package hysteria2_e2e

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// lzrEnv returns an isolated environment (temp HOME) plus the lazyray data dir,
// so the e2e never touches the developer's real lazyray config/data.
func lzrEnv(t *testing.T) ([]string, string) {
	t.Helper()
	home := t.TempDir()
	env := append(os.Environ(),
		"HOME="+home,
		"XDG_CONFIG_HOME="+filepath.Join(home, ".config"),
		"XDG_DATA_HOME="+filepath.Join(home, ".local", "share"),
	)
	return env, filepath.Join(home, ".local", "share", "lazyray")
}

// provisionXray copies XRAY_BIN (and its geo data) into the lazyray data dir.
// lazyray's generated routing references geoip:private, so xray needs the geo
// files next to the binary or config loading fails.
func provisionXray(t *testing.T, dataDir string) {
	t.Helper()
	src := os.Getenv("XRAY_BIN")
	if src == "" {
		t.Skip("XRAY_BIN not set; skipping e2e")
	}
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		t.Fatal(err)
	}
	copyFile(t, src, filepath.Join(dataDir, "xray"), 0o755)
	srcDir := filepath.Dir(src)
	for _, dat := range []string{"geoip.dat", "geosite.dat"} {
		if _, err := os.Stat(filepath.Join(srcDir, dat)); err == nil {
			copyFile(t, filepath.Join(srcDir, dat), filepath.Join(dataDir, dat), 0o644)
		}
	}
}

func copyFile(t *testing.T, src, dst string, mode os.FileMode) {
	t.Helper()
	b, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("read %s: %v", src, err)
	}
	if err := os.WriteFile(dst, b, mode); err != nil {
		t.Fatalf("write %s: %v", dst, err)
	}
}

// lzrBin returns the lzr binary path (LZR_BIN env, or built from the repo root).
func lzrBin(t *testing.T) string {
	t.Helper()
	if b := os.Getenv("LZR_BIN"); b != "" {
		return b
	}
	out := filepath.Join(t.TempDir(), "lzr")
	cmd := exec.Command("go", "build", "-o", out, ".")
	cmd.Dir = repoRoot(t)
	if b, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build lzr: %v\n%s", err, b)
	}
	return out
}

func repoRoot(t *testing.T) string {
	t.Helper()
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		t.Fatalf("repo root: %v", err)
	}
	return strings.TrimSpace(string(out))
}

// run executes lzr with the isolated env and returns combined output.
func run(t *testing.T, env []string, bin string, args ...string) (string, error) {
	t.Helper()
	cmd := exec.Command(bin, args...)
	cmd.Env = env
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// socksPortFromConfig reads the generated xray config.json and returns the socks
// inbound port (deterministic: whatever lazyray generated).
func socksPortFromConfig(t *testing.T, dataDir string) int {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dataDir, "config.json"))
	if err != nil {
		t.Fatalf("read config.json: %v", err)
	}
	var cfg struct {
		Inbounds []struct {
			Protocol string `json:"protocol"`
			Port     int    `json:"port"`
		} `json:"inbounds"`
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("parse config.json: %v", err)
	}
	for _, in := range cfg.Inbounds {
		if in.Protocol == "socks" {
			return in.Port
		}
	}
	t.Fatal("no socks inbound in config.json")
	return 0
}

// curlThroughProxy fetches targetURL via the SOCKS5 proxy (remote DNS), retrying
// while the tunnel warms up.
func curlThroughProxy(t *testing.T, socksPort int, targetURL string) (string, error) {
	t.Helper()
	socks := "127.0.0.1:" + strconv.Itoa(socksPort)
	var lastErr error
	for i := 0; i < 15; i++ {
		out, err := exec.Command("curl", "-s", "--max-time", "5",
			"--socks5-hostname", socks, targetURL).CombinedOutput()
		if err == nil && len(out) > 0 {
			return string(out), nil
		}
		lastErr = err
		time.Sleep(time.Second)
	}
	return "", lastErr
}
