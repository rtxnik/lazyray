package core

import (
	"strings"
	"testing"

	"github.com/rtxnik/lazyray/internal/config"
)

func TestStripControl(t *testing.T) {
	cases := map[string]string{
		"\x1b[31mHACKED\x1b[0m": "[31mHACKED[0m", // ESC dropped, printable kept
		"clean-name":            "clean-name",
		"tab\tnl\nx":            "tabnlx",
		"привет🚀":               "привет🚀", // multi-byte UTF-8 preserved
		"\u0085next":            "next",    // C1 NEL dropped
		"del\x7fx":              "delx",    // DEL dropped
		"\x1b\x1b":              "",        // all-control input reduces to empty
		"":                      "",
	}
	for in, want := range cases {
		if got := StripControl(in); got != want {
			t.Errorf("StripControl(%q) = %q, want %q", in, got, want)
		}
	}
	// Idempotent.
	once := StripControl("\x1b[1mx")
	if StripControl(once) != once {
		t.Errorf("StripControl not idempotent")
	}
}

func TestValidateDNSServer(t *testing.T) {
	ok := []string{"8.8.8.8", "8.8.8.8:53", "[2001:db8::1]:53",
		"https://dns.google/dns-query", "tcp://1.1.1.1:53", "https+local://1.1.1.1/dns-query"}
	bad := []string{"", "dns.google", "http://evil/dns-query", "file:///etc/passwd", "javascript:alert(1)",
		"8.8.8.8:99999", "8.8.8.8:0", "8.8.8.8:bad"}
	for _, s := range ok {
		if err := ValidateDNSServer(s); err != nil {
			t.Errorf("ValidateDNSServer(%q) = %v, want nil", s, err)
		}
	}
	for _, s := range bad {
		if err := ValidateDNSServer(s); err == nil {
			t.Errorf("ValidateDNSServer(%q) = nil, want error", s)
		}
	}
}

func TestSanitizeProfileDisplay_StripsNameNetworkAndChain(t *testing.T) {
	p := &config.Profile{
		Name:   "a\x1bb",
		Server: config.ServerConfig{Address: "h\x1bost", Transport: config.TransportConfig{Network: "w\x1bs"}},
		Chain:  []config.ServerConfig{{Address: "c\x1bhain", Transport: config.TransportConfig{Network: "t\x1bcp"}}},
		SSH:    config.SSHConfig{Host: "h\x1bost.example", User: "u\x1bser"},
	}
	SanitizeProfileDisplay(p)
	if strings.ContainsRune(p.Name, 0x1b) || strings.ContainsRune(p.Server.Address, 0x1b) ||
		strings.ContainsRune(p.Server.Transport.Network, 0x1b) ||
		strings.ContainsRune(p.Chain[0].Address, 0x1b) || strings.ContainsRune(p.Chain[0].Transport.Network, 0x1b) ||
		strings.ContainsRune(p.SSH.Host, 0x1b) || strings.ContainsRune(p.SSH.User, 0x1b) {
		t.Errorf("control char survived: %+v", p)
	}
	if p.SSH.Host != "host.example" {
		t.Errorf("SSH.Host = %q, want %q", p.SSH.Host, "host.example")
	}
	if p.SSH.User != "user" {
		t.Errorf("SSH.User = %q, want %q", p.SSH.User, "user")
	}
}

func TestHasRoutingOverrides(t *testing.T) {
	if HasRoutingOverrides(&config.Profile{}) {
		t.Error("empty routing should be false")
	}
	if !HasRoutingOverrides(&config.Profile{Routing: config.ProfileRouting{Bypass: []string{"x"}}}) {
		t.Error("non-empty bypass should be true")
	}
}
