//go:build e2e

package hysteria2_e2e

import (
	"os"
	"strings"
	"testing"
)

// readPin returns the client pinSHA256 written by gen-cert.sh.
func readPin(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile("certs/pin.sha256")
	if err != nil {
		t.Fatalf("read certs/pin.sha256 (run ./gen-cert.sh first): %v", err)
	}
	return strings.TrimSpace(string(b))
}

// runEgressCase imports a hysteria2 link into an isolated lazyray, starts xray,
// fetches targetURL through the SOCKS proxy, and asserts wantSubstr is present.
func runEgressCase(t *testing.T, link, targetURL, wantSubstr string) {
	t.Helper()
	env, dataDir := lzrEnv(t)
	provisionXray(t, dataDir)
	bin := lzrBin(t)

	if out, err := run(t, env, bin, "import", link, "--force"); err != nil {
		t.Fatalf("import: %v\n%s", err, out)
	}
	// --no-proxy: this case exercises the SOCKS data tunnel, not system-proxy
	// integration (headless CI has no proxy backend, and AutoSystemProxy now
	// defaults on).
	if out, err := run(t, env, bin, "start", "--no-proxy"); err != nil {
		t.Fatalf("start: %v\n%s", err, out)
	}
	t.Cleanup(func() { _, _ = run(t, env, bin, "stop") })

	port := socksPortFromConfig(t, dataDir)
	body, err := curlThroughProxy(t, port, targetURL)
	if err != nil {
		t.Fatalf("curl through proxy failed: %v", err)
	}
	if !strings.Contains(body, wantSubstr) {
		t.Fatalf("response %q does not contain %q", body, wantSubstr)
	}
}

// TestE2E_Hysteria2_Pinned drives the apernet/hysteria server (docker compose)
// with a pinSHA256-pinned client link and asserts egress to the whoami target.
// xray-core >= v26 removed allowInsecure, so pinning is the supported
// self-signed-trust path (this is also what the lazyray client emits).
func TestE2E_Hysteria2_Pinned(t *testing.T) {
	pin := readPin(t)
	link := "hysteria2://e2e-test-password@127.0.0.1:8443/?pinSHA256=" + pin +
		"&obfs=salamander&obfs-password=e2e-obfs-password&sni=hy2.test.local#e2e-pin"
	runEgressCase(t, link, "http://whoami/", "Hostname:")
}
