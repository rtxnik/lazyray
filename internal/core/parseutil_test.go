package core

import (
	"encoding/base64"
	"encoding/json"
	"testing"
)

func TestAnyToInt(t *testing.T) {
	cases := []struct {
		in   any
		want int
	}{
		{float64(443), 443},
		{"8080", 8080},
		{json.Number("1080"), 1080},
		{"not-a-number", 0},
		{nil, 0},
		{true, 0},
	}
	for _, c := range cases {
		if got := anyToInt(c.in); got != c.want {
			t.Errorf("anyToInt(%#v) = %d, want %d", c.in, got, c.want)
		}
	}
}

func TestDefaultRemark(t *testing.T) {
	if got := defaultRemark("1.2.3.4", 443); got != "1.2.3.4:443" {
		t.Errorf("defaultRemark = %q, want %q", got, "1.2.3.4:443")
	}
}

func TestDecodeBase64Any(t *testing.T) {
	want := "method:pass@host"
	encs := []string{
		base64.StdEncoding.EncodeToString([]byte(want)),
		base64.RawStdEncoding.EncodeToString([]byte(want)),
		base64.URLEncoding.EncodeToString([]byte(want)),
		base64.RawURLEncoding.EncodeToString([]byte(want)),
	}
	for _, e := range encs {
		got, err := decodeBase64Any(e)
		if err != nil || string(got) != want {
			t.Errorf("decodeBase64Any(%q) = %q, %v; want %q", e, got, err, want)
		}
	}
	if _, err := decodeBase64Any("!!!not base64!!!"); err == nil {
		t.Error("expected error for non-base64")
	}
}

func TestSplitHostPort(t *testing.T) {
	cases := []struct {
		in, host, portSpec string
		wantErr            bool
	}{
		{"1.2.3.4:443", "1.2.3.4", "443", false},
		{"[2001:db8::1]:8388", "2001:db8::1", "8388", false},
		{"host:443,5000-6000", "host", "443,5000-6000", false},
		{"hostonly", "hostonly", "", false},
		{"[bad", "", "", true},
	}
	for _, c := range cases {
		h, p, err := splitHostPort(c.in)
		if (err != nil) != c.wantErr || h != c.host || p != c.portSpec {
			t.Errorf("splitHostPort(%q) = (%q,%q,%v), want (%q,%q,err=%v)",
				c.in, h, p, err, c.host, c.portSpec, c.wantErr)
		}
	}
}

func TestParseUserinfoURL(t *testing.T) {
	u, err := parseUserinfoURL("vless", "VLESS", "vless://uuid-1@1.2.3.4:443?type=tcp#name")
	if err != nil {
		t.Fatal(err)
	}
	if u.User.Username() != "uuid-1" || u.Hostname() != "1.2.3.4" || u.Port() != "443" {
		t.Errorf("parsed = user=%q host=%q port=%q", u.User.Username(), u.Hostname(), u.Port())
	}
	if u.Query().Get("type") != "tcp" || u.Fragment != "name" {
		t.Errorf("query/fragment = %q / %q", u.Query().Get("type"), u.Fragment)
	}
	if _, err := parseUserinfoURL("vless", "VLESS", "trojan://x@h:1"); err == nil {
		t.Error("expected prefix-mismatch error")
	}
}
