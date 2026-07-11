package core

import (
	"encoding/base64"
	"strings"
	"testing"

	"github.com/rtxnik/lazyray/internal/config"
)

func TestParseShadowsocks_SIP002(t *testing.T) {
	// SIP002 format: ss://base64(method:password)@host:port#name
	userinfo := base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(
		[]byte("aes-256-gcm:testpassword123"),
	)
	rawURL := "ss://" + userinfo + "@1.2.3.4:8388#SS%20Server"

	p, err := ParseShadowsocks(rawURL)
	if err != nil {
		t.Fatalf("ParseShadowsocks() error = %v", err)
	}

	if p.Name != "SS Server" {
		t.Errorf("Name = %q, want %q", p.Name, "SS Server")
	}
	if p.Server.Address != "1.2.3.4" {
		t.Errorf("Address = %q, want %q", p.Server.Address, "1.2.3.4")
	}
	if p.Server.Port != 8388 {
		t.Errorf("Port = %d, want 8388", p.Server.Port)
	}
	if p.Server.UUID != "testpassword123" {
		t.Errorf("Password = %q, want %q", p.Server.UUID, "testpassword123")
	}
	if p.Server.Encryption != "aes-256-gcm" {
		t.Errorf("Method = %q, want %q", p.Server.Encryption, "aes-256-gcm")
	}
	if p.Server.GetProtocol() != "shadowsocks" {
		t.Errorf("Protocol = %q, want shadowsocks", p.Server.GetProtocol())
	}
}

func TestParseShadowsocks_ChaCha20(t *testing.T) {
	userinfo := base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(
		[]byte("chacha20-ietf-poly1305:secretkey"),
	)
	rawURL := "ss://" + userinfo + "@example.com:443#ChaCha"

	p, err := ParseShadowsocks(rawURL)
	if err != nil {
		t.Fatalf("ParseShadowsocks() error = %v", err)
	}

	if p.Server.Encryption != "chacha20-ietf-poly1305" {
		t.Errorf("Method = %q, want chacha20-ietf-poly1305", p.Server.Encryption)
	}
}

func TestParseShadowsocks_Blake3(t *testing.T) {
	userinfo := base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(
		[]byte("2022-blake3-aes-256-gcm:YWJjZGVmZ2hpamtsbW5vcHFyc3R1dnd4eXoxMjM0NTY="),
	)
	rawURL := "ss://" + userinfo + "@10.0.0.1:1234#Blake3"

	p, err := ParseShadowsocks(rawURL)
	if err != nil {
		t.Fatalf("ParseShadowsocks() error = %v", err)
	}

	if p.Server.Encryption != "2022-blake3-aes-256-gcm" {
		t.Errorf("Method = %q, want 2022-blake3-aes-256-gcm", p.Server.Encryption)
	}
	if p.Server.UUID != "YWJjZGVmZ2hpamtsbW5vcHFyc3R1dnd4eXoxMjM0NTY=" {
		t.Errorf("Password = %q", p.Server.UUID)
	}
}

func TestParseShadowsocks_PlaintextUserinfo(t *testing.T) {
	// Some tools use plaintext method:password without base64
	rawURL := "ss://aes-256-gcm:mypassword@5.6.7.8:9999#Plain"

	p, err := ParseShadowsocks(rawURL)
	if err != nil {
		t.Fatalf("ParseShadowsocks() error = %v", err)
	}

	if p.Server.Encryption != "aes-256-gcm" {
		t.Errorf("Method = %q, want aes-256-gcm", p.Server.Encryption)
	}
	if p.Server.UUID != "mypassword" {
		t.Errorf("Password = %q, want mypassword", p.Server.UUID)
	}
}

func TestParseShadowsocks_DefaultName(t *testing.T) {
	userinfo := base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(
		[]byte("aes-256-gcm:pass"),
	)
	rawURL := "ss://" + userinfo + "@1.2.3.4:8388"

	p, err := ParseShadowsocks(rawURL)
	if err != nil {
		t.Fatalf("ParseShadowsocks() error = %v", err)
	}

	if p.Name != "1.2.3.4:8388" {
		t.Errorf("Name = %q, want %q (default host:port)", p.Name, "1.2.3.4:8388")
	}
}

func TestParseShadowsocks_Invalid(t *testing.T) {
	tests := []struct {
		name   string
		rawURL string
		errMsg string
	}{
		{
			name:   "wrong prefix",
			rawURL: "http://example.com",
			errMsg: "must start with ss://",
		},
		{
			name:   "missing password",
			rawURL: "ss://" + base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString([]byte("aes-256-gcm:")) + "@1.2.3.4:8388#test",
			errMsg: "missing password",
		},
		{
			name:   "unsupported method",
			rawURL: "ss://" + base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString([]byte("rc4-md5:pass")) + "@1.2.3.4:8388#test",
			errMsg: "unsupported Shadowsocks method",
		},
		{
			name:   "missing host",
			rawURL: "ss://" + base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString([]byte("aes-256-gcm:pass")) + "@:8388#test",
			errMsg: "missing host",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParseShadowsocks(tc.rawURL)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tc.errMsg) {
				t.Errorf("error %q should contain %q", err.Error(), tc.errMsg)
			}
		})
	}
}

func TestToShadowsocksURL(t *testing.T) {
	p := &config.Profile{
		Name: "My SS",
		Server: config.ServerConfig{
			Address:    "1.2.3.4",
			Port:       8388,
			UUID:       "testpass",
			Encryption: "aes-256-gcm",
			Protocol:   "shadowsocks",
		},
	}

	result := ToShadowsocksURL(p)

	if !strings.HasPrefix(result, "ss://") {
		t.Errorf("URL should start with ss://, got: %s", result)
	}
	if !strings.Contains(result, "@1.2.3.4:8388") {
		t.Errorf("URL should contain host:port, got: %s", result)
	}
	if !strings.HasSuffix(result, "#My%20SS") {
		t.Errorf("URL should end with fragment, got: %s", result)
	}
}

func TestShadowsocks_Roundtrip(t *testing.T) {
	userinfo := base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(
		[]byte("aes-256-gcm:mysecretpassword"),
	)
	original := "ss://" + userinfo + "@192.168.1.1:8388#TestProfile"

	p1, err := ParseShadowsocks(original)
	if err != nil {
		t.Fatalf("first ParseShadowsocks() error = %v", err)
	}

	exported := ToShadowsocksURL(p1)

	p2, err := ParseShadowsocks(exported)
	if err != nil {
		t.Fatalf("second ParseShadowsocks() error = %v", err)
	}

	checks := []struct {
		name string
		a, b string
	}{
		{"Name", p1.Name, p2.Name},
		{"Address", p1.Server.Address, p2.Server.Address},
		{"Password", p1.Server.UUID, p2.Server.UUID},
		{"Method", p1.Server.Encryption, p2.Server.Encryption},
		{"Protocol", p1.Server.GetProtocol(), p2.Server.GetProtocol()},
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

func TestShadowsocks_BuildOutbound(t *testing.T) {
	profile := &config.Profile{
		Name: "SS Test",
		Server: config.ServerConfig{
			Address:    "1.2.3.4",
			Port:       8388,
			UUID:       "password123",
			Encryption: "aes-256-gcm",
			Protocol:   "shadowsocks",
			Transport: config.TransportConfig{
				Network: "tcp",
			},
			Security: config.SecurityConfig{
				Type: "none",
			},
		},
	}

	cfg, err := GenerateXrayConfig(profile, testSettings())
	if err != nil {
		t.Fatalf("GenerateXrayConfig() error = %v", err)
	}

	if cfg.Outbounds[0].Protocol != "shadowsocks" {
		t.Errorf("Outbound protocol = %q, want shadowsocks", cfg.Outbounds[0].Protocol)
	}
}

func TestParseProxyURL_Shadowsocks(t *testing.T) {
	userinfo := base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(
		[]byte("aes-256-gcm:pass"),
	)
	rawURL := "ss://" + userinfo + "@1.2.3.4:8388#test"

	p, err := ParseProxyURL(rawURL)
	if err != nil {
		t.Fatalf("ParseProxyURL() error = %v", err)
	}

	if p.Server.GetProtocol() != "shadowsocks" {
		t.Errorf("Protocol = %q, want shadowsocks", p.Server.GetProtocol())
	}
}

func TestParseShadowsocks_QueryAndPathVariants(t *testing.T) {
	userinfo := base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString([]byte("aes-256-gcm:pass"))
	cases := []struct {
		name   string
		rawURL string
	}{
		{"bare question mark", "ss://" + userinfo + "@1.2.3.4:8388?#n"},
		{"empty plugin value", "ss://" + userinfo + "@1.2.3.4:8388/?plugin=#n"},
		{"ipv6 with trailing slash", "ss://" + userinfo + "@[2001:db8::1]:8388/#n"},
		{"non-plugin query", "ss://" + userinfo + "@1.2.3.4:8388/?group=abc#n"},
		{"unparseable query kept lenient", "ss://" + userinfo + "@1.2.3.4:8388/?%zz=1#n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p, err := ParseShadowsocks(tc.rawURL)
			if err != nil {
				t.Fatalf("should parse, got: %v", err)
			}
			if p.Server.Port != 8388 {
				t.Errorf("Port = %d, want 8388", p.Server.Port)
			}
		})
	}
}

func TestParseShadowsocks_PlaintextLiteralSpecials(t *testing.T) {
	// RFC-invalid but accepted before this change: literal '?' and '/' in a
	// plaintext password must keep parsing to the same profile.
	cases := []struct {
		name, rawURL, wantPassword string
	}{
		{"literal question mark", "ss://aes-256-gcm:p?ss@example.com:8388#n", "p?ss"},
		{"literal slash", "ss://aes-256-gcm:p/ss@example.com:8388#n", "p/ss"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p, err := ParseShadowsocks(tc.rawURL)
			if err != nil {
				t.Fatalf("should parse, got: %v", err)
			}
			if p.Server.UUID != tc.wantPassword {
				t.Errorf("password = %q, want %q", p.Server.UUID, tc.wantPassword)
			}
			if p.Server.Port != 8388 {
				t.Errorf("Port = %d, want 8388", p.Server.Port)
			}
		})
	}
}

func TestParseShadowsocks_PlaintextPercentDecoding(t *testing.T) {
	// Percent-encoded method decodes; an invalid escape keeps the raw value
	// (lenient, mirrors the fragment handling).
	p, err := ParseShadowsocks("ss://aes-256%2Dgcm:pw@example.com:8388#n")
	if err != nil {
		t.Fatalf("should parse, got: %v", err)
	}
	if p.Server.Encryption != "aes-256-gcm" {
		t.Errorf("method = %q, want aes-256-gcm (percent-decoded)", p.Server.Encryption)
	}

	p, err = ParseShadowsocks("ss://aes-256-gcm:p%zzss@example.com:8388#n")
	if err != nil {
		t.Fatalf("invalid escape must stay lenient, got: %v", err)
	}
	if p.Server.UUID != "p%zzss" {
		t.Errorf("password = %q, want raw %q (unescape failed, keep raw)", p.Server.UUID, "p%zzss")
	}
}

func TestParseShadowsocks_LegacyQueryVariants(t *testing.T) {
	legacy := base64.StdEncoding.EncodeToString([]byte("aes-256-gcm:pass@9.9.9.9:8388"))

	if _, err := ParseShadowsocks("ss://" + legacy + "?plugin=obfs#n"); err == nil {
		t.Fatal("legacy link with plugin= must be rejected explicitly")
	} else if !strings.Contains(err.Error(), "plugin") {
		t.Fatalf("error should name the plugin, got: %v", err)
	}

	p, err := ParseShadowsocks("ss://" + legacy + "?foo=1#n")
	if err != nil {
		t.Fatalf("legacy link with benign query should parse, got: %v", err)
	}
	if p.Server.Port != 8388 {
		t.Errorf("Port = %d, want 8388", p.Server.Port)
	}
}

func TestShadowsocks_IPv6RoundTripPreservesProfile(t *testing.T) {
	userinfo := base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString([]byte("aes-256-gcm:pw"))
	p, err := ParseShadowsocks("ss://" + userinfo + "@[2001:db8::1]:443#v6")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	p2, err := ParseShadowsocks(ToShadowsocksURL(p))
	if err != nil {
		t.Fatalf("re-parse: %v", err)
	}
	if p2.Server.Address != "2001:db8::1" || p2.Server.Port != 443 {
		t.Errorf("round trip = %s:%d, want 2001:db8::1:443", p2.Server.Address, p2.Server.Port)
	}
	if p2.Server.UUID != "pw" || p2.Server.Encryption != "aes-256-gcm" {
		t.Errorf("credentials mismatch: %q / %q", p2.Server.Encryption, p2.Server.UUID)
	}
}
