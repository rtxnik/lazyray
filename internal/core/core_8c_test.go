package core

import (
	"encoding/json"
	"testing"

	"github.com/rtxnik/lazyray/internal/config"
)

// --- VMess config generation tests ---

func TestGenerateXrayConfig_VMessOutbound(t *testing.T) {
	profile := &config.Profile{
		Name: "VMess Test",
		Server: config.ServerConfig{
			Address:    "1.2.3.4",
			Port:       443,
			UUID:       "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
			Encryption: "auto",
			Protocol:   "vmess",
			AlterID:    0,
			Transport: config.TransportConfig{
				Network: "ws",
				Path:    "/vmess",
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
	if proxy.Protocol != "vmess" {
		t.Errorf("Protocol = %q, want vmess", proxy.Protocol)
	}

	// Verify VMess settings structure
	var settings map[string]interface{}
	if err := json.Unmarshal(proxy.Settings, &settings); err != nil {
		t.Fatalf("unmarshal settings: %v", err)
	}
	vnext, ok := settings["vnext"].([]interface{})
	if !ok || len(vnext) == 0 {
		t.Fatal("vmess settings should contain vnext array")
	}
	server := vnext[0].(map[string]interface{})
	if server["address"] != "1.2.3.4" {
		t.Errorf("address = %v, want 1.2.3.4", server["address"])
	}
	users := server["users"].([]interface{})
	user := users[0].(map[string]interface{})
	if user["security"] != "auto" {
		t.Errorf("security = %v, want auto", user["security"])
	}

	// Verify stream settings
	if proxy.StreamSettings.Network != "ws" {
		t.Errorf("Network = %q, want ws", proxy.StreamSettings.Network)
	}
	if proxy.StreamSettings.WSSettings == nil {
		t.Fatal("WSSettings should not be nil")
	}
	if proxy.StreamSettings.WSSettings.Path != "/vmess" {
		t.Errorf("WSSettings.Path = %q, want /vmess", proxy.StreamSettings.WSSettings.Path)
	}

	// Verify valid JSON output
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		t.Fatalf("json.MarshalIndent() error = %v", err)
	}
	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("not valid JSON: %v", err)
	}
}

func TestGenerateXrayConfig_VMessDefaultEncryption(t *testing.T) {
	profile := &config.Profile{
		Name: "VMess No Enc",
		Server: config.ServerConfig{
			Address:    "1.2.3.4",
			Port:       443,
			UUID:       "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
			Encryption: "", // empty → should default to "auto"
			Protocol:   "vmess",
			Transport:  config.TransportConfig{Network: "tcp"},
			Security:   config.SecurityConfig{Type: "none"},
		},
	}

	cfg, err := GenerateXrayConfig(profile, testSettings())
	if err != nil {
		t.Fatalf("GenerateXrayConfig() error = %v", err)
	}

	var settings map[string]interface{}
	if err := json.Unmarshal(cfg.Outbounds[0].Settings, &settings); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	vnext := settings["vnext"].([]interface{})
	users := vnext[0].(map[string]interface{})["users"].([]interface{})
	user := users[0].(map[string]interface{})
	if user["security"] != "auto" {
		t.Errorf("security = %v, want auto (default)", user["security"])
	}
}

// --- Trojan config generation tests ---

func TestGenerateXrayConfig_TrojanOutbound(t *testing.T) {
	profile := &config.Profile{
		Name: "Trojan Test",
		Server: config.ServerConfig{
			Address:  "5.6.7.8",
			Port:     443,
			UUID:     "trojanpassword",
			Protocol: "trojan",
			Transport: config.TransportConfig{
				Network: "tcp",
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
	if proxy.Protocol != "trojan" {
		t.Errorf("Protocol = %q, want trojan", proxy.Protocol)
	}

	// Verify Trojan settings structure (uses "servers" not "vnext")
	var settings map[string]interface{}
	if err := json.Unmarshal(proxy.Settings, &settings); err != nil {
		t.Fatalf("unmarshal settings: %v", err)
	}
	servers, ok := settings["servers"].([]interface{})
	if !ok || len(servers) == 0 {
		t.Fatal("trojan settings should contain servers array")
	}
	server := servers[0].(map[string]interface{})
	if server["address"] != "5.6.7.8" {
		t.Errorf("address = %v, want 5.6.7.8", server["address"])
	}
	if server["password"] != "trojanpassword" {
		t.Errorf("password = %v, want trojanpassword", server["password"])
	}
	if int(server["port"].(float64)) != 443 {
		t.Errorf("port = %v, want 443", server["port"])
	}

	// Verify TLS
	if proxy.StreamSettings.Security != "tls" {
		t.Errorf("Security = %q, want tls", proxy.StreamSettings.Security)
	}
	if proxy.StreamSettings.TLSSettings == nil {
		t.Fatal("TLSSettings should not be nil")
	}
	if proxy.StreamSettings.TLSSettings.ServerName != "example.com" {
		t.Errorf("ServerName = %q, want example.com", proxy.StreamSettings.TLSSettings.ServerName)
	}
}

// --- Shadowsocks config generation with full JSON validation ---

func TestGenerateXrayConfig_ShadowsocksOutbound_FullJSON(t *testing.T) {
	profile := &config.Profile{
		Name: "SS Full",
		Server: config.ServerConfig{
			Address:    "10.0.0.1",
			Port:       8388,
			UUID:       "mypass123",
			Encryption: "chacha20-ietf-poly1305",
			Protocol:   "shadowsocks",
			Transport:  config.TransportConfig{Network: "tcp"},
			Security:   config.SecurityConfig{Type: "none"},
		},
	}

	cfg, err := GenerateXrayConfig(profile, testSettings())
	if err != nil {
		t.Fatalf("GenerateXrayConfig() error = %v", err)
	}

	proxy := cfg.Outbounds[0]
	if proxy.Protocol != "shadowsocks" {
		t.Errorf("Protocol = %q, want shadowsocks", proxy.Protocol)
	}

	var settings map[string]interface{}
	if err := json.Unmarshal(proxy.Settings, &settings); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	servers := settings["servers"].([]interface{})
	server := servers[0].(map[string]interface{})
	if server["method"] != "chacha20-ietf-poly1305" {
		t.Errorf("method = %v, want chacha20-ietf-poly1305", server["method"])
	}
	if server["password"] != "mypass123" {
		t.Errorf("password = %v, want mypass123", server["password"])
	}

	// Verify full JSON validity
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		t.Fatalf("json.MarshalIndent() error = %v", err)
	}
	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("not valid JSON: %v", err)
	}
}

// --- Transport URL parsing tests ---

func TestParseVLESS_H2Transport(t *testing.T) {
	rawURL := "vless://uuid@1.2.3.4:443?type=h2&path=/h2path&host=example.com&security=tls&sni=example.com&fp=chrome#H2+Test"

	p, err := ParseVLESS(rawURL)
	if err != nil {
		t.Fatalf("ParseVLESS() error = %v", err)
	}

	if p.Server.Transport.Network != "h2" {
		t.Errorf("Network = %q, want h2", p.Server.Transport.Network)
	}
	if p.Server.Transport.Path != "/h2path" {
		t.Errorf("Path = %q, want /h2path", p.Server.Transport.Path)
	}
	if p.Server.Transport.Host != "example.com" {
		t.Errorf("Host = %q, want example.com", p.Server.Transport.Host)
	}
}

func TestParseVLESS_HTTPUpgradeTransport(t *testing.T) {
	rawURL := "vless://uuid@1.2.3.4:443?type=httpupgrade&path=/upgrade&host=proxy.example.com&security=tls&sni=proxy.example.com&fp=chrome#HTTPUpgrade+Test"

	p, err := ParseVLESS(rawURL)
	if err != nil {
		t.Fatalf("ParseVLESS() error = %v", err)
	}

	if p.Server.Transport.Network != "httpupgrade" {
		t.Errorf("Network = %q, want httpupgrade", p.Server.Transport.Network)
	}
	if p.Server.Transport.Path != "/upgrade" {
		t.Errorf("Path = %q, want /upgrade", p.Server.Transport.Path)
	}
	if p.Server.Transport.Host != "proxy.example.com" {
		t.Errorf("Host = %q, want proxy.example.com", p.Server.Transport.Host)
	}
}

func TestParseVLESS_SplitHTTPTransport(t *testing.T) {
	rawURL := "vless://uuid@1.2.3.4:443?type=splithttp&path=/split&host=cdn.example.com&security=tls&sni=cdn.example.com&fp=chrome#SplitHTTP+Test"

	p, err := ParseVLESS(rawURL)
	if err != nil {
		t.Fatalf("ParseVLESS() error = %v", err)
	}

	if p.Server.Transport.Network != "splithttp" {
		t.Errorf("Network = %q, want splithttp", p.Server.Transport.Network)
	}
	if p.Server.Transport.Path != "/split" {
		t.Errorf("Path = %q, want /split", p.Server.Transport.Path)
	}
	if p.Server.Transport.Host != "cdn.example.com" {
		t.Errorf("Host = %q, want cdn.example.com", p.Server.Transport.Host)
	}
}

// --- Full xray config gen for each new transport ---

func TestGenerateXrayConfig_HTTPUpgradeTransport(t *testing.T) {
	profile := &config.Profile{
		Name: "HU Profile",
		Server: config.ServerConfig{
			Address:    "1.2.3.4",
			Port:       443,
			UUID:       "uuid",
			Encryption: "none",
			Transport: config.TransportConfig{
				Network: "httpupgrade",
				Path:    "/up",
				Host:    "proxy.example.com",
			},
			Security: config.SecurityConfig{
				Type: "tls",
				SNI:  "proxy.example.com",
			},
		},
	}

	cfg, err := GenerateXrayConfig(profile, testSettings())
	if err != nil {
		t.Fatalf("GenerateXrayConfig() error = %v", err)
	}

	proxy := cfg.Outbounds[0]
	if proxy.StreamSettings.HTTPUpgradeSettings == nil {
		t.Fatal("HTTPUpgradeSettings should not be nil")
	}
	if proxy.StreamSettings.HTTPUpgradeSettings.Path != "/up" {
		t.Errorf("Path = %q, want /up", proxy.StreamSettings.HTTPUpgradeSettings.Path)
	}
	if proxy.StreamSettings.HTTPUpgradeSettings.Host != "proxy.example.com" {
		t.Errorf("Host = %q, want proxy.example.com", proxy.StreamSettings.HTTPUpgradeSettings.Host)
	}

	// Verify valid JSON
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		t.Fatalf("json.MarshalIndent() error = %v", err)
	}
	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("not valid JSON: %v", err)
	}
}

func TestGenerateXrayConfig_SplitHTTPTransport(t *testing.T) {
	profile := &config.Profile{
		Name: "SH Profile",
		Server: config.ServerConfig{
			Address:    "1.2.3.4",
			Port:       443,
			UUID:       "uuid",
			Encryption: "none",
			Transport: config.TransportConfig{
				Network:              "splithttp",
				Path:                 "/split",
				Host:                 "cdn.example.com",
				MaxConcurrentUploads: 5,
				MaxUploadSize:        524288,
			},
			Security: config.SecurityConfig{
				Type: "tls",
				SNI:  "cdn.example.com",
			},
		},
	}

	cfg, err := GenerateXrayConfig(profile, testSettings())
	if err != nil {
		t.Fatalf("GenerateXrayConfig() error = %v", err)
	}

	proxy := cfg.Outbounds[0]
	if proxy.StreamSettings.SplitHTTPSettings == nil {
		t.Fatal("SplitHTTPSettings should not be nil")
	}
	if proxy.StreamSettings.SplitHTTPSettings.MaxConcurrentUploads != 5 {
		t.Errorf("MaxConcurrentUploads = %d, want 5", proxy.StreamSettings.SplitHTTPSettings.MaxConcurrentUploads)
	}
	if proxy.StreamSettings.SplitHTTPSettings.MaxUploadSize != 524288 {
		t.Errorf("MaxUploadSize = %d, want 524288", proxy.StreamSettings.SplitHTTPSettings.MaxUploadSize)
	}

	// Verify valid JSON
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		t.Fatalf("json.MarshalIndent() error = %v", err)
	}
	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("not valid JSON: %v", err)
	}
}

// --- ValidateProfile tests ---

func TestValidateProfile_ShadowsocksValid(t *testing.T) {
	profile := &config.Profile{
		Name: "SS Valid",
		Server: config.ServerConfig{
			Address:    "1.2.3.4",
			Port:       8388,
			UUID:       "password",
			Encryption: "aes-256-gcm",
			Protocol:   "shadowsocks",
			Transport:  config.TransportConfig{Network: "tcp"},
		},
	}

	if err := ValidateProfile(profile); err != nil {
		t.Errorf("valid SS profile should pass: %v", err)
	}
}

func TestValidateProfile_ShadowsocksEmptyPassword(t *testing.T) {
	profile := &config.Profile{
		Name: "SS Bad",
		Server: config.ServerConfig{
			Address:    "1.2.3.4",
			Port:       8388,
			UUID:       "",
			Encryption: "aes-256-gcm",
			Protocol:   "shadowsocks",
			Transport:  config.TransportConfig{Network: "tcp"},
		},
	}

	err := ValidateProfile(profile)
	if err == nil {
		t.Fatal("expected error for empty SS password")
	}
}

func TestValidateProfile_ShadowsocksEmptyMethod(t *testing.T) {
	profile := &config.Profile{
		Name: "SS No Method",
		Server: config.ServerConfig{
			Address:    "1.2.3.4",
			Port:       8388,
			UUID:       "password",
			Encryption: "",
			Protocol:   "shadowsocks",
			Transport:  config.TransportConfig{Network: "tcp"},
		},
	}

	err := ValidateProfile(profile)
	if err == nil {
		t.Fatal("expected error for empty SS encryption method")
	}
}

func TestValidateProfile_TrojanValid(t *testing.T) {
	profile := &config.Profile{
		Name: "Trojan Valid",
		Server: config.ServerConfig{
			Address:  "1.2.3.4",
			Port:     443,
			UUID:     "password",
			Protocol: "trojan",
			Transport: config.TransportConfig{
				Network: "tcp",
			},
			Security: config.SecurityConfig{
				Type: "tls",
			},
		},
	}

	if err := ValidateProfile(profile); err != nil {
		t.Errorf("valid Trojan profile should pass: %v", err)
	}
}

func TestValidateProfile_TrojanEmptyPassword(t *testing.T) {
	profile := &config.Profile{
		Name: "Trojan Bad",
		Server: config.ServerConfig{
			Address:   "1.2.3.4",
			Port:      443,
			UUID:      "",
			Protocol:  "trojan",
			Transport: config.TransportConfig{Network: "tcp"},
			Security:  config.SecurityConfig{Type: "tls"},
		},
	}

	err := ValidateProfile(profile)
	if err == nil {
		t.Fatal("expected error for empty Trojan password")
	}
}

func TestValidateProfile_InvalidPort_Extended(t *testing.T) {
	tests := []struct {
		name string
		port int
	}{
		{"zero port", 0},
		{"negative port", -1},
		{"over 65535", 70000},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			profile := &config.Profile{
				Name: "Test",
				Server: config.ServerConfig{
					Address:    "1.2.3.4",
					Port:       tc.port,
					UUID:       "uuid",
					Encryption: "none",
					Transport:  config.TransportConfig{Network: "tcp"},
				},
			}
			if err := ValidateProfile(profile); err == nil {
				t.Errorf("expected error for port %d", tc.port)
			}
		})
	}
}

// --- anyToInt tests (consolidated JSON port/int coercion) ---

func TestParseAnyPort(t *testing.T) {
	tests := []struct {
		name string
		val  interface{}
		want int
	}{
		{"float64", float64(8443), 8443},
		{"string", "443", 443},
		{"json.Number", json.Number("1234"), 1234},
		{"nil", nil, 0},
		{"bool", true, 0},
		{"empty string", "", 0},
		{"invalid string", "abc", 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := anyToInt(tc.val)
			if got != tc.want {
				t.Errorf("anyToInt(%v) = %d, want %d", tc.val, got, tc.want)
			}
		})
	}
}

func TestParseAnyInt(t *testing.T) {
	tests := []struct {
		name string
		val  interface{}
		want int
	}{
		{"float64", float64(42), 42},
		{"string zero", "0", 0},
		{"string positive", "100", 100},
		{"json.Number", json.Number("64"), 64},
		{"nil", nil, 0},
		{"bool false", false, 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := anyToInt(tc.val)
			if got != tc.want {
				t.Errorf("anyToInt(%v) = %d, want %d", tc.val, got, tc.want)
			}
		})
	}
}

// --- buildStreamSettings edge cases ---

func TestBuildStreamSettings_WS_WithHost(t *testing.T) {
	server := config.ServerConfig{
		Transport: config.TransportConfig{
			Network: "ws",
			Path:    "/websocket",
			Host:    "ws.example.com",
		},
		Security: config.SecurityConfig{Type: "tls", SNI: "ws.example.com"},
	}

	stream := buildStreamSettings(server)
	if stream.WSSettings == nil {
		t.Fatal("WSSettings should not be nil")
	}
	if stream.WSSettings.Path != "/websocket" {
		t.Errorf("Path = %q, want /websocket", stream.WSSettings.Path)
	}
	if stream.WSSettings.Headers == nil || stream.WSSettings.Headers["Host"] != "ws.example.com" {
		t.Errorf("Host header = %v, want ws.example.com", stream.WSSettings.Headers)
	}
}

func TestBuildStreamSettings_GRPC(t *testing.T) {
	server := config.ServerConfig{
		Transport: config.TransportConfig{
			Network: "grpc",
			Path:    "myservice",
		},
		Security: config.SecurityConfig{Type: "tls", SNI: "grpc.example.com"},
	}

	stream := buildStreamSettings(server)
	if stream.GRPCSettings == nil {
		t.Fatal("GRPCSettings should not be nil")
	}
	if stream.GRPCSettings.ServiceName != "myservice" {
		t.Errorf("ServiceName = %q, want myservice", stream.GRPCSettings.ServiceName)
	}
}

func TestBuildStreamSettings_TLS_WithALPN(t *testing.T) {
	server := config.ServerConfig{
		Transport: config.TransportConfig{Network: "tcp"},
		Security: config.SecurityConfig{
			Type:        "tls",
			SNI:         "example.com",
			Fingerprint: "chrome",
			ALPN:        "h2,http/1.1",
		},
	}

	stream := buildStreamSettings(server)
	if stream.TLSSettings == nil {
		t.Fatal("TLSSettings should not be nil")
	}
	if len(stream.TLSSettings.ALPN) != 2 {
		t.Fatalf("ALPN count = %d, want 2", len(stream.TLSSettings.ALPN))
	}
	if stream.TLSSettings.ALPN[0] != "h2" {
		t.Errorf("ALPN[0] = %q, want h2", stream.TLSSettings.ALPN[0])
	}
	if stream.TLSSettings.ALPN[1] != "http/1.1" {
		t.Errorf("ALPN[1] = %q, want http/1.1", stream.TLSSettings.ALPN[1])
	}
}

func TestBuildStreamSettings_H2_SingleHost(t *testing.T) {
	server := config.ServerConfig{
		Transport: config.TransportConfig{
			Network: "h2",
			Path:    "/h2",
			Host:    "single.example.com",
		},
		Security: config.SecurityConfig{Type: "tls", SNI: "single.example.com"},
	}

	stream := buildStreamSettings(server)
	if stream.H2Settings == nil {
		t.Fatal("H2Settings should not be nil")
	}
	if len(stream.H2Settings.Host) != 1 {
		t.Fatalf("Host count = %d, want 1", len(stream.H2Settings.Host))
	}
	if stream.H2Settings.Host[0] != "single.example.com" {
		t.Errorf("Host[0] = %q, want single.example.com", stream.H2Settings.Host[0])
	}
}

func TestBuildStreamSettings_H2_EmptyHost(t *testing.T) {
	server := config.ServerConfig{
		Transport: config.TransportConfig{
			Network: "h2",
			Path:    "/h2",
		},
		Security: config.SecurityConfig{Type: "tls"},
	}

	stream := buildStreamSettings(server)
	if stream.H2Settings == nil {
		t.Fatal("H2Settings should not be nil")
	}
	if len(stream.H2Settings.Host) != 0 {
		t.Errorf("Host should be empty when not set, got %v", stream.H2Settings.Host)
	}
}

// --- Multi-hop with different protocols ---

func TestGenerateXrayConfig_MultiHop_MixedProtocols(t *testing.T) {
	profile := &config.Profile{
		Name: "Mixed Chain",
		Server: config.ServerConfig{
			Address:    "1.2.3.4",
			Port:       443,
			UUID:       "uuid-entry",
			Encryption: "none",
			Transport: config.TransportConfig{
				Network: "xhttp",
				Path:    "/x",
				Mode:    "auto",
			},
			Security: config.SecurityConfig{
				Type:        "reality",
				SNI:         "example.com",
				Fingerprint: "chrome",
				PublicKey:   "testpk",
				ShortID:     "1234",
			},
		},
		Chain: []config.ServerConfig{
			{
				Address:    "5.6.7.8",
				Port:       443,
				UUID:       "uuid-exit",
				Encryption: "none",
				Transport: config.TransportConfig{
					Network: "xhttp",
					Path:    "/y",
					Mode:    "auto",
				},
				Security: config.SecurityConfig{
					Type:        "reality",
					SNI:         "exit.com",
					Fingerprint: "chrome",
					PublicKey:   "exitpk",
					ShortID:     "5678",
				},
			},
		},
	}

	cfg, err := GenerateXrayConfig(profile, testSettings())
	if err != nil {
		t.Fatalf("GenerateXrayConfig() error = %v", err)
	}

	// Should have: hop-0, proxy (exit), direct, block = 4 outbounds
	if len(cfg.Outbounds) != 4 {
		t.Fatalf("Outbounds count = %d, want 4", len(cfg.Outbounds))
	}

	if cfg.Outbounds[0].Tag != "hop-0" {
		t.Errorf("Outbound[0].Tag = %q, want hop-0", cfg.Outbounds[0].Tag)
	}
	if cfg.Outbounds[1].Tag != "proxy" {
		t.Errorf("Outbound[1].Tag = %q, want proxy", cfg.Outbounds[1].Tag)
	}
	if cfg.Outbounds[1].ProxySettings == nil || cfg.Outbounds[1].ProxySettings.Tag != "hop-0" {
		t.Error("exit outbound should reference hop-0 via proxySettings")
	}
}

// --- ToTrojanURL edge cases ---

func TestToTrojanURL_WithTransport(t *testing.T) {
	p := &config.Profile{
		Name: "Trojan WS",
		Server: config.ServerConfig{
			Address:  "1.2.3.4",
			Port:     443,
			UUID:     "pass",
			Protocol: "trojan",
			Transport: config.TransportConfig{
				Network: "ws",
				Path:    "/ws",
				Host:    "example.com",
			},
			Security: config.SecurityConfig{
				Type:        "tls",
				SNI:         "example.com",
				Fingerprint: "chrome",
			},
		},
	}

	url := ToTrojanURL(p)
	if url == "" {
		t.Fatal("ToTrojanURL() should return non-empty string")
	}

	// Parse it back to verify roundtrip
	p2, err := ParseTrojan(url)
	if err != nil {
		t.Fatalf("ParseTrojan(exported) error = %v", err)
	}
	if p2.Server.Address != "1.2.3.4" {
		t.Errorf("Address = %q, want 1.2.3.4", p2.Server.Address)
	}
	if p2.Server.Transport.Network != "ws" {
		t.Errorf("Network = %q, want ws", p2.Server.Transport.Network)
	}
}

// --- buildOutbound dispatch tests ---

func TestBuildOutbound_AllProtocols(t *testing.T) {
	tests := []struct {
		name     string
		protocol string
		want     string
	}{
		{"vless", "", "vless"},
		{"vmess", "vmess", "vmess"},
		{"trojan", "trojan", "trojan"},
		{"shadowsocks", "shadowsocks", "shadowsocks"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			server := config.ServerConfig{
				Address:    "1.2.3.4",
				Port:       443,
				UUID:       "test",
				Encryption: "none",
				Protocol:   tc.protocol,
				Transport:  config.TransportConfig{Network: "tcp"},
				Security:   config.SecurityConfig{Type: "none"},
			}
			if tc.protocol == "shadowsocks" {
				server.Encryption = "aes-256-gcm"
			}

			ob := buildOutbound("proxy", server, "")
			if ob.Protocol != tc.want {
				t.Errorf("Protocol = %q, want %q", ob.Protocol, tc.want)
			}
		})
	}
}

// --- isIPEntry tests ---

func TestIsIPEntry_Extended(t *testing.T) {
	tests := []struct {
		s    string
		want bool
	}{
		{"10.0.0.0/8", true},
		{"192.168.1.1", true},
		{"::1", true},
		{"geoip:private", true},
		{"domain:example.com", false},
		{"geosite:category-ads", false},
		{"regexp:.*\\.cn$", false},
		{"full:example.com", false},
		{"keyword:ads", false},
	}

	for _, tc := range tests {
		t.Run(tc.s, func(t *testing.T) {
			if got := isIPEntry(tc.s); got != tc.want {
				t.Errorf("isIPEntry(%q) = %v, want %v", tc.s, got, tc.want)
			}
		})
	}
}
