package core

import (
	"strings"
	"testing"

	"github.com/rtxnik/lazyray/internal/config"
)

func TestParseVLESS_AllParams(t *testing.T) {
	rawURL := "vless://123e4567-e89b-42d3-a456-426614174000@192.0.2.10:8443" +
		"?type=xhttp&security=reality&path=%2FTestPath8&mode=auto" +
		"&sni=example.org&fp=chrome&pbk=DXLqqc2ZxtxKHm_ab5GnF59s4d0SLpWz8VOwlsW3wyY" +
		"&sid=abc123&spx=%2F&flow=&encryption=none&host=example.com" +
		"#Alpha%E2%86%92Beta%20Cascade"

	p, err := ParseVLESS(rawURL)
	if err != nil {
		t.Fatalf("ParseVLESS() error = %v", err)
	}

	tests := []struct {
		name string
		got  string
		want string
	}{
		{"Name", p.Name, "Alpha→Beta Cascade"},
		{"Address", p.Server.Address, "192.0.2.10"},
		{"UUID", p.Server.UUID, "123e4567-e89b-42d3-a456-426614174000"},
		{"Encryption", p.Server.Encryption, "none"},
		{"Flow", p.Server.Flow, ""},
		{"Network", p.Server.Transport.Network, "xhttp"},
		{"Path", p.Server.Transport.Path, "/TestPath8"},
		{"Mode", p.Server.Transport.Mode, "auto"},
		{"Host", p.Server.Transport.Host, "example.com"},
		{"Security", p.Server.Security.Type, "reality"},
		{"SNI", p.Server.Security.SNI, "example.org"},
		{"Fingerprint", p.Server.Security.Fingerprint, "chrome"},
		{"PublicKey", p.Server.Security.PublicKey, "DXLqqc2ZxtxKHm_ab5GnF59s4d0SLpWz8VOwlsW3wyY"},
		{"ShortID", p.Server.Security.ShortID, "abc123"},
		{"SpiderX", p.Server.Security.SpiderX, "/"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.got != tc.want {
				t.Errorf("got %q, want %q", tc.got, tc.want)
			}
		})
	}

	if p.Server.Port != 8443 {
		t.Errorf("Port = %d, want 8443", p.Server.Port)
	}
}

func TestParseVLESS_URLDecodePath(t *testing.T) {
	rawURL := "vless://aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee@1.2.3.4:443" +
		"?type=xhttp&path=%2Fmy%2Fpath&spx=%2Findex.html#test"

	p, err := ParseVLESS(rawURL)
	if err != nil {
		t.Fatalf("ParseVLESS() error = %v", err)
	}

	if p.Server.Transport.Path != "/my/path" {
		t.Errorf("Path = %q, want %q", p.Server.Transport.Path, "/my/path")
	}

	if p.Server.Security.SpiderX != "/index.html" {
		t.Errorf("SpiderX = %q, want %q", p.Server.Security.SpiderX, "/index.html")
	}
}

func TestParseVLESS_EmptyFlowForXHTTP(t *testing.T) {
	rawURL := "vless://aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee@1.2.3.4:443" +
		"?type=xhttp&security=reality&sni=example.com&fp=chrome" +
		"&pbk=AAAA&sid=1234#no-flow"

	p, err := ParseVLESS(rawURL)
	if err != nil {
		t.Fatalf("ParseVLESS() error = %v", err)
	}

	if p.Server.Flow != "" {
		t.Errorf("Flow = %q, want empty string for XHTTP", p.Server.Flow)
	}
}

func TestParseVLESS_Defaults(t *testing.T) {
	rawURL := "vless://aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee@1.2.3.4:443#test"

	p, err := ParseVLESS(rawURL)
	if err != nil {
		t.Fatalf("ParseVLESS() error = %v", err)
	}

	if p.Server.Transport.Network != "tcp" {
		t.Errorf("Network = %q, want %q (default)", p.Server.Transport.Network, "tcp")
	}

	if p.Server.Security.Type != "none" {
		t.Errorf("Security = %q, want %q (default)", p.Server.Security.Type, "none")
	}

	if p.Server.Encryption != "none" {
		t.Errorf("Encryption = %q, want %q (default)", p.Server.Encryption, "none")
	}
}

func TestParseVLESS_DefaultName(t *testing.T) {
	rawURL := "vless://aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee@1.2.3.4:443"

	p, err := ParseVLESS(rawURL)
	if err != nil {
		t.Fatalf("ParseVLESS() error = %v", err)
	}

	if p.Name != "1.2.3.4:443" {
		t.Errorf("Name = %q, want %q (host:port default)", p.Name, "1.2.3.4:443")
	}
}

func TestParseVLESS_Invalid(t *testing.T) {
	tests := []struct {
		name   string
		rawURL string
		errMsg string
	}{
		{
			name:   "no vless prefix",
			rawURL: "https://example.com",
			errMsg: "must start with vless://",
		},
		{
			name:   "missing UUID",
			rawURL: "vless://@1.2.3.4:443#test",
			errMsg: "missing UUID",
		},
		{
			name:   "missing host",
			rawURL: "vless://aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee@:443#test",
			errMsg: "missing host",
		},
		{
			name:   "missing port",
			rawURL: "vless://aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee@1.2.3.4#test",
			errMsg: "missing port",
		},
		{
			name:   "invalid port",
			rawURL: "vless://aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee@1.2.3.4:abc#test",
			errMsg: "invalid port",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParseVLESS(tc.rawURL)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tc.errMsg) {
				t.Errorf("error %q should contain %q", err.Error(), tc.errMsg)
			}
		})
	}
}

func TestToVLESSURL(t *testing.T) {
	p := &config.Profile{
		Name: "Test Profile",
		Server: config.ServerConfig{
			Address:    "1.2.3.4",
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
				PublicKey:   "AAAA",
				ShortID:     "1234",
				SpiderX:     "/",
			},
		},
	}

	result := ToVLESSURL(p)

	if !strings.HasPrefix(result, "vless://123e4567-e89b-42d3-a456-426614174000@1.2.3.4:8443?") {
		t.Errorf("URL prefix mismatch: %s", result)
	}

	// Verify the URL contains all expected parameters
	expectedParams := []string{
		"type=xhttp",
		"security=reality",
		"sni=example.org",
		"fp=chrome",
		"pbk=AAAA",
		"sid=1234",
		"mode=auto",
	}
	for _, param := range expectedParams {
		if !strings.Contains(result, param) {
			t.Errorf("URL %q missing param %q", result, param)
		}
	}

	// Should not include encryption=none
	if strings.Contains(result, "encryption=") {
		t.Errorf("URL should not include encryption=none: %s", result)
	}

	// Fragment should contain profile name
	if !strings.HasSuffix(result, "#Test%20Profile") {
		t.Errorf("URL should end with escaped profile name, got: %s", result)
	}
}

func TestRoundtrip_ParseAndGenerate(t *testing.T) {
	original := "vless://123e4567-e89b-42d3-a456-426614174000@192.0.2.10:8443" +
		"?type=xhttp&security=reality&path=%2FTestPath8&mode=auto" +
		"&sni=example.org&fp=chrome&pbk=DXLqqc2ZxtxKHm_ab5GnF59s4d0SLpWz8VOwlsW3wyY" +
		"&sid=abc123&spx=%2F#TestProfile"

	// Parse
	p1, err := ParseVLESS(original)
	if err != nil {
		t.Fatalf("first ParseVLESS() error = %v", err)
	}

	// Convert back to URL
	generated := ToVLESSURL(p1)

	// Parse the generated URL
	p2, err := ParseVLESS(generated)
	if err != nil {
		t.Fatalf("second ParseVLESS() error = %v", err)
	}

	// Compare fields
	checks := []struct {
		name string
		a, b string
	}{
		{"Name", p1.Name, p2.Name},
		{"Address", p1.Server.Address, p2.Server.Address},
		{"UUID", p1.Server.UUID, p2.Server.UUID},
		{"Encryption", p1.Server.Encryption, p2.Server.Encryption},
		{"Flow", p1.Server.Flow, p2.Server.Flow},
		{"Network", p1.Server.Transport.Network, p2.Server.Transport.Network},
		{"Path", p1.Server.Transport.Path, p2.Server.Transport.Path},
		{"Mode", p1.Server.Transport.Mode, p2.Server.Transport.Mode},
		{"Security", p1.Server.Security.Type, p2.Server.Security.Type},
		{"SNI", p1.Server.Security.SNI, p2.Server.Security.SNI},
		{"Fingerprint", p1.Server.Security.Fingerprint, p2.Server.Security.Fingerprint},
		{"PublicKey", p1.Server.Security.PublicKey, p2.Server.Security.PublicKey},
		{"ShortID", p1.Server.Security.ShortID, p2.Server.Security.ShortID},
		{"SpiderX", p1.Server.Security.SpiderX, p2.Server.Security.SpiderX},
	}

	for _, c := range checks {
		if c.a != c.b {
			t.Errorf("roundtrip %s mismatch: %q vs %q", c.name, c.a, c.b)
		}
	}

	if p1.Server.Port != p2.Server.Port {
		t.Errorf("roundtrip Port mismatch: %d vs %d", p1.Server.Port, p2.Server.Port)
	}
}

func TestVLESS_IPv6RoundTripPreservesProfile(t *testing.T) {
	p, err := ParseVLESS("vless://11111111-1111-1111-1111-111111111111@[2001:db8::1]:443?security=reality&type=grpc&sni=s.example#v6")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	p2, err := ParseVLESS(ToVLESSURL(p))
	if err != nil {
		t.Fatalf("re-parse: %v", err)
	}
	if p2.Server.Address != p.Server.Address || p2.Server.Port != p.Server.Port || p2.Server.UUID != p.Server.UUID {
		t.Errorf("round trip mismatch: got %s:%d %s, want %s:%d %s",
			p2.Server.Address, p2.Server.Port, p2.Server.UUID,
			p.Server.Address, p.Server.Port, p.Server.UUID)
	}
	if out := ToVLESSURL(p); strings.Contains(out, "[[") {
		t.Errorf("address must not be double-bracketed: %s", out)
	}
}

func TestToVLESSURL_IPv4AndHostnameUnchanged(t *testing.T) {
	for _, host := range []string{"1.2.3.4", "example.com"} {
		p, err := ParseVLESS("vless://uuid-1@" + host + ":443?security=none&type=tcp#n")
		if err != nil {
			t.Fatalf("parse: %v", err)
		}
		if out := ToVLESSURL(p); !strings.Contains(out, "@"+host+":443") {
			t.Errorf("non-IPv6 authority must stay unbracketed, got: %s", out)
		}
	}
}
