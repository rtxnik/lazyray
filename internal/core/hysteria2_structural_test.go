package core

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/rtxnik/lazyray/internal/config"
)

// xrayBinForTest returns the xray binary path from XRAY_BIN, or skips the test.
func xrayBinForTest(t *testing.T) string {
	t.Helper()
	bin := os.Getenv("XRAY_BIN")
	if bin == "" {
		t.Skip("XRAY_BIN not set; skipping structural xray -test validation")
	}
	if _, err := os.Stat(bin); err != nil {
		t.Skipf("XRAY_BIN %q not found: %v", bin, err)
	}
	return bin
}

// TestHysteria2Config_AcceptedByXray generates xray configs from representative
// hysteria2 profiles and confirms a real xray binary accepts them via `xray run
// -test`. This validates the emitted schema (hysteriaSettings, finalmask
// salamander, finalmask.quicParams.udpHop.ports, tlsSettings.pinnedPeerCertSha256)
// against the actual xray-core the project targets. Self-skips without XRAY_BIN.
func TestHysteria2Config_AcceptedByXray(t *testing.T) {
	bin := xrayBinForTest(t)
	// Isolate lazyray's config/data under a temp HOME and create the dirs xray's
	// logger needs. In real use lazyray's prepareStart calls EnsureDirs before
	// starting xray; this test drives xray directly, so it must do the same.
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(home, ".local", "share"))
	if err := config.EnsureDirs(); err != nil {
		t.Fatalf("ensure dirs: %v", err)
	}
	const pin = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	// NOTE: cases are pin-based, not allowInsecure-based. xray-core >= v26 has
	// REMOVED tlsSettings.allowInsecure (it now errors at config build), so the
	// supported "trust a self-signed/specific cert" path is pinnedPeerCertSha256.
	// These cases validate the schema this project adds: salamander finalmask,
	// pinnedPeerCertSha256, and finalmask.quicParams.udpHop.ports.
	cases := map[string]config.ServerConfig{
		"salamander_pinned": {
			Address: "127.0.0.1", Port: 8443, UUID: "auth", Protocol: "hysteria2",
			Obfs: "salamander", ObfsPassword: "op",
			Transport: config.TransportConfig{Network: "hysteria"},
			Security:  config.SecurityConfig{Type: "tls", SNI: "h", PinSHA256: pin},
		},
		"pinned": {
			Address: "127.0.0.1", Port: 8443, UUID: "auth", Protocol: "hysteria2",
			Transport: config.TransportConfig{Network: "hysteria"},
			Security:  config.SecurityConfig{Type: "tls", SNI: "h", PinSHA256: pin},
		},
		"porthop_pinned": {
			Address: "127.0.0.1", Port: 8443, UUID: "auth", Protocol: "hysteria2",
			PortHopping: "8443,9000-9100",
			Transport:   config.TransportConfig{Network: "hysteria"},
			Security:    config.SecurityConfig{Type: "tls", SNI: "h", PinSHA256: pin},
		},
	}
	settings := config.DefaultSettings()
	for name, srv := range cases {
		srv := srv
		t.Run(name, func(t *testing.T) {
			prof := &config.Profile{Name: name, Server: srv}
			cfg, err := GenerateXrayConfig(prof, settings)
			if err != nil {
				t.Fatalf("generate: %v", err)
			}
			raw, _ := json.MarshalIndent(cfg, "", "  ")
			path := filepath.Join(t.TempDir(), "config.json")
			if err := os.WriteFile(path, raw, 0644); err != nil {
				t.Fatal(err)
			}
			out, err := exec.Command(bin, "run", "-test", "-c", path).CombinedOutput()
			if err != nil {
				t.Fatalf("xray rejected %s config: %v\n%s", name, err, out)
			}
		})
	}
}
