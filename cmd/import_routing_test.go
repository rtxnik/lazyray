package cmd

import (
	"testing"

	"github.com/rtxnik/lazyray/internal/config"
	"github.com/rtxnik/lazyray/internal/core"
	"github.com/spf13/cobra"
)

func encBlob(t *testing.T, profiles []config.Profile) string {
	t.Helper()
	blob, err := core.ExportEncrypted(profiles, "pw")
	if err != nil {
		t.Fatal(err)
	}
	return blob
}

func routingProfile() config.Profile {
	return config.Profile{
		Name:   "p",
		Server: config.ServerConfig{Address: "h.example", Port: 443, UUID: "11111111-1111-1111-1111-111111111111", Transport: config.TransportConfig{Network: "tcp"}, Security: config.SecurityConfig{Type: "none"}},
		Routing: config.ProfileRouting{
			Bypass:   []string{"example.com"},
			DNSRules: []config.DNSRule{{Server: "https://dns.google/dns-query", Domains: []string{"a.com"}}},
		},
	}
}

func TestImportEncrypted_DropsRoutingByDefault(t *testing.T) {
	isolateConfig(t)
	importDecrypt = "pw"
	importAllowRouting = false
	t.Cleanup(func() { importDecrypt = ""; importAllowRouting = false })

	if err := importEncrypted(&cobra.Command{}, encBlob(t, []config.Profile{routingProfile()})); err != nil {
		t.Fatal(err)
	}
	servers, err := config.LoadServers()
	if err != nil {
		t.Fatal(err)
	}
	if len(servers.Profiles) != 1 || core.HasRoutingOverrides(&servers.Profiles[0]) {
		t.Fatalf("routing overrides should have been dropped: %+v", servers.Profiles)
	}
}

func TestImportEncrypted_AllowRoutingRejectsBadDNS(t *testing.T) {
	isolateConfig(t)
	p := routingProfile()
	p.Routing.DNSRules = []config.DNSRule{{Server: "file:///etc/passwd"}}
	importDecrypt = "pw"
	importAllowRouting = true
	t.Cleanup(func() { importDecrypt = ""; importAllowRouting = false })

	if err := importEncrypted(&cobra.Command{}, encBlob(t, []config.Profile{p})); err != nil {
		t.Fatal(err)
	}
	servers, _ := config.LoadServers()
	if len(servers.Profiles) != 0 {
		t.Fatalf("profile with disallowed DNS server should be skipped, got %+v", servers.Profiles)
	}
}
