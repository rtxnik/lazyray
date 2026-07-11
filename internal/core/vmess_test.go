package core

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"

	"github.com/rtxnik/lazyray/internal/config"
)

func TestParseVMess_Standard(t *testing.T) {
	vj := vmessJSON{
		V:    "2",
		PS:   "Test VMess",
		Add:  "1.2.3.4",
		Port: 443,
		ID:   "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
		Aid:  0,
		Scy:  "auto",
		Net:  "ws",
		Host: "example.com",
		Path: "/ws",
		TLS:  "tls",
		SNI:  "example.com",
		FP:   "chrome",
	}
	data, _ := json.Marshal(vj)
	rawURL := "vmess://" + base64.StdEncoding.EncodeToString(data)

	p, err := ParseVMess(rawURL)
	if err != nil {
		t.Fatalf("ParseVMess() error = %v", err)
	}

	if p.Name != "Test VMess" {
		t.Errorf("Name = %q, want %q", p.Name, "Test VMess")
	}
	if p.Server.Address != "1.2.3.4" {
		t.Errorf("Address = %q, want %q", p.Server.Address, "1.2.3.4")
	}
	if p.Server.Port != 443 {
		t.Errorf("Port = %d, want 443", p.Server.Port)
	}
	if p.Server.UUID != "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee" {
		t.Errorf("UUID = %q", p.Server.UUID)
	}
	if p.Server.GetProtocol() != "vmess" {
		t.Errorf("Protocol = %q, want vmess", p.Server.GetProtocol())
	}
	if p.Server.Transport.Network != "ws" {
		t.Errorf("Network = %q, want ws", p.Server.Transport.Network)
	}
	if p.Server.Transport.Path != "/ws" {
		t.Errorf("Path = %q, want /ws", p.Server.Transport.Path)
	}
	if p.Server.Security.Type != "tls" {
		t.Errorf("Security = %q, want tls", p.Server.Security.Type)
	}
	if p.Server.AlterID != 0 {
		t.Errorf("AlterID = %d, want 0", p.Server.AlterID)
	}
}

func TestParseVMess_InvalidBase64(t *testing.T) {
	_, err := ParseVMess("vmess://not-valid-base64!!!")
	if err == nil {
		t.Fatal("expected error for invalid base64")
	}
}

func TestParseVMess_MissingFields(t *testing.T) {
	vj := vmessJSON{V: "2"} // Missing required fields
	data, _ := json.Marshal(vj)
	rawURL := "vmess://" + base64.StdEncoding.EncodeToString(data)

	_, err := ParseVMess(rawURL)
	if err == nil {
		t.Fatal("expected error for missing fields")
	}
}

func TestToVMessURL_Roundtrip(t *testing.T) {
	vj := vmessJSON{
		V:    "2",
		PS:   "RT Test",
		Add:  "5.6.7.8",
		Port: 8443,
		ID:   "11111111-2222-3333-4444-555555555555",
		Aid:  0,
		Scy:  "chacha20-poly1305",
		Net:  "tcp",
		TLS:  "tls",
		SNI:  "test.com",
	}
	data, _ := json.Marshal(vj)
	original := "vmess://" + base64.StdEncoding.EncodeToString(data)

	p, err := ParseVMess(original)
	if err != nil {
		t.Fatalf("ParseVMess() error = %v", err)
	}

	exported := ToVMessURL(p)
	p2, err := ParseVMess(exported)
	if err != nil {
		t.Fatalf("ParseVMess(exported) error = %v", err)
	}

	if p2.Server.Address != p.Server.Address {
		t.Errorf("Address mismatch: %q vs %q", p2.Server.Address, p.Server.Address)
	}
	if p2.Server.Port != p.Server.Port {
		t.Errorf("Port mismatch: %d vs %d", p2.Server.Port, p.Server.Port)
	}
	if p2.Server.UUID != p.Server.UUID {
		t.Errorf("UUID mismatch")
	}
}

func TestParseTrojan_Standard(t *testing.T) {
	rawURL := "trojan://mypassword@1.2.3.4:443?security=tls&sni=example.com&type=ws&path=/ws#TrojanTest"

	p, err := ParseTrojan(rawURL)
	if err != nil {
		t.Fatalf("ParseTrojan() error = %v", err)
	}

	if p.Name != "TrojanTest" {
		t.Errorf("Name = %q, want %q", p.Name, "TrojanTest")
	}
	if p.Server.Address != "1.2.3.4" {
		t.Errorf("Address = %q", p.Server.Address)
	}
	if p.Server.Port != 443 {
		t.Errorf("Port = %d", p.Server.Port)
	}
	if p.Server.UUID != "mypassword" {
		t.Errorf("UUID (password) = %q", p.Server.UUID)
	}
	if p.Server.GetProtocol() != "trojan" {
		t.Errorf("Protocol = %q, want trojan", p.Server.GetProtocol())
	}
	if p.Server.Transport.Network != "ws" {
		t.Errorf("Network = %q, want ws", p.Server.Transport.Network)
	}
	if p.Server.Security.Type != "tls" {
		t.Errorf("Security = %q, want tls", p.Server.Security.Type)
	}
	if p.Server.Security.SNI != "example.com" {
		t.Errorf("SNI = %q", p.Server.Security.SNI)
	}
}

func TestParseTrojan_DefaultPort(t *testing.T) {
	rawURL := "trojan://pass@host#name"
	p, err := ParseTrojan(rawURL)
	if err != nil {
		t.Fatalf("ParseTrojan() error = %v", err)
	}
	if p.Server.Port != 443 {
		t.Errorf("Port = %d, want 443 (default)", p.Server.Port)
	}
}

func TestToTrojanURL_Roundtrip(t *testing.T) {
	rawURL := "trojan://testpass@9.8.7.6:8443?sni=test.com#Trojan+RT"

	p, err := ParseTrojan(rawURL)
	if err != nil {
		t.Fatalf("ParseTrojan() error = %v", err)
	}

	exported := ToTrojanURL(p)
	p2, err := ParseTrojan(exported)
	if err != nil {
		t.Fatalf("ParseTrojan(exported) error = %v", err)
	}

	if p2.Server.Address != p.Server.Address {
		t.Errorf("Address mismatch")
	}
	if p2.Server.Port != p.Server.Port {
		t.Errorf("Port mismatch")
	}
	if p2.Server.UUID != p.Server.UUID {
		t.Errorf("Password mismatch")
	}
}

func TestToProxyURL_AllProtocols(t *testing.T) {
	tests := []struct {
		name     string
		protocol string
		prefix   string
	}{
		{"vless", "", "vless://"},
		{"vmess", "vmess", "vmess://"},
		{"trojan", "trojan", "trojan://"},
		{"shadowsocks", "shadowsocks", "ss://"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := &config.Profile{
				Name: "Test",
				Server: config.ServerConfig{
					Address:    "1.2.3.4",
					Port:       443,
					UUID:       "test-uuid",
					Encryption: "none",
					Protocol:   tc.protocol,
					Transport: config.TransportConfig{
						Network: "tcp",
					},
					Security: config.SecurityConfig{
						Type: "none",
					},
				},
			}
			if tc.protocol == "shadowsocks" {
				p.Server.Encryption = "aes-256-gcm"
			}
			url := ToProxyURL(p)
			if !strings.HasPrefix(url, tc.prefix) {
				t.Errorf("ToProxyURL() = %q, want prefix %q", url, tc.prefix)
			}
		})
	}
}

func TestParseProxyURL_AutoDetect(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		protocol string
	}{
		{"vless", "vless://uuid@1.2.3.4:443?type=tcp#test", "vless"},
		{"trojan", "trojan://pass@1.2.3.4:443#test", "trojan"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p, err := ParseProxyURL(tc.url)
			if err != nil {
				t.Fatalf("ParseProxyURL() error = %v", err)
			}
			if p.Server.GetProtocol() != tc.protocol {
				t.Errorf("Protocol = %q, want %q", p.Server.GetProtocol(), tc.protocol)
			}
		})
	}
}

func TestParseProxyURL_Invalid(t *testing.T) {
	_, err := ParseProxyURL("http://example.com")
	if err == nil {
		t.Fatal("expected error for unsupported protocol")
	}
	if !strings.Contains(err.Error(), "unsupported") {
		t.Errorf("error should mention unsupported: %v", err)
	}
}

func TestTrojan_IPv6RoundTripPreservesProfile(t *testing.T) {
	p, err := ParseTrojan("trojan://pw@[2001:db8::1]:443?sni=s.example#v6")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	p2, err := ParseTrojan(ToTrojanURL(p))
	if err != nil {
		t.Fatalf("re-parse: %v", err)
	}
	if p2.Server.Address != p.Server.Address || p2.Server.Port != p.Server.Port || p2.Server.UUID != p.Server.UUID {
		t.Errorf("round trip mismatch: got %s:%d %s, want %s:%d %s",
			p2.Server.Address, p2.Server.Port, p2.Server.UUID,
			p.Server.Address, p.Server.Port, p.Server.UUID)
	}
}

func TestToTrojanURL_IPv4AndHostnameUnchanged(t *testing.T) {
	for _, host := range []string{"1.2.3.4", "example.com"} {
		p, err := ParseTrojan("trojan://pw@" + host + ":443#n")
		if err != nil {
			t.Fatalf("parse: %v", err)
		}
		if out := ToTrojanURL(p); !strings.Contains(out, "@"+host+":443") {
			t.Errorf("non-IPv6 authority must stay unbracketed, got: %s", out)
		}
	}
}
