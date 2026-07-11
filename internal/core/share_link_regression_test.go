package core

import (
	"encoding/base64"
	"strings"
	"testing"

	"github.com/rtxnik/lazyray/internal/config"
)

// Regression suite for share-link parser/exporter round-trip defects.
// Committed red-first; each fix turns its subset green and keeps it green.

func TestParseShadowsocksTrailingSlash(t *testing.T) {
	if _, err := ParseShadowsocks("ss://aes-256-gcm:password@example.com:8388#n"); err != nil {
		t.Fatalf("control (no slash) should parse: %v", err)
	}
	p, err := ParseShadowsocks("ss://aes-256-gcm:password@example.com:8388/#n")
	if err != nil {
		t.Fatalf("SIP002 URL with trailing slash should parse, got: %v", err)
	}
	if p.Server.Port != 8388 {
		t.Fatalf("Port = %d, want 8388", p.Server.Port)
	}
}

func TestParseShadowsocksPluginRejectedWithReason(t *testing.T) {
	_, err := ParseShadowsocks("ss://aes-256-gcm:password@example.com:8388/?plugin=v2ray-plugin#n")
	if err == nil {
		t.Fatal("plugin= must be rejected explicitly, got nil error")
	}
	if !strings.Contains(err.Error(), "plugin") || !strings.Contains(err.Error(), "not supported") {
		// The pre-fix port-parse error echoes the raw query (and thus the word
		// "plugin"); only an explicit not-supported rejection may pass.
		t.Fatalf("error must reject the plugin as not supported, got: %v", err)
	}
}

func TestParseShadowsocksPlaintextPercentDecoded(t *testing.T) {
	p, err := ParseShadowsocks("ss://aes-256-gcm:p%40ss@h.example.com:443#n")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if p.Server.UUID != "p@ss" {
		t.Fatalf("password = %q, want %q (percent-decoded)", p.Server.UUID, "p@ss")
	}
}

func TestToVLESSURLBracketsIPv6(t *testing.T) {
	in := "vless://11111111-1111-1111-1111-111111111111@[2001:db8::1]:443?type=tcp&security=none#n"
	p, err := ParseVLESS(in)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if p.Server.Address != "2001:db8::1" {
		t.Fatalf("Address = %q, want unbracketed 2001:db8::1", p.Server.Address)
	}
	out := ToVLESSURL(p)
	if !strings.Contains(out, "@[2001:db8::1]:443") {
		t.Fatalf("export must bracket IPv6, got: %s", out)
	}
	if _, err := ParseVLESS(out); err != nil {
		t.Fatalf("re-import of exported IPv6 URL failed: %v\nexported=%s", err, out)
	}
}

func TestToTrojanURLBracketsIPv6(t *testing.T) {
	p, err := ParseTrojan("trojan://pw@[2001:db8::1]:443?sni=example.com#n")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	out := ToTrojanURL(p)
	if !strings.Contains(out, "@[2001:db8::1]:443") {
		t.Fatalf("export must bracket IPv6, got: %s", out)
	}
	if _, err := ParseTrojan(out); err != nil {
		t.Fatalf("re-import of exported IPv6 URL failed: %v\nexported=%s", err, out)
	}
}

func TestToShadowsocksURLBracketsIPv6(t *testing.T) {
	userinfo := base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString([]byte("aes-256-gcm:pw"))
	p, err := ParseShadowsocks("ss://" + userinfo + "@[2001:db8::1]:443#n")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	out := ToShadowsocksURL(p)
	if !strings.Contains(out, "@[2001:db8::1]:443") {
		t.Fatalf("export must bracket IPv6, got: %s", out)
	}
}

func TestParseSubscriptionBodyRawURLBase64(t *testing.T) {
	line := "vless://11111111-1111-1111-1111-111111111111@h.example.com:443?type=tcp&security=none#nn"
	body := base64.RawURLEncoding.EncodeToString([]byte(line))
	// Fixture guards: the defect triggers only when the encoding contains
	// URL-safe characters AND padded-URL decoding cannot rescue it.
	if !strings.ContainsAny(body, "-_") {
		t.Fatal("fixture error: encoded body must contain URL-safe base64 chars")
	}
	if len(body)%4 == 0 {
		t.Fatal("fixture error: body length must not be a multiple of 4")
	}
	profiles, err := ParseSubscriptionBody(body)
	if err != nil {
		t.Fatalf("URL-safe unpadded base64 subscription should decode, got: %v", err)
	}
	if len(profiles) != 1 {
		t.Fatalf("profiles = %d, want 1", len(profiles))
	}
}

func TestToHysteria2URLKeepsBasePortWithHopRange(t *testing.T) {
	p := &config.Profile{
		Name: "n",
		Server: config.ServerConfig{
			Address:     "example.com",
			Port:        443,
			UUID:        "auth",
			Protocol:    "hysteria2",
			PortHopping: "5000-6000",
			Transport:   config.TransportConfig{Network: "hysteria"},
			Security:    config.SecurityConfig{Type: "tls", SNI: "example.com"},
		},
	}
	out := ToHysteria2URL(p)
	if !strings.Contains(out, "@example.com:443,5000-6000") {
		t.Fatalf("export must keep the base port ahead of hop ranges, got: %s", out)
	}
	p2, err := ParseHysteria2(out)
	if err != nil {
		t.Fatalf("re-parse: %v", err)
	}
	if p2.Server.Port != 443 {
		t.Fatalf("base Port = %d, want 443", p2.Server.Port)
	}
	if p2.Server.PortHopping != "443,5000-6000" {
		t.Fatalf("PortHopping = %q, want %q", p2.Server.PortHopping, "443,5000-6000")
	}
}
