package panels

import (
	"strings"
	"testing"
	"time"

	"github.com/rtxnik/lazyray/internal/config"
	"github.com/rtxnik/lazyray/internal/core"
)

func TestNewStatusPanelInitsRings(t *testing.T) {
	s := NewStatusPanel()
	if s.SpeedRing == nil || s.LatencyRing == nil {
		t.Fatal("rings must be initialized by NewStatusPanel")
	}
	s.SpeedRing.Push(1)
	if s.SpeedRing.Len() != 1 {
		t.Fatalf("SpeedRing.Len = %d, want 1", s.SpeedRing.Len())
	}
}

func TestToggleMetricCycles(t *testing.T) {
	s := NewStatusPanel()
	if s.Metric != MetricSpeed {
		t.Fatalf("default metric = %v, want MetricSpeed", s.Metric)
	}
	s.ToggleMetric()
	if s.Metric != MetricLatency {
		t.Fatalf("after toggle = %v, want MetricLatency", s.Metric)
	}
	s.ToggleMetric()
	if s.Metric != MetricSpeed {
		t.Fatalf("after 2nd toggle = %v, want MetricSpeed", s.Metric)
	}
}

func runningPanel() StatusPanel {
	s := NewStatusPanel()
	s.Width = 50
	s.Height = 24 // generous: no collapse
	s.Settings = config.DefaultSettings()
	s.Status = &core.XrayStatus{Running: true, PID: 1234, Uptime: 90 * time.Minute, SocksOK: true, HTTPOK: true}
	s.ActiveProfile = &config.Profile{
		Name:   "tokyo",
		Server: config.ServerConfig{Address: "host.example", Port: 443, Protocol: "vless"},
	}
	s.ExitIP = "203.0.113.45"
	s.Latency = "45ms"
	s.DnSpeed = "2.4 MB/s"
	s.UpSpeed = "0.3 MB/s"
	s.Traffic = &core.TrafficStats{Uplink: 125, Downlink: 540}
	return s
}

func TestDashboardBadgeAndKPIs(t *testing.T) {
	s := runningPanel()
	view := s.View()
	for _, want := range []string{"Connected", "VLESS", "Latency", "45ms", "Exit IP", "203.0.113.45", "SOCKS", "HTTP", "Traffic"} {
		if !strings.Contains(view, want) {
			t.Errorf("dashboard view missing %q\n---\n%s", want, view)
		}
	}
}

func TestDashboardSparklineDefaultsToSpeed(t *testing.T) {
	s := runningPanel()
	view := s.View()
	if !strings.Contains(view, "Download") {
		t.Errorf("default sparkline header should read Download:\n%s", view)
	}
}

func TestDashboardSparklineLatencyAfterToggle(t *testing.T) {
	s := runningPanel()
	s.ToggleMetric()
	view := s.View()
	if !strings.Contains(view, "Latency  45ms") && !strings.Contains(view, "Latency 45ms") {
		// the sparkline header for latency mode shows the latency value
		if !strings.Contains(view, "Latency") {
			t.Errorf("latency-mode sparkline header missing:\n%s", view)
		}
	}
}

func TestDashboardTopologyStrip(t *testing.T) {
	s := runningPanel()
	view := s.View()
	if !strings.Contains(view, "host.example:443") {
		t.Errorf("topology strip should show endpoint:\n%s", view)
	}
	if !strings.Contains(view, "→") {
		t.Errorf("topology strip should use arrows:\n%s", view)
	}
}

func TestTopologyNarrowCollapseNoPanic(t *testing.T) {
	// Narrow width takes the collapse branch in renderTopology, which must
	// bounds-check segs before indexing segs[len(segs)-2]. A minimal profile
	// must not panic and must collapse to the endpoint with an ellipsis.
	s := NewStatusPanel()
	s.Width = 20 // < 28: forces the collapse branch
	s.Settings = config.DefaultSettings()
	s.Status = &core.XrayStatus{Running: true}
	s.ActiveProfile = &config.Profile{
		Name:   "x",
		Server: config.ServerConfig{Address: "h", Port: 1, Protocol: "vless"},
	}
	out := s.renderTopology()
	if !strings.Contains(out, "…") {
		t.Errorf("narrow topology should collapse with an ellipsis: %q", out)
	}
	if !strings.Contains(out, "h:1") {
		t.Errorf("narrow topology should keep the endpoint: %q", out)
	}
}

func TestCollapseDropsSparklineBeforeLatency(t *testing.T) {
	s := runningPanel()
	s.Height = 7 // small: budget = 5 content lines
	view := s.View()
	if strings.Contains(view, "Download") {
		t.Errorf("sparkline should be shed first under tight height:\n%s", view)
	}
	if !strings.Contains(view, "Latency") {
		t.Errorf("Latency (floor) must survive collapse:\n%s", view)
	}
	if !strings.Contains(view, "Connected") {
		t.Errorf("badge (floor) must survive collapse:\n%s", view)
	}
}

func TestCollapseHeightBudgetRespected(t *testing.T) {
	s := runningPanel()
	s.Height = 9
	lines := strings.Count(s.View(), "\n") + 1
	if lines > s.Height-2 {
		t.Errorf("rendered %d lines, budget is %d", lines, s.Height-2)
	}
}

func TestNarrowWidthStacksKPIs(t *testing.T) {
	s := runningPanel()
	s.Width = 30 // below the 2-column threshold
	view := s.View()
	// In narrow mode the Down speed moves to its own line, so "Latency" and
	// "Down" must not share a single line.
	for _, ln := range strings.Split(view, "\n") {
		if strings.Contains(ln, "Latency") && strings.Contains(ln, "Down") {
			t.Errorf("narrow mode must stack KPIs; found combined row %q", ln)
		}
	}
}
