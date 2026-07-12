package core

import (
	"encoding/base64"
	"strings"
	"testing"

	"github.com/rtxnik/lazyray/internal/config"
)

func TestParseProxyURL_StripsControlFromName(t *testing.T) {
	// #%1B%5B31mHACKED%1B%5B0m  ->  ESC [ 31 m HACKED ESC [ 0 m
	p, err := ParseProxyURL("vless://11111111-1111-1111-1111-111111111111@example.com:443?type=tcp#%1B%5B31mHACKED%1B%5B0m")
	if err != nil {
		t.Fatal(err)
	}
	if strings.ContainsRune(p.Name, 0x1b) {
		t.Fatalf("ESC survived in Name: %q", p.Name)
	}
}

func TestParseProxyURL_StripsControlFromNetwork_VMess(t *testing.T) {
	vj := `{"v":"2","ps":"x","add":"h.example","port":"443","id":"11111111-1111-1111-1111-111111111111","net":"\u001b[31mws","tls":""}`
	url := "vmess://" + base64.StdEncoding.EncodeToString([]byte(vj))
	p, err := ParseProxyURL(url)
	if err != nil {
		t.Fatal(err)
	}
	if strings.ContainsRune(p.Server.Transport.Network, 0x1b) {
		t.Fatalf("ESC survived in Network: %q", p.Server.Transport.Network)
	}
}

func TestImportEncrypted_StripsControlFromName(t *testing.T) {
	blob, err := ExportEncrypted([]config.Profile{{
		Name:   "e\x1b[31mvil",
		Server: config.ServerConfig{Address: "h.example", Port: 443, UUID: "11111111-1111-1111-1111-111111111111", Transport: config.TransportConfig{Network: "tcp"}, Security: config.SecurityConfig{Type: "none"}},
	}}, "pw")
	if err != nil {
		t.Fatal(err)
	}
	profiles, err := ImportEncrypted(blob, "pw")
	if err != nil {
		t.Fatal(err)
	}
	if strings.ContainsRune(profiles[0].Name, 0x1b) {
		t.Fatalf("ESC survived encrypted import: %q", profiles[0].Name)
	}
}
