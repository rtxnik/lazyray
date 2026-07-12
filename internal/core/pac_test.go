package core

import (
	"strings"
	"testing"

	"github.com/rtxnik/lazyray/internal/config"
)

func TestGeneratePAC_Basic(t *testing.T) {
	settings := config.DefaultSettings()
	pac := GeneratePAC(settings, nil)

	if !strings.Contains(pac, "FindProxyForURL") {
		t.Error("PAC should contain FindProxyForURL function")
	}
	if !strings.Contains(pac, "PROXY 127.0.0.1:10809") {
		t.Error("PAC should contain proxy address")
	}
	if !strings.Contains(pac, "isPlainHostName") {
		t.Error("PAC should bypass plain hostnames")
	}
	if !strings.Contains(pac, "192.168.*") {
		t.Error("PAC should bypass private networks")
	}
}

func TestGeneratePAC_WithBypassRules(t *testing.T) {
	settings := config.DefaultSettings()
	profile := &config.Profile{
		Routing: config.ProfileRouting{
			Bypass: []string{
				"domain:example.com",
				"full:exact.host.com",
				"keyword:local",
			},
		},
	}

	pac := GeneratePAC(settings, profile)

	if !strings.Contains(pac, "example.com") {
		t.Error("PAC should contain bypass domain")
	}
	if !strings.Contains(pac, "exact.host.com") {
		t.Error("PAC should contain exact host bypass")
	}
	if !strings.Contains(pac, `indexOf("local")`) {
		t.Error("PAC should contain keyword bypass")
	}
}

func TestGeneratePAC_CustomPorts(t *testing.T) {
	settings := config.DefaultSettings()
	settings.Local.HTTPPort = 12345
	settings.Local.Listen = "0.0.0.0"

	pac := GeneratePAC(settings, nil)

	if !strings.Contains(pac, "PROXY 0.0.0.0:12345") {
		t.Error("PAC should use custom HTTP port and listen address")
	}
}

func TestGeneratePAC_EscapesBypassInjection(t *testing.T) {
	profile := &config.Profile{
		Routing: config.ProfileRouting{Bypass: []string{`full:a"; INJECTED_MARKER; b="`}},
	}
	out := GeneratePAC(&config.Settings{}, profile)
	// The raw breakout (unescaped quote + bare JS) must NOT appear.
	if strings.Contains(out, `"; INJECTED_MARKER; b="`) {
		t.Fatalf("PAC injection not neutralised:\n%s", out)
	}
	// The value still appears, but only inside an escaped string literal.
	if !strings.Contains(out, `INJECTED_MARKER`) || !strings.Contains(out, `\"`) {
		t.Fatalf("expected value present but escaped:\n%s", out)
	}
}

func TestGeneratePAC_EscapesAngleBrackets(t *testing.T) {
	profile := &config.Profile{Routing: config.ProfileRouting{Bypass: []string{`keyword:<script>`}}}
	out := GeneratePAC(&config.Settings{}, profile)
	if strings.Contains(out, "<script>") {
		t.Fatalf("angle brackets not escaped:\n%s", out)
	}
}

func TestPacConditionFromEntry(t *testing.T) {
	tests := []struct {
		entry    string
		contains string
		empty    bool
	}{
		{"domain:example.com", "dnsDomainIs", false},
		{"full:exact.com", `host === "exact.com"`, false},
		{"keyword:test", `indexOf("test")`, false},
		{"geoip:private", "", true},
		{"geosite:cn", "", true},
		{"10.0.0.1", "isInNet", false},
		{"example.org", "dnsDomainIs", false},
	}

	for _, tc := range tests {
		t.Run(tc.entry, func(t *testing.T) {
			result := pacConditionFromEntry(tc.entry)
			if tc.empty && result != "" {
				t.Errorf("expected empty for %q, got %q", tc.entry, result)
			}
			if !tc.empty && !strings.Contains(result, tc.contains) {
				t.Errorf("expected %q to contain %q, got %q", tc.entry, tc.contains, result)
			}
		})
	}
}
