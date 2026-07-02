package core

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/rtxnik/lazyray/internal/config"
)

func testProfile() *config.Profile {
	return &config.Profile{
		Name: "Test",
		Server: config.ServerConfig{
			Address:    "192.0.2.10",
			Port:       8443,
			UUID:       "123e4567-e89b-42d3-a456-426614174000",
			Encryption: "none",
			Flow:       "",
			Transport: config.TransportConfig{
				Network: "xhttp",
				Path:    "/TestPath8",
				Mode:    "auto",
			},
			Security: config.SecurityConfig{
				Type:        "reality",
				SNI:         "example.org",
				Fingerprint: "chrome",
				PublicKey:   "DXLqqc2ZxtxKHm_ab5GnF59s4d0SLpWz8VOwlsW3wyY",
				ShortID:     "abc123",
				SpiderX:     "/",
			},
		},
	}
}

func testSettings() *config.Settings {
	return config.DefaultSettings()
}

func TestGenerateXrayConfig_BasicStructure(t *testing.T) {
	cfg, err := GenerateXrayConfig(testProfile(), testSettings())
	if err != nil {
		t.Fatalf("GenerateXrayConfig() error = %v", err)
	}

	if cfg.Log.LogLevel != "warning" {
		t.Errorf("LogLevel = %q, want %q", cfg.Log.LogLevel, "warning")
	}

	if len(cfg.DNS.Servers) != 2 {
		t.Errorf("DNS servers count = %d, want 2", len(cfg.DNS.Servers))
	}
}

func TestGenerateXrayConfig_Inbounds(t *testing.T) {
	cfg, err := GenerateXrayConfig(testProfile(), testSettings())
	if err != nil {
		t.Fatalf("GenerateXrayConfig() error = %v", err)
	}

	if len(cfg.Inbounds) != 3 {
		t.Fatalf("Inbounds count = %d, want 3", len(cfg.Inbounds))
	}

	// SOCKS5
	socks := cfg.Inbounds[0]
	if socks.Tag != "socks-in" {
		t.Errorf("Inbound[0].Tag = %q, want %q", socks.Tag, "socks-in")
	}
	if socks.Port != 10808 {
		t.Errorf("Inbound[0].Port = %d, want %d", socks.Port, 10808)
	}
	if socks.Protocol != "socks" {
		t.Errorf("Inbound[0].Protocol = %q, want %q", socks.Protocol, "socks")
	}
	if socks.Listen != "127.0.0.1" {
		t.Errorf("Inbound[0].Listen = %q, want %q", socks.Listen, "127.0.0.1")
	}
	if socks.Sniffing == nil || !socks.Sniffing.Enabled {
		t.Error("SOCKS5 sniffing should be enabled")
	}

	// HTTP
	httpIn := cfg.Inbounds[1]
	if httpIn.Tag != "http-in" {
		t.Errorf("Inbound[1].Tag = %q, want %q", httpIn.Tag, "http-in")
	}
	if httpIn.Port != 10809 {
		t.Errorf("Inbound[1].Port = %d, want %d", httpIn.Port, 10809)
	}
	if httpIn.Protocol != "http" {
		t.Errorf("Inbound[1].Protocol = %q, want %q", httpIn.Protocol, "http")
	}

	// Stats API (dokodemo-door)
	api := cfg.Inbounds[2]
	if api.Tag != "api-in" {
		t.Errorf("Inbound[2].Tag = %q, want %q", api.Tag, "api-in")
	}
	if api.Port != StatsAPIPort {
		t.Errorf("Inbound[2].Port = %d, want %d", api.Port, StatsAPIPort)
	}
	if api.Protocol != "dokodemo-door" {
		t.Errorf("Inbound[2].Protocol = %q, want %q", api.Protocol, "dokodemo-door")
	}
}

func TestGenerateXrayConfig_StatsAPI(t *testing.T) {
	cfg, err := GenerateXrayConfig(testProfile(), testSettings())
	if err != nil {
		t.Fatalf("GenerateXrayConfig() error = %v", err)
	}

	if cfg.API == nil {
		t.Fatal("API section should not be nil")
	}
	if cfg.API.Tag != "api" {
		t.Errorf("API.Tag = %q, want %q", cfg.API.Tag, "api")
	}
	if len(cfg.API.Services) != 1 || cfg.API.Services[0] != "StatsService" {
		t.Errorf("API.Services = %v, want [StatsService]", cfg.API.Services)
	}

	if cfg.Stats == nil {
		t.Error("Stats section should not be nil")
	}

	if cfg.Policy == nil {
		t.Fatal("Policy section should not be nil")
	}
	if !cfg.Policy.System.StatsOutboundUplink {
		t.Error("Policy.System.StatsOutboundUplink should be true")
	}
	if !cfg.Policy.System.StatsOutboundDownlink {
		t.Error("Policy.System.StatsOutboundDownlink should be true")
	}
}

func TestGenerateXrayConfig_Outbounds(t *testing.T) {
	profile := testProfile()
	cfg, err := GenerateXrayConfig(profile, testSettings())
	if err != nil {
		t.Fatalf("GenerateXrayConfig() error = %v", err)
	}

	if len(cfg.Outbounds) != 3 {
		t.Fatalf("Outbounds count = %d, want 3", len(cfg.Outbounds))
	}

	// Proxy outbound
	proxy := cfg.Outbounds[0]
	if proxy.Tag != "proxy" {
		t.Errorf("Outbound[0].Tag = %q, want %q", proxy.Tag, "proxy")
	}
	if proxy.Protocol != "vless" {
		t.Errorf("Outbound[0].Protocol = %q, want %q", proxy.Protocol, "vless")
	}

	// Verify settings contain correct server info
	var settings map[string]interface{}
	if err := json.Unmarshal(proxy.Settings, &settings); err != nil {
		t.Fatalf("unmarshaling proxy settings: %v", err)
	}
	vnext, ok := settings["vnext"].([]interface{})
	if !ok || len(vnext) == 0 {
		t.Fatal("proxy settings should contain vnext array")
	}
	server := vnext[0].(map[string]interface{})
	if server["address"] != profile.Server.Address {
		t.Errorf("vnext address = %v, want %v", server["address"], profile.Server.Address)
	}
	if int(server["port"].(float64)) != profile.Server.Port {
		t.Errorf("vnext port = %v, want %v", server["port"], profile.Server.Port)
	}

	// StreamSettings
	if proxy.StreamSettings == nil {
		t.Fatal("proxy StreamSettings should not be nil")
	}
	if proxy.StreamSettings.Network != "xhttp" {
		t.Errorf("StreamSettings.Network = %q, want %q", proxy.StreamSettings.Network, "xhttp")
	}
	if proxy.StreamSettings.Security != "reality" {
		t.Errorf("StreamSettings.Security = %q, want %q", proxy.StreamSettings.Security, "reality")
	}
	if proxy.StreamSettings.XHTTPSettings == nil {
		t.Fatal("XHTTPSettings should not be nil")
	}
	if proxy.StreamSettings.XHTTPSettings.Path != "/TestPath8" {
		t.Errorf("XHTTPSettings.Path = %q, want %q", proxy.StreamSettings.XHTTPSettings.Path, "/TestPath8")
	}
	if proxy.StreamSettings.RealitySettings == nil {
		t.Fatal("RealitySettings should not be nil")
	}
	if proxy.StreamSettings.RealitySettings.ServerName != "example.org" {
		t.Errorf("RealitySettings.ServerName = %q, want %q", proxy.StreamSettings.RealitySettings.ServerName, "example.org")
	}
	if proxy.StreamSettings.RealitySettings.Fingerprint != "chrome" {
		t.Errorf("RealitySettings.Fingerprint = %q, want %q", proxy.StreamSettings.RealitySettings.Fingerprint, "chrome")
	}

	// Direct and block outbounds
	if cfg.Outbounds[1].Tag != "direct" || cfg.Outbounds[1].Protocol != "freedom" {
		t.Errorf("Outbound[1] should be direct/freedom, got %s/%s", cfg.Outbounds[1].Tag, cfg.Outbounds[1].Protocol)
	}
	if cfg.Outbounds[2].Tag != "block" || cfg.Outbounds[2].Protocol != "blackhole" {
		t.Errorf("Outbound[2] should be block/blackhole, got %s/%s", cfg.Outbounds[2].Tag, cfg.Outbounds[2].Protocol)
	}
}

func TestGenerateXrayConfig_Routing(t *testing.T) {
	cfg, err := GenerateXrayConfig(testProfile(), testSettings())
	if err != nil {
		t.Fatalf("GenerateXrayConfig() error = %v", err)
	}

	if cfg.Routing.DomainStrategy != "AsIs" {
		t.Errorf("DomainStrategy = %q, want %q", cfg.Routing.DomainStrategy, "AsIs")
	}

	if len(cfg.Routing.Rules) < 2 {
		t.Fatalf("Routing rules count = %d, want at least 2", len(cfg.Routing.Rules))
	}

	// API inbound routing rule
	apiRule := cfg.Routing.Rules[0]
	if apiRule.OutboundTag != "api" {
		t.Errorf("Rule[0].OutboundTag = %q, want %q", apiRule.OutboundTag, "api")
	}
	if len(apiRule.InboundTag) == 0 || apiRule.InboundTag[0] != "api-in" {
		t.Errorf("Rule[0].InboundTag = %v, want [api-in]", apiRule.InboundTag)
	}

	// Private IP bypass rule
	privateRule := cfg.Routing.Rules[1]
	if privateRule.OutboundTag != "direct" {
		t.Errorf("Rule[1].OutboundTag = %q, want %q", privateRule.OutboundTag, "direct")
	}
	if len(privateRule.IP) == 0 || privateRule.IP[0] != "geoip:private" {
		t.Errorf("Rule[1].IP = %v, want [geoip:private]", privateRule.IP)
	}
}

func TestGenerateXrayConfig_JSON(t *testing.T) {
	cfg, err := GenerateXrayConfig(testProfile(), testSettings())
	if err != nil {
		t.Fatalf("GenerateXrayConfig() error = %v", err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		t.Fatalf("json.MarshalIndent() error = %v", err)
	}

	// Verify it produces valid JSON
	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("generated config is not valid JSON: %v", err)
	}

	// Verify key top-level fields exist
	for _, key := range []string{"log", "api", "stats", "policy", "inbounds", "outbounds", "routing"} {
		if _, ok := parsed[key]; !ok {
			t.Errorf("generated JSON missing top-level key %q", key)
		}
	}
}

func TestGenerateXrayConfig_CustomSettings(t *testing.T) {
	settings := testSettings()
	settings.Local.SocksPort = 1080
	settings.Local.HTTPPort = 1081
	settings.Local.Listen = "0.0.0.0"
	settings.Xray.LogLevel = "debug"

	cfg, err := GenerateXrayConfig(testProfile(), settings)
	if err != nil {
		t.Fatalf("GenerateXrayConfig() error = %v", err)
	}

	if cfg.Inbounds[0].Port != 1080 {
		t.Errorf("SOCKS5 port = %d, want 1080", cfg.Inbounds[0].Port)
	}
	if cfg.Inbounds[1].Port != 1081 {
		t.Errorf("HTTP port = %d, want 1081", cfg.Inbounds[1].Port)
	}
	if cfg.Inbounds[0].Listen != "0.0.0.0" {
		t.Errorf("Listen = %q, want %q", cfg.Inbounds[0].Listen, "0.0.0.0")
	}
	if cfg.Log.LogLevel != "debug" {
		t.Errorf("LogLevel = %q, want %q", cfg.Log.LogLevel, "debug")
	}
}

func TestGenerateXrayConfig_ProfileRouting(t *testing.T) {
	profile := testProfile()
	profile.Routing = config.ProfileRouting{
		Bypass: []string{"geoip:ru", "domain:yandex.ru"},
		Block:  []string{"geosite:category-ads"},
	}

	cfg, err := GenerateXrayConfig(profile, testSettings())
	if err != nil {
		t.Fatalf("GenerateXrayConfig() error = %v", err)
	}

	// Base rules (api-in, geoip:private) + bypass rule + block rule = 4
	if len(cfg.Routing.Rules) != 4 {
		t.Fatalf("Routing rules count = %d, want 4", len(cfg.Routing.Rules))
	}

	// Bypass rule
	bypassRule := cfg.Routing.Rules[2]
	if bypassRule.OutboundTag != "direct" {
		t.Errorf("bypass rule OutboundTag = %q, want %q", bypassRule.OutboundTag, "direct")
	}
	foundGeoIP := false
	for _, ip := range bypassRule.IP {
		if ip == "geoip:ru" {
			foundGeoIP = true
		}
	}
	if !foundGeoIP {
		t.Errorf("bypass rule IP = %v, should contain geoip:ru", bypassRule.IP)
	}
	foundDomain := false
	for _, d := range bypassRule.Domain {
		if d == "domain:yandex.ru" {
			foundDomain = true
		}
	}
	if !foundDomain {
		t.Errorf("bypass rule Domain = %v, should contain domain:yandex.ru", bypassRule.Domain)
	}

	// Block rule
	blockRule := cfg.Routing.Rules[3]
	if blockRule.OutboundTag != "block" {
		t.Errorf("block rule OutboundTag = %q, want %q", blockRule.OutboundTag, "block")
	}
	if len(blockRule.Domain) == 0 || blockRule.Domain[0] != "geosite:category-ads" {
		t.Errorf("block rule Domain = %v, want [geosite:category-ads]", blockRule.Domain)
	}
}

func TestGenerateXrayConfig_EmptyProfileRouting(t *testing.T) {
	profile := testProfile()
	// No routing rules set

	cfg, err := GenerateXrayConfig(profile, testSettings())
	if err != nil {
		t.Fatalf("GenerateXrayConfig() error = %v", err)
	}

	// Only base rules: api-in + geoip:private
	if len(cfg.Routing.Rules) != 2 {
		t.Errorf("Routing rules count = %d, want 2 (no profile routing)", len(cfg.Routing.Rules))
	}
}

func TestBuildStreamSettings_H2(t *testing.T) {
	server := config.ServerConfig{
		Transport: config.TransportConfig{
			Network: "h2",
			Path:    "/h2path",
			Host:    "example.com,alt.example.com",
		},
		Security: config.SecurityConfig{
			Type: "tls",
			SNI:  "example.com",
		},
	}

	stream := buildStreamSettings(server)

	if stream.Network != "h2" {
		t.Errorf("Network = %q, want h2", stream.Network)
	}
	if stream.H2Settings == nil {
		t.Fatal("H2Settings should not be nil")
	}
	if stream.H2Settings.Path != "/h2path" {
		t.Errorf("H2Settings.Path = %q, want /h2path", stream.H2Settings.Path)
	}
	if len(stream.H2Settings.Host) != 2 {
		t.Fatalf("H2Settings.Host count = %d, want 2", len(stream.H2Settings.Host))
	}
	if stream.H2Settings.Host[0] != "example.com" {
		t.Errorf("H2Settings.Host[0] = %q, want example.com", stream.H2Settings.Host[0])
	}
	if stream.H2Settings.Host[1] != "alt.example.com" {
		t.Errorf("H2Settings.Host[1] = %q, want alt.example.com", stream.H2Settings.Host[1])
	}
	if stream.TLSSettings == nil {
		t.Fatal("TLSSettings should not be nil for h2")
	}
}

func TestBuildStreamSettings_HTTPUpgrade(t *testing.T) {
	server := config.ServerConfig{
		Transport: config.TransportConfig{
			Network: "httpupgrade",
			Path:    "/upgrade",
			Host:    "proxy.example.com",
		},
		Security: config.SecurityConfig{
			Type: "tls",
			SNI:  "proxy.example.com",
		},
	}

	stream := buildStreamSettings(server)

	if stream.Network != "httpupgrade" {
		t.Errorf("Network = %q, want httpupgrade", stream.Network)
	}
	if stream.HTTPUpgradeSettings == nil {
		t.Fatal("HTTPUpgradeSettings should not be nil")
	}
	if stream.HTTPUpgradeSettings.Path != "/upgrade" {
		t.Errorf("HTTPUpgradeSettings.Path = %q, want /upgrade", stream.HTTPUpgradeSettings.Path)
	}
	if stream.HTTPUpgradeSettings.Host != "proxy.example.com" {
		t.Errorf("HTTPUpgradeSettings.Host = %q, want proxy.example.com", stream.HTTPUpgradeSettings.Host)
	}
}

func TestBuildStreamSettings_SplitHTTP(t *testing.T) {
	server := config.ServerConfig{
		Transport: config.TransportConfig{
			Network:              "splithttp",
			Path:                 "/split",
			Host:                 "cdn.example.com",
			MaxConcurrentUploads: 10,
			MaxUploadSize:        1048576,
		},
		Security: config.SecurityConfig{
			Type: "tls",
			SNI:  "cdn.example.com",
		},
	}

	stream := buildStreamSettings(server)

	if stream.Network != "splithttp" {
		t.Errorf("Network = %q, want splithttp", stream.Network)
	}
	if stream.SplitHTTPSettings == nil {
		t.Fatal("SplitHTTPSettings should not be nil")
	}
	if stream.SplitHTTPSettings.Path != "/split" {
		t.Errorf("SplitHTTPSettings.Path = %q, want /split", stream.SplitHTTPSettings.Path)
	}
	if stream.SplitHTTPSettings.Host != "cdn.example.com" {
		t.Errorf("SplitHTTPSettings.Host = %q, want cdn.example.com", stream.SplitHTTPSettings.Host)
	}
	if stream.SplitHTTPSettings.MaxConcurrentUploads != 10 {
		t.Errorf("MaxConcurrentUploads = %d, want 10", stream.SplitHTTPSettings.MaxConcurrentUploads)
	}
	if stream.SplitHTTPSettings.MaxUploadSize != 1048576 {
		t.Errorf("MaxUploadSize = %d, want 1048576", stream.SplitHTTPSettings.MaxUploadSize)
	}
}

func TestBuildStreamSettings_SplitHTTP_NoOptionalFields(t *testing.T) {
	server := config.ServerConfig{
		Transport: config.TransportConfig{
			Network: "splithttp",
			Path:    "/basic",
		},
		Security: config.SecurityConfig{
			Type: "none",
		},
	}

	stream := buildStreamSettings(server)

	if stream.SplitHTTPSettings == nil {
		t.Fatal("SplitHTTPSettings should not be nil")
	}
	if stream.SplitHTTPSettings.MaxConcurrentUploads != 0 {
		t.Errorf("MaxConcurrentUploads = %d, want 0 (default)", stream.SplitHTTPSettings.MaxConcurrentUploads)
	}
	if stream.SplitHTTPSettings.MaxUploadSize != 0 {
		t.Errorf("MaxUploadSize = %d, want 0 (default)", stream.SplitHTTPSettings.MaxUploadSize)
	}
}

func TestGenerateXrayConfig_H2Transport(t *testing.T) {
	profile := &config.Profile{
		Name: "H2 Profile",
		Server: config.ServerConfig{
			Address:    "1.2.3.4",
			Port:       443,
			UUID:       "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
			Encryption: "none",
			Transport: config.TransportConfig{
				Network: "h2",
				Path:    "/h2",
				Host:    "example.com",
			},
			Security: config.SecurityConfig{
				Type: "tls",
				SNI:  "example.com",
			},
		},
	}

	cfg, err := GenerateXrayConfig(profile, testSettings())
	if err != nil {
		t.Fatalf("GenerateXrayConfig() error = %v", err)
	}

	proxy := cfg.Outbounds[0]
	if proxy.StreamSettings.H2Settings == nil {
		t.Fatal("proxy should have H2Settings")
	}
	if proxy.StreamSettings.H2Settings.Path != "/h2" {
		t.Errorf("H2Settings.Path = %q, want /h2", proxy.StreamSettings.H2Settings.Path)
	}

	// Verify valid JSON output
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		t.Fatalf("json.MarshalIndent() error = %v", err)
	}
	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("generated config is not valid JSON: %v", err)
	}
}

func TestGenerateXrayConfig_DoHDNS(t *testing.T) {
	settings := testSettings()
	settings.Local.DNS = []string{"https://dns.google/dns-query", "1.1.1.1"}

	cfg, err := GenerateXrayConfig(testProfile(), settings)
	if err != nil {
		t.Fatalf("GenerateXrayConfig() error = %v", err)
	}

	if len(cfg.DNS.Servers) != 2 {
		t.Fatalf("DNS servers count = %d, want 2", len(cfg.DNS.Servers))
	}

	// First server should be a DNSServerObject (DoH URL)
	dohServer, ok := cfg.DNS.Servers[0].(DNSServerObject)
	if !ok {
		t.Fatalf("DNS server[0] should be DNSServerObject, got %T", cfg.DNS.Servers[0])
	}
	if dohServer.Address != "https://dns.google/dns-query" {
		t.Errorf("DoH address = %q, want https://dns.google/dns-query", dohServer.Address)
	}

	// Second server should be a plain string
	plainServer, ok := cfg.DNS.Servers[1].(string)
	if !ok {
		t.Fatalf("DNS server[1] should be string, got %T", cfg.DNS.Servers[1])
	}
	if plainServer != "1.1.1.1" {
		t.Errorf("plain DNS = %q, want 1.1.1.1", plainServer)
	}

	// Verify JSON marshaling produces valid output
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		t.Fatalf("json.MarshalIndent() error = %v", err)
	}
	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("generated DoH config is not valid JSON: %v", err)
	}

	// Verify dns.servers contains mixed types in JSON
	dnsSection := parsed["dns"].(map[string]interface{})
	servers := dnsSection["servers"].([]interface{})
	if len(servers) != 2 {
		t.Fatalf("JSON dns.servers count = %d, want 2", len(servers))
	}
	// First should be object with "address" key
	if serverObj, ok := servers[0].(map[string]interface{}); ok {
		if serverObj["address"] != "https://dns.google/dns-query" {
			t.Errorf("JSON DoH address = %v, want https://dns.google/dns-query", serverObj["address"])
		}
	} else {
		t.Errorf("JSON dns.servers[0] should be object, got %T", servers[0])
	}
	// Second should be plain string
	if servers[1] != "1.1.1.1" {
		t.Errorf("JSON dns.servers[1] = %v, want 1.1.1.1", servers[1])
	}
}

func TestGenerateXrayConfig_DoTDNS(t *testing.T) {
	settings := testSettings()
	settings.Local.DNS = []string{"tcp://dns.google:853", "8.8.8.8"}

	cfg, err := GenerateXrayConfig(testProfile(), settings)
	if err != nil {
		t.Fatalf("GenerateXrayConfig() error = %v", err)
	}

	if len(cfg.DNS.Servers) != 2 {
		t.Fatalf("DNS servers count = %d, want 2", len(cfg.DNS.Servers))
	}

	// DoT server should be a DNSServerObject
	dotServer, ok := cfg.DNS.Servers[0].(DNSServerObject)
	if !ok {
		t.Fatalf("DNS server[0] should be DNSServerObject for DoT, got %T", cfg.DNS.Servers[0])
	}
	if dotServer.Address != "tcp://dns.google:853" {
		t.Errorf("DoT address = %q, want tcp://dns.google:853", dotServer.Address)
	}
}

func TestGenerateXrayConfig_DoHLocal(t *testing.T) {
	settings := testSettings()
	settings.Local.DNS = []string{"https+local://dns.google/dns-query"}

	cfg, err := GenerateXrayConfig(testProfile(), settings)
	if err != nil {
		t.Fatalf("GenerateXrayConfig() error = %v", err)
	}

	if len(cfg.DNS.Servers) != 1 {
		t.Fatalf("DNS servers count = %d, want 1", len(cfg.DNS.Servers))
	}

	dohLocal, ok := cfg.DNS.Servers[0].(DNSServerObject)
	if !ok {
		t.Fatalf("DNS server[0] should be DNSServerObject, got %T", cfg.DNS.Servers[0])
	}
	if dohLocal.Address != "https+local://dns.google/dns-query" {
		t.Errorf("DoH local address = %q, want https+local://dns.google/dns-query", dohLocal.Address)
	}
}

func TestGenerateXrayConfig_DNSRouting(t *testing.T) {
	settings := testSettings()
	profile := testProfile()
	profile.Routing.DNSRules = []config.DNSRule{
		{
			Server:  "https://dns.google/dns-query",
			Domains: []string{"domain:google.com", "domain:youtube.com"},
		},
		{
			Server:    "1.1.1.1",
			Domains:   []string{"geosite:private"},
			ExpectIPs: []string{"geoip:private"},
		},
	}

	cfg, err := GenerateXrayConfig(profile, settings)
	if err != nil {
		t.Fatalf("GenerateXrayConfig() error = %v", err)
	}

	// Default DNS (2) + DNS rules (2) = 4
	if len(cfg.DNS.Servers) != 4 {
		t.Fatalf("DNS servers count = %d, want 4", len(cfg.DNS.Servers))
	}

	// Check the DNS routing rule entries
	rule1, ok := cfg.DNS.Servers[2].(DNSServerObject)
	if !ok {
		t.Fatalf("DNS server[2] should be DNSServerObject, got %T", cfg.DNS.Servers[2])
	}
	if rule1.Address != "https://dns.google/dns-query" {
		t.Errorf("DNS rule[0] address = %q, want https://dns.google/dns-query", rule1.Address)
	}
	if len(rule1.Domains) != 2 {
		t.Errorf("DNS rule[0] domains count = %d, want 2", len(rule1.Domains))
	}

	rule2, ok := cfg.DNS.Servers[3].(DNSServerObject)
	if !ok {
		t.Fatalf("DNS server[3] should be DNSServerObject, got %T", cfg.DNS.Servers[3])
	}
	if rule2.Address != "1.1.1.1" {
		t.Errorf("DNS rule[1] address = %q, want 1.1.1.1", rule2.Address)
	}
	if len(rule2.ExpectIPs) != 1 || rule2.ExpectIPs[0] != "geoip:private" {
		t.Errorf("DNS rule[1] expectIPs = %v, want [geoip:private]", rule2.ExpectIPs)
	}

	// Verify JSON output
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		t.Fatalf("json.MarshalIndent() error = %v", err)
	}
	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("generated DNS routing config is not valid JSON: %v", err)
	}
}

func TestIsAdvancedDNS(t *testing.T) {
	tests := []struct {
		server string
		want   bool
	}{
		{"1.1.1.1", false},
		{"8.8.8.8", false},
		{"https://dns.google/dns-query", true},
		{"https+local://dns.google/dns-query", true},
		{"tcp://dns.google:853", true},
		{"localhost", false},
	}

	for _, tc := range tests {
		t.Run(tc.server, func(t *testing.T) {
			if got := isAdvancedDNS(tc.server); got != tc.want {
				t.Errorf("isAdvancedDNS(%q) = %v, want %v", tc.server, got, tc.want)
			}
		})
	}
}

func TestBuildHysteria2Outbound(t *testing.T) {
	server := config.ServerConfig{
		Address: "example.com", Port: 29347, UUID: "secretauth",
		Protocol: "hysteria2", Obfs: "salamander", ObfsPassword: "obfspw",
		Transport: config.TransportConfig{Network: "hysteria"},
		Security: config.SecurityConfig{
			Type: "tls", SNI: "real.example.com", ALPN: "h3",
			Fingerprint: "chrome", AllowInsecure: true,
		},
	}
	ob := buildOutbound("proxy", server, "")
	if ob.Protocol != "hysteria" {
		t.Fatalf("Protocol = %q, want hysteria", ob.Protocol)
	}
	if ob.StreamSettings == nil || ob.StreamSettings.HysteriaSettings == nil {
		t.Fatal("missing hysteriaSettings")
	}
	if ob.StreamSettings.HysteriaSettings.Auth != "secretauth" {
		t.Errorf("auth = %q", ob.StreamSettings.HysteriaSettings.Auth)
	}
	fm := ob.StreamSettings.FinalMask
	if fm == nil || len(fm.UDP) != 1 || fm.UDP[0].Type != "salamander" ||
		fm.UDP[0].Settings.Password != "obfspw" {
		t.Errorf("finalmask salamander wrong: %+v", fm)
	}
	if ob.StreamSettings.TLSSettings == nil || !ob.StreamSettings.TLSSettings.AllowInsecure {
		t.Error("tlsSettings.allowInsecure not set")
	}
	data, _ := json.Marshal(ob)
	for _, want := range []string{`"protocol":"hysteria"`, `"finalmask"`, `"hysteriaSettings"`, `"allowInsecure":true`} {
		if !strings.Contains(string(data), want) {
			t.Errorf("JSON missing %s: %s", want, data)
		}
	}
}

func TestValidateProfile_Hysteria2(t *testing.T) {
	good := &config.Profile{
		Name: "hy2",
		Server: config.ServerConfig{
			Address: "h", Port: 443, UUID: "auth", Protocol: "hysteria2",
			Transport: config.TransportConfig{Network: "hysteria"},
			Security:  config.SecurityConfig{Type: "tls"},
		},
	}
	if err := ValidateProfile(good); err != nil {
		t.Errorf("valid hy2 profile rejected: %v", err)
	}
	bad := *good
	bad.Server.UUID = ""
	err := ValidateProfile(&bad)
	if err == nil || !strings.Contains(err.Error(), "hysteria2 auth is empty") {
		t.Errorf("want 'hysteria2 auth is empty' error, got %v", err)
	}
}

func TestValidateProfile_Hysteria2_Obfs(t *testing.T) {
	p := &config.Profile{Name: "x", Server: config.ServerConfig{
		Address: "h", Port: 443, UUID: "a", Protocol: "hysteria2",
		Obfs: "rot13", Transport: config.TransportConfig{Network: "hysteria"},
	}}
	if err := ValidateProfile(p); err == nil {
		t.Fatal("expected error for unsupported obfs")
	}
}

func TestValidateProfile_Hysteria2_BadPin(t *testing.T) {
	p := &config.Profile{Name: "x", Server: config.ServerConfig{
		Address: "h", Port: 443, UUID: "a", Protocol: "hysteria2",
		Transport: config.TransportConfig{Network: "hysteria"},
		Security:  config.SecurityConfig{Type: "tls", PinSHA256: "zzz"},
	}}
	if err := ValidateProfile(p); err == nil {
		t.Fatal("expected error for malformed pinSHA256")
	}
}

func TestValidateProfile_Hysteria2_BadPortHop(t *testing.T) {
	p := &config.Profile{Name: "x", Server: config.ServerConfig{
		Address: "h", Port: 443, UUID: "a", Protocol: "hysteria2",
		PortHopping: "443,99999", Transport: config.TransportConfig{Network: "hysteria"},
	}}
	if err := ValidateProfile(p); err == nil {
		t.Fatal("expected error for out-of-range port-hopping port")
	}
}

func TestValidateProfile_Hysteria2_ReversedRange(t *testing.T) {
	p := &config.Profile{Name: "x", Server: config.ServerConfig{
		Address: "h", Port: 443, UUID: "a", Protocol: "hysteria2",
		PortHopping: "6000-5000", Transport: config.TransportConfig{Network: "hysteria"},
	}}
	if err := ValidateProfile(p); err == nil {
		t.Fatal("expected error for reversed port-hopping range (start > end)")
	}
}

func TestBuildHysteria2Outbound_Pinned(t *testing.T) {
	const pin = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	srv := config.ServerConfig{
		Address: "h", Port: 443, UUID: "auth", Protocol: "hysteria2",
		Transport: config.TransportConfig{Network: "hysteria"},
		Security:  config.SecurityConfig{Type: "tls", SNI: "h", AllowInsecure: true, PinSHA256: pin},
	}
	ob := buildHysteria2Outbound("proxy", srv, "")
	tls := ob.StreamSettings.TLSSettings
	if tls == nil || tls.PinnedPeerCertSha256 != pin {
		t.Fatalf("pinnedPeerCertSha256 = %+v, want %s", tls, pin)
	}
	if tls.AllowInsecure {
		t.Error("allowInsecure must be false when pinning is set")
	}
}

func TestBuildHysteria2Outbound_PortHopping(t *testing.T) {
	srv := config.ServerConfig{
		Address: "h", Port: 443, UUID: "auth", Protocol: "hysteria2",
		PortHopping: "443,5000-6000",
		Transport:   config.TransportConfig{Network: "hysteria"},
		Security:    config.SecurityConfig{Type: "tls", SNI: "h"},
	}
	ob := buildHysteria2Outbound("proxy", srv, "")
	fm := ob.StreamSettings.FinalMask
	if fm == nil || fm.QuicParams == nil || fm.QuicParams.UdpHop == nil {
		t.Fatalf("finalmask.quicParams.udpHop missing: %+v", fm)
	}
	if fm.QuicParams.UdpHop.Ports != "443,5000-6000" {
		t.Errorf("udpHop.ports = %q, want 443,5000-6000", fm.QuicParams.UdpHop.Ports)
	}
}

func TestNewOutbound_ProxyTagTail(t *testing.T) {
	srv := config.ServerConfig{Address: "h", Port: 1, Protocol: "vless", Transport: config.TransportConfig{Network: "tcp"}}
	with := newOutbound("proxy", "vless", json.RawMessage(`{}`), srv, "chain-1")
	if with.ProxySettings == nil || with.ProxySettings.Tag != "chain-1" {
		t.Fatalf("proxyTag not wired: %+v", with.ProxySettings)
	}
	without := newOutbound("proxy", "vless", json.RawMessage(`{}`), srv, "")
	if without.ProxySettings != nil {
		t.Fatalf("empty proxyTag must leave ProxySettings nil")
	}
	if without.Tag != "proxy" || without.Protocol != "vless" || without.StreamSettings == nil {
		t.Fatalf("outbound base fields wrong: %+v", without)
	}
}

func TestValidateProfile_TLSPin(t *testing.T) {
	const validPin = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"  // 64 hex
	const shortPin = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcde"   // 63 hex
	const nonHexPin = "zz23456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef" // 64 chars, not hex

	base := func(proto, pin string) *config.Profile {
		return &config.Profile{
			Name: "tls-pin",
			Server: config.ServerConfig{
				Address:   "h",
				Port:      443,
				UUID:      "id",
				Protocol:  proto,
				Transport: config.TransportConfig{Network: "tcp"},
				Security:  config.SecurityConfig{Type: "tls", SNI: "h", PinSHA256: pin},
			},
		}
	}

	tests := []struct {
		name    string
		profile *config.Profile
		wantErr bool
	}{
		{"vless valid pin", base("vless", validPin), false},
		{"vmess valid pin", base("vmess", validPin), false},
		{"vless empty pin", base("vless", ""), false},
		{"vless short pin", base("vless", shortPin), true},
		{"vmess short pin", base("vmess", shortPin), true},
		{"vless non-hex pin", base("vless", nonHexPin), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateProfile(tt.profile)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("ValidateProfile(%s) = nil, want error", tt.name)
				}
				if !strings.Contains(err.Error(), "pinSHA256") {
					t.Errorf("error = %v, want it to mention pinSHA256", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("ValidateProfile(%s) = %v, want nil", tt.name, err)
			}
		})
	}
}
