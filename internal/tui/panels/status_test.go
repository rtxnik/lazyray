package panels

import (
	"strings"
	"testing"
	"time"

	"github.com/rtxnik/lazyray/internal/config"
	"github.com/rtxnik/lazyray/internal/core"
)

func TestStatusPanel_ViewStopped(t *testing.T) {
	p := NewStatusPanel()
	p.Width = 40
	p.Height = 10
	p.Settings = config.DefaultSettings()
	p.Status = &core.XrayStatus{Running: false}

	view := p.View()
	if !strings.Contains(view, "Stopped") {
		t.Error("stopped status should contain 'Stopped'")
	}
}

func TestStatusPanel_ViewRunning(t *testing.T) {
	p := NewStatusPanel()
	p.Width = 40
	p.Height = 24
	p.Settings = config.DefaultSettings()
	p.Status = &core.XrayStatus{
		Running: true,
		PID:     1234,
		Uptime:  5 * time.Minute,
		SocksOK: true,
		HTTPOK:  true,
	}

	view := p.View()
	if !strings.Contains(view, "Connected") {
		t.Error("running status should contain 'Connected'")
	}
}

func TestStatusPanel_ViewExitIP(t *testing.T) {
	p := NewStatusPanel()
	p.Width = 40
	p.Height = 24
	p.Settings = config.DefaultSettings()
	p.Status = &core.XrayStatus{Running: true, PID: 1}
	p.ExitIP = "1.2.3.4"

	view := p.View()
	if !strings.Contains(view, "1.2.3.4") {
		t.Error("view should contain exit IP")
	}
}

func TestStatusPanel_ViewTrafficZero(t *testing.T) {
	p := NewStatusPanel()
	p.Width = 40
	p.Height = 24
	p.Settings = config.DefaultSettings()
	p.Status = &core.XrayStatus{Running: true, PID: 1}
	p.Traffic = &core.TrafficStats{Uplink: 0, Downlink: 0}

	view := p.View()
	if !strings.Contains(view, "Traffic") {
		t.Error("view should contain Traffic line even with zero values")
	}
}

func TestStatusPanel_ViewTrafficWithData(t *testing.T) {
	p := NewStatusPanel()
	p.Width = 40
	p.Height = 24
	p.Settings = config.DefaultSettings()
	p.Status = &core.XrayStatus{Running: true, PID: 1}
	p.Traffic = &core.TrafficStats{Uplink: 1048576, Downlink: 2097152}
	p.UpSpeed = "100 KB/s"
	p.DnSpeed = "200 KB/s"

	view := p.View()
	if !strings.Contains(view, "Traffic") {
		t.Error("view should contain Traffic line")
	}
	if !strings.Contains(view, "100 KB/s") {
		t.Error("view should contain upload speed")
	}
	if !strings.Contains(view, "200 KB/s") {
		t.Error("view should contain download speed")
	}
}

func TestStatusPanel_ViewNoTrafficWhenNil(t *testing.T) {
	p := NewStatusPanel()
	p.Width = 40
	p.Height = 10
	p.Settings = config.DefaultSettings()
	p.Status = &core.XrayStatus{Running: false}
	p.Traffic = nil

	view := p.View()
	if strings.Contains(view, "Traffic") {
		t.Error("view should NOT contain Traffic when traffic is nil")
	}
}

func TestStatusPanel_ViewLatency(t *testing.T) {
	p := NewStatusPanel()
	p.Width = 40
	p.Height = 10
	p.Settings = config.DefaultSettings()
	p.Status = &core.XrayStatus{Running: true, PID: 1}
	p.Latency = "45ms"

	view := p.View()
	if !strings.Contains(view, "45ms") {
		t.Error("view should contain latency value")
	}
}

func TestFormatLatencyIndicator(t *testing.T) {
	tests := []struct {
		input string
	}{
		{"45ms"},
		{"150ms"},
		{"500ms"},
	}

	for _, tc := range tests {
		result := formatLatencyIndicator(tc.input)
		if result == "" {
			t.Errorf("formatLatencyIndicator(%q) returned empty", tc.input)
		}
		if !strings.Contains(result, tc.input) {
			t.Errorf("formatLatencyIndicator(%q) should contain the latency value", tc.input)
		}
	}
}

func TestStatusPanel_ViewVersionWarning(t *testing.T) {
	p := NewStatusPanel()
	p.Width = 60
	p.Height = 15
	p.Settings = config.DefaultSettings()
	p.Status = &core.XrayStatus{Running: true, PID: 1}
	p.VersionWarning = "Xray 1.7.0 is outdated (min 1.8.0), press u to update"

	view := p.View()
	if !strings.Contains(view, "outdated") {
		t.Error("view should contain version warning when set")
	}
}

func TestStatusPanel_ViewNoVersionWarning(t *testing.T) {
	p := NewStatusPanel()
	p.Width = 60
	p.Height = 15
	p.Settings = config.DefaultSettings()
	p.Status = &core.XrayStatus{Running: true, PID: 1}
	p.VersionWarning = ""

	view := p.View()
	if strings.Contains(view, "outdated") {
		t.Error("view should NOT contain version warning when empty")
	}
}

func TestParseLatencyMS(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"45ms", 45},
		{"150ms", 150},
		{"0ms", 0},
		{"1000ms", 1000},
	}

	for _, tc := range tests {
		got := parseLatencyMS(tc.input)
		if got != tc.want {
			t.Errorf("parseLatencyMS(%q) = %d, want %d", tc.input, got, tc.want)
		}
	}
}

func TestKpiRowW_IPv4DoesNotGlue(t *testing.T) {
	row := kpiRowW(80, "Exit IP", "255.255.255.255", "Up", "9.9 MB/s")
	if strings.Contains(row, "\n") {
		t.Fatalf("a full IPv4 row must stay single-line, got:\n%q", row)
	}
	if strings.Contains(row, "255.255.255.255Up") {
		t.Errorf("right label glued to the IPv4 value (no gap), got:\n%q", row)
	}
	if !strings.Contains(row, "Up") || !strings.Contains(row, "255.255.255.255") {
		t.Errorf("row lost a cell, got:\n%q", row)
	}
}

func TestKpiRowW_IPv6Stacks(t *testing.T) {
	row := kpiRowW(80, "Exit IP", "2001:0db8:85a3:0000:0000:8a2e:0370:7334", "Up", "9.9 MB/s")
	if !strings.Contains(row, "\n") {
		t.Errorf("an over-long (IPv6) value must stack the row, got:\n%q", row)
	}
	if !strings.Contains(row, "Up") {
		t.Errorf("stacked row dropped the right cell, got:\n%q", row)
	}
}

func TestKpiRowW_NarrowStacks(t *testing.T) {
	row := kpiRowW(30, "Exit IP", "1.2.3.4", "Up", "x")
	if !strings.Contains(row, "\n") {
		t.Errorf("narrow width must stack, got:\n%q", row)
	}
}
