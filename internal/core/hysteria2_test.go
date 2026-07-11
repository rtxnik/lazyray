package core

import (
	"strings"
	"testing"

	"github.com/rtxnik/lazyray/internal/config"
)

func TestParseHysteria2_Full(t *testing.T) {
	raw := "hysteria2://secretauth@example.com:29347?alpn=h3&fp=chrome&obfs=salamander&obfs-password=obfspw&security=tls&sni=real.example.com#hy2-exit"
	p, err := ParseHysteria2(raw)
	if err != nil {
		t.Fatalf("ParseHysteria2() error = %v", err)
	}
	if p.Server.GetProtocol() != "hysteria2" {
		t.Errorf("Protocol = %q, want hysteria2", p.Server.GetProtocol())
	}
	if p.Server.Address != "example.com" || p.Server.Port != 29347 {
		t.Errorf("addr/port = %s:%d", p.Server.Address, p.Server.Port)
	}
	if p.Server.UUID != "secretauth" {
		t.Errorf("auth (UUID) = %q", p.Server.UUID)
	}
	if p.Server.Obfs != "salamander" || p.Server.ObfsPassword != "obfspw" {
		t.Errorf("obfs = %q / %q", p.Server.Obfs, p.Server.ObfsPassword)
	}
	if p.Server.Security.SNI != "real.example.com" {
		t.Errorf("SNI = %q", p.Server.Security.SNI)
	}
	if p.Server.Security.ALPN != "h3" || p.Server.Security.Fingerprint != "chrome" {
		t.Errorf("alpn/fp = %q / %q", p.Server.Security.ALPN, p.Server.Security.Fingerprint)
	}
	if p.Server.Transport.Network != "hysteria" {
		t.Errorf("Network = %q, want hysteria", p.Server.Transport.Network)
	}
	if p.Name != "hy2-exit" {
		t.Errorf("Name = %q", p.Name)
	}
}

func TestParseHysteria2_AliasDefaultPort(t *testing.T) {
	p, err := ParseHysteria2("hy2://pw@host#n")
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if p.Server.Port != 443 {
		t.Errorf("Port = %d, want 443", p.Server.Port)
	}
}

func TestParseHysteria2_Insecure(t *testing.T) {
	p, err := ParseHysteria2("hysteria2://pw@host:443?insecure=1#n")
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if !p.Server.Security.AllowInsecure {
		t.Error("AllowInsecure = false, want true")
	}
}

func TestParseHysteria2_MissingAuth(t *testing.T) {
	if _, err := ParseHysteria2("hysteria2://@host:443#n"); err == nil {
		t.Fatal("expected error for missing auth")
	}
}

func TestParseHysteria2_PortHoppingPreserved(t *testing.T) {
	p, err := ParseHysteria2("hysteria2://pw@host:443,5000-6000#n")
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if p.Server.Port != 443 {
		t.Errorf("base Port = %d, want 443", p.Server.Port)
	}
	if p.Server.PortHopping != "443,5000-6000" {
		t.Errorf("PortHopping = %q, want 443,5000-6000", p.Server.PortHopping)
	}
}

func TestParseHysteria2_PortHoppingPureRange(t *testing.T) {
	p, err := ParseHysteria2("hysteria2://pw@host:5000-6000#n")
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if p.Server.Port != 5000 {
		t.Errorf("base Port = %d, want 5000 (low end of range)", p.Server.Port)
	}
	if p.Server.PortHopping != "5000-6000" {
		t.Errorf("PortHopping = %q, want 5000-6000", p.Server.PortHopping)
	}
}

func TestParseProxyURL_Hysteria2(t *testing.T) {
	for _, u := range []string{"hysteria2://pw@h:443#n", "hy2://pw@h:443#n"} {
		p, err := ParseProxyURL(u)
		if err != nil {
			t.Fatalf("ParseProxyURL(%q) error = %v", u, err)
		}
		if p.Server.GetProtocol() != "hysteria2" {
			t.Errorf("Protocol = %q, want hysteria2", p.Server.GetProtocol())
		}
	}
}

func TestToHysteria2URL_Roundtrip(t *testing.T) {
	raw := "hysteria2://mypw@9.8.7.6:8443?alpn=h3&fp=chrome&insecure=1&obfs=salamander&obfs-password=op&sni=test.com#hy2 RT"
	p, err := ParseHysteria2(raw)
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	p2, err := ParseHysteria2(ToHysteria2URL(p))
	if err != nil {
		t.Fatalf("re-parse error = %v", err)
	}
	if p2.Server.UUID != p.Server.UUID || p2.Server.Address != p.Server.Address ||
		p2.Server.Port != p.Server.Port || p2.Server.Obfs != p.Server.Obfs ||
		p2.Server.ObfsPassword != p.Server.ObfsPassword || p2.Name != p.Name {
		t.Errorf("roundtrip mismatch:\n got  %+v\n want %+v", p2.Server, p.Server)
	}
	if p2.Server.Security.SNI != p.Server.Security.SNI ||
		p2.Server.Security.ALPN != p.Server.Security.ALPN ||
		p2.Server.Security.Fingerprint != p.Server.Security.Fingerprint ||
		p2.Server.Security.AllowInsecure != p.Server.Security.AllowInsecure {
		t.Errorf("security roundtrip mismatch:\n got  %+v\n want %+v", p2.Server.Security, p.Server.Security)
	}
}

func TestParseHysteria2_PinSHA256Normalized(t *testing.T) {
	raw := "hysteria2://pw@host:443?pinSHA256=BA:88:45:17#n"
	p, err := ParseHysteria2(raw)
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if p.Server.Security.PinSHA256 != "ba884517" {
		t.Errorf("PinSHA256 = %q, want ba884517 (lowercased, separators stripped)", p.Server.Security.PinSHA256)
	}
}

func TestToHysteria2URL_PinAndHopRoundtrip(t *testing.T) {
	const pin = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	raw := "hysteria2://pw@host:443,5000-6000?obfs=salamander&obfs-password=op&pinSHA256=" + pin + "&sni=s#rt"
	p, err := ParseHysteria2(raw)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	p2, err := ParseHysteria2(ToHysteria2URL(p))
	if err != nil {
		t.Fatalf("re-parse: %v", err)
	}
	if p2.Server.PortHopping != "443,5000-6000" {
		t.Errorf("PortHopping roundtrip = %q", p2.Server.PortHopping)
	}
	if p2.Server.Security.PinSHA256 != pin {
		t.Errorf("PinSHA256 roundtrip = %q", p2.Server.Security.PinSHA256)
	}
	if p2.Server.Security.AllowInsecure {
		t.Error("insecure must not be set when pin is present")
	}
}

func TestHopBasePort(t *testing.T) {
	cases := []struct {
		in      string
		want    int
		wantErr bool
	}{
		{"443", 443, false},
		{"443,5000-6000", 443, false},
		{"5000-6000", 5000, false},
		{"x-y", 0, true},
		{"", 0, true},
	}
	for _, c := range cases {
		got, err := hopBasePort(c.in)
		if (err != nil) != c.wantErr || got != c.want {
			t.Errorf("hopBasePort(%q) = (%d, %v), want (%d, wantErr=%v)", c.in, got, err, c.want, c.wantErr)
		}
	}
}

func TestToHysteria2URL_HopPortVariants(t *testing.T) {
	mk := func(port int, hop string) *config.Profile {
		return &config.Profile{
			Name: "n",
			Server: config.ServerConfig{
				Address: "h", Port: port, UUID: "pw", Protocol: "hysteria2",
				PortHopping: hop,
				Transport:   config.TransportConfig{Network: "hysteria"},
				Security:    config.SecurityConfig{Type: "tls", SNI: "h"},
			},
		}
	}
	cases := []struct {
		name     string
		port     int
		hop      string
		wantAuth string
	}{
		{"pure range with matching base", 5000, "5000-6000", "@h:5000-6000"},
		{"spec already leads with base", 443, "443,5000-6000", "@h:443,5000-6000"},
		{"base differs from range", 443, "5000-6000", "@h:443,5000-6000"},
		{"no hopping", 443, "", "@h:443"},
		{"malformed hop falls back to prepend", 443, "x-y", "@h:443,x-y"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out := ToHysteria2URL(mk(tc.port, tc.hop))
			if !strings.Contains(out, tc.wantAuth) {
				t.Fatalf("export = %s, want authority %q", out, tc.wantAuth)
			}
			p2, err := ParseHysteria2(out)
			if err != nil {
				t.Fatalf("re-parse: %v", err)
			}
			if p2.Server.Port != tc.port {
				t.Errorf("re-parsed base Port = %d, want %d", p2.Server.Port, tc.port)
			}
		})
	}
}
