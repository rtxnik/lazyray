package core

import (
	"testing"

	"github.com/rtxnik/lazyray/internal/config"
)

func TestGenerateXrayConfig_MultiHopChain(t *testing.T) {
	profile := testProfile()
	profile.Chain = []config.ServerConfig{
		{
			Address:    "5.6.7.8",
			Port:       443,
			UUID:       "exit-uuid",
			Encryption: "none",
			Transport: config.TransportConfig{
				Network: "xhttp",
				Path:    "/exit",
				Mode:    "auto",
			},
			Security: config.SecurityConfig{
				Type:        "reality",
				SNI:         "exit.example.com",
				Fingerprint: "chrome",
				PublicKey:   "EXIT_KEY",
				ShortID:     "exit1",
			},
		},
	}

	cfg, err := GenerateXrayConfig(profile, testSettings())
	if err != nil {
		t.Fatalf("GenerateXrayConfig() error = %v", err)
	}

	// Should have: hop-0 (entry) + proxy (exit) + direct + block = 4 outbounds
	if len(cfg.Outbounds) != 4 {
		t.Fatalf("Outbounds count = %d, want 4 for 2-hop chain", len(cfg.Outbounds))
	}

	// First outbound: hop-0 (entry node, no proxy settings)
	hop0 := cfg.Outbounds[0]
	if hop0.Tag != "hop-0" {
		t.Errorf("Outbound[0].Tag = %q, want %q", hop0.Tag, "hop-0")
	}
	if hop0.ProxySettings != nil {
		t.Error("hop-0 should not have proxySettings (it's the entry)")
	}

	// Second outbound: proxy (exit node, with proxySettings pointing to hop-0)
	proxyOut := cfg.Outbounds[1]
	if proxyOut.Tag != "proxy" {
		t.Errorf("Outbound[1].Tag = %q, want %q", proxyOut.Tag, "proxy")
	}
	if proxyOut.ProxySettings == nil {
		t.Fatal("proxy should have proxySettings")
	}
	if proxyOut.ProxySettings.Tag != "hop-0" {
		t.Errorf("proxy proxySettings.Tag = %q, want %q", proxyOut.ProxySettings.Tag, "hop-0")
	}
}

func TestGenerateXrayConfig_ThreeHopChain(t *testing.T) {
	profile := testProfile()
	profile.Chain = []config.ServerConfig{
		{
			Address: "10.0.0.1", Port: 443, UUID: "mid-uuid",
			Encryption: "none",
			Transport:  config.TransportConfig{Network: "tcp"},
			Security:   config.SecurityConfig{Type: "none"},
		},
		{
			Address: "10.0.0.2", Port: 443, UUID: "exit-uuid",
			Encryption: "none",
			Transport:  config.TransportConfig{Network: "tcp"},
			Security:   config.SecurityConfig{Type: "none"},
		},
	}

	cfg, err := GenerateXrayConfig(profile, testSettings())
	if err != nil {
		t.Fatalf("GenerateXrayConfig() error = %v", err)
	}

	// hop-0 + hop-1 + proxy + direct + block = 5
	if len(cfg.Outbounds) != 5 {
		t.Fatalf("Outbounds count = %d, want 5 for 3-hop chain", len(cfg.Outbounds))
	}

	// Verify chain linking
	if cfg.Outbounds[0].Tag != "hop-0" {
		t.Errorf("Outbound[0].Tag = %q, want hop-0", cfg.Outbounds[0].Tag)
	}
	if cfg.Outbounds[1].Tag != "hop-1" {
		t.Errorf("Outbound[1].Tag = %q, want hop-1", cfg.Outbounds[1].Tag)
	}
	if cfg.Outbounds[1].ProxySettings == nil || cfg.Outbounds[1].ProxySettings.Tag != "hop-0" {
		t.Error("hop-1 should proxy through hop-0")
	}
	if cfg.Outbounds[2].Tag != "proxy" {
		t.Errorf("Outbound[2].Tag = %q, want proxy", cfg.Outbounds[2].Tag)
	}
	if cfg.Outbounds[2].ProxySettings == nil || cfg.Outbounds[2].ProxySettings.Tag != "hop-1" {
		t.Error("proxy should proxy through hop-1")
	}
}

func TestIsIPEntry(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"geoip:private", true},
		{"geoip:ru", true},
		{"10.0.0.0/8", true},
		{"192.168.1.1", true},
		{"::1", true},
		{"domain:google.com", false},
		{"geosite:google", false},
		{"regexp:.*google.*", false},
		{"full:google.com", false},
		{"keyword:google", false},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got := isIPEntry(tc.input)
			if got != tc.want {
				t.Errorf("isIPEntry(%q) = %v, want %v", tc.input, got, tc.want)
			}
		})
	}
}

func TestBuildRoutingRule_Empty(t *testing.T) {
	rule := buildRoutingRule(nil, "direct")
	if rule != nil {
		t.Error("buildRoutingRule with empty entries should return nil")
	}
}

func TestBuildRoutingRule_IPsOnly(t *testing.T) {
	rule := buildRoutingRule([]string{"geoip:private", "10.0.0.0/8"}, "direct")
	if rule == nil {
		t.Fatal("rule should not be nil")
	}
	if rule.OutboundTag != "direct" {
		t.Errorf("OutboundTag = %q, want direct", rule.OutboundTag)
	}
	if len(rule.IP) != 2 {
		t.Errorf("IP count = %d, want 2", len(rule.IP))
	}
	if len(rule.Domain) != 0 {
		t.Errorf("Domain count = %d, want 0", len(rule.Domain))
	}
}

func TestBuildRoutingRule_DomainsOnly(t *testing.T) {
	rule := buildRoutingRule([]string{"domain:google.com", "geosite:ads"}, "block")
	if rule == nil {
		t.Fatal("rule should not be nil")
	}
	if len(rule.Domain) != 2 {
		t.Errorf("Domain count = %d, want 2", len(rule.Domain))
	}
	if len(rule.IP) != 0 {
		t.Errorf("IP count = %d, want 0", len(rule.IP))
	}
}

func TestBuildRoutingRule_Mixed(t *testing.T) {
	rule := buildRoutingRule([]string{"geoip:ru", "domain:yandex.ru"}, "direct")
	if rule == nil {
		t.Fatal("rule should not be nil")
	}
	if len(rule.IP) != 1 {
		t.Errorf("IP count = %d, want 1", len(rule.IP))
	}
	if len(rule.Domain) != 1 {
		t.Errorf("Domain count = %d, want 1", len(rule.Domain))
	}
}
