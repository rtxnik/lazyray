package panels

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/rtxnik/lazyray/internal/config"
	"github.com/rtxnik/lazyray/internal/core"
	"github.com/rtxnik/lazyray/internal/tui/sparkline"
	"github.com/rtxnik/lazyray/internal/tui/theme"
)

// Metric selects which series the dashboard sparkline plots.
type Metric int

const (
	MetricSpeed Metric = iota // download speed (default)
	MetricLatency
)

// StatusPanel displays the xray process status.
type StatusPanel struct {
	Status        *core.XrayStatus
	ExitIP        string
	Latency       string
	Traffic       *core.TrafficStats
	Profile       string
	ActiveProfile *config.Profile
	Settings      *config.Settings
	Width         int
	Height        int

	// Traffic speed tracking
	PrevTraffic   *core.TrafficStats
	PrevTrafficAt int64 // unix timestamp in seconds
	UpSpeed       string
	DnSpeed       string

	// Version compatibility warning
	VersionWarning string

	// Dashboard (E2e): live in-memory series, reset each session.
	SpeedRing   *sparkline.Ring // download bytes/sec samples (2s cadence)
	LatencyRing *sparkline.Ring // latency ms samples (~30s cadence)
	Metric      Metric          // which series the sparkline shows
	Focused     bool            // is the Status panel the active panel
	TodayBytes  int64           // today's down+up total (from stats manager)
	MetricKey   string          // display key for the toggle (from the registry)

	// Stopped-state teaching, fed by the app.
	StoppedReason string // "<stage>: <message>" when stopped after a startup error; else ""
	ConnectKey    string // display key to connect (clean stop)
	DoctorKey     string // display key to open diagnostics (error stop)
}

// NewStatusPanel creates a new status panel with empty sample rings.
func NewStatusPanel() StatusPanel {
	return StatusPanel{
		SpeedRing:   sparkline.NewRing(120),
		LatencyRing: sparkline.NewRing(120),
	}
}

// ToggleMetric switches the sparkline between download speed and latency.
func (s *StatusPanel) ToggleMetric() {
	if s.Metric == MetricSpeed {
		s.Metric = MetricLatency
	} else {
		s.Metric = MetricSpeed
	}
}

// dashboard line groups (used for separators and collapse shedding).
const (
	grpBadge = iota
	grpKPI
	grpSpark
	grpTopo
	grpWarn
)

// dashLine is one rendered line tagged with its group and a shed priority
// (higher = more essential; shedding drops lowest-priority lines first).
type dashLine struct {
	text  string
	group int
	prio  int
}

// View renders the status dashboard, shedding the least-essential lines when
// the panel height is too small to show everything (so it never overflows and
// starves the logs panel).
func (s *StatusPanel) View() string {
	lines := s.buildLines()
	if s.Height > 0 {
		lines = shedToFit(lines, s.Height-2) // -2 for the top/bottom border
	}
	return assemble(lines)
}

// renderedHeight counts content lines plus one blank per group boundary.
func renderedHeight(lines []dashLine) int {
	if len(lines) == 0 {
		return 0
	}
	h := len(lines)
	for i := 1; i < len(lines); i++ {
		if lines[i].group != lines[i-1].group {
			h++
		}
	}
	return h
}

// shedToFit removes the lowest-priority lines until the rendered height fits the
// budget. Lines at or above floorPrio (badge, version warning, Latency, Ports)
// are never shed; if only floor lines remain it accepts a small overflow.
func shedToFit(lines []dashLine, budget int) []dashLine {
	const floorPrio = 85
	for renderedHeight(lines) > budget {
		idx := -1
		for i, ln := range lines {
			if ln.prio >= floorPrio {
				continue
			}
			if idx == -1 || ln.prio < lines[idx].prio {
				idx = i
			}
		}
		if idx == -1 {
			break
		}
		lines = append(lines[:idx], lines[idx+1:]...)
	}
	return lines
}

// assemble joins lines in display order, inserting one blank line between
// lines that belong to different groups.
func assemble(lines []dashLine) string {
	var b strings.Builder
	for i, ln := range lines {
		if i > 0 {
			b.WriteString("\n")
			if ln.group != lines[i-1].group {
				b.WriteString("\n")
			}
		}
		b.WriteString(ln.text)
	}
	return b.String()
}

// buildLines produces the full dashboard in display order (no collapse here).
func (s *StatusPanel) buildLines() []dashLine {
	var out []dashLine
	out = append(out, s.badgeLines()...)
	if s.VersionWarning != "" {
		warn := lipgloss.NewStyle().Foreground(theme.Current().Selected)
		out = append(out, dashLine{"  " + warn.Render(s.VersionWarning), grpWarn, 95})
	}
	out = append(out, s.kpiLines()...)
	out = append(out, s.sparklineLines()...)
	if topo := s.renderTopology(); topo != "" {
		out = append(out, dashLine{topo, grpTopo, 10})
	}
	return out
}

// badgeLines returns the status badge plus, when stopped, one teaching line set:
// the failure reason + doctor handoff after an error stop, or a connect hint
// after a clean stop. Each entry is a single display line.
func (s *StatusPanel) badgeLines() []dashLine {
	if s.Status != nil && s.Status.Running {
		return []dashLine{{s.renderBadge(), grpBadge, 100}}
	}
	stopped := lipgloss.NewStyle().Foreground(theme.Current().Error).Bold(true)
	muted := lipgloss.NewStyle().Foreground(theme.Current().Muted)
	if s.StoppedReason != "" {
		dk := s.DoctorKey
		if dk == "" {
			dk = "h"
		}
		return []dashLine{
			{"  " + stopped.Render("○ Stopped — exited with error"), grpBadge, 100},
			{"  " + muted.Render("last: "+s.StoppedReason), grpBadge, 96},
			{"  " + muted.Render("press ["+dk+"] to diagnose"), grpBadge, 94},
		}
	}
	ck := s.ConnectKey
	if ck == "" {
		ck = "enter"
	}
	return []dashLine{
		{"  " + stopped.Render("○ Stopped") + muted.Render(" — press ["+ck+"] on a profile to connect"), grpBadge, 100},
	}
}

func (s *StatusPanel) renderBadge() string {
	muted := lipgloss.NewStyle().Foreground(theme.Current().Muted)
	if s.Status == nil || !s.Status.Running {
		return "  " + lipgloss.NewStyle().Foreground(theme.Current().Error).Bold(true).Render("○ Stopped")
	}
	parts := []string{lipgloss.NewStyle().Foreground(theme.Current().Success).Bold(true).Render("● Connected")}
	if s.ActiveProfile != nil {
		label := strings.ToUpper(s.ActiveProfile.Server.GetProtocol())
		if sec := s.ActiveProfile.Server.Security.Type; sec != "" && sec != "none" {
			label += "+" + strings.ToUpper(sec)
		}
		parts = append(parts, label)
	}
	parts = append(parts, "up "+core.FormatUptime(s.Status.Uptime))
	return "  " + strings.Join(parts, muted.Render(" · "))
}

func orDash(v string) string {
	if v == "" {
		return "-"
	}
	return v
}

// kpiCell renders "label  value" with a fixed-width muted label.
func kpiCell(label, value string) string {
	return lipgloss.NewStyle().Foreground(theme.Current().Muted).Width(9).Render(label) + value
}

// kpiColW fits the 9-wide label + a full 15-char IPv4 value + a 2-space gap.
// Values that would not leave the gap (e.g. a 39-char IPv6 exit IP) stack onto
// their own line instead of gluing to the right column.
const (
	kpiColW   = 26
	kpiMinGap = 2
)

// kpiRowW renders a two-column row, or stacks the two cells when the terminal is
// narrow or the left value is too wide to keep a gap before the right column.
func kpiRowW(width int, lLabel, lValue, rLabel, rValue string) string {
	left := kpiCell(lLabel, lValue)
	if rLabel == "" {
		return "  " + left
	}
	right := kpiCell(rLabel, rValue)
	if (width > 0 && width < 40) || lipgloss.Width(left)+kpiMinGap > kpiColW {
		return "  " + left + "\n  " + right
	}
	leftCol := lipgloss.NewStyle().Width(kpiColW).Render(left)
	return "  " + leftCol + right
}

func (s *StatusPanel) kpiLines() []dashLine {
	latVal := orDash("")
	if s.Latency != "" {
		latVal = formatLatencyIndicator(s.Latency)
	}
	push := func(out []dashLine, text string, prio int) []dashLine {
		for _, ln := range strings.Split(text, "\n") {
			out = append(out, dashLine{ln, grpKPI, prio})
		}
		return out
	}
	var out []dashLine
	out = push(out, kpiRowW(s.Width, "Latency", latVal, "Down", orDash(s.DnSpeed)), 90)
	out = push(out, kpiRowW(s.Width, "Exit IP", orDash(s.ExitIP), "Up", orDash(s.UpSpeed)), 30)
	out = push(out, kpiRowW(s.Width, "Ports", s.portsCell(), "", ""), 85)
	if s.Traffic != nil {
		out = push(out, kpiRowW(s.Width, "Traffic", s.trafficCell(), "", ""), 30)
	}
	return out
}

func (s *StatusPanel) portsCell() string {
	ok := lipgloss.NewStyle().Foreground(theme.Current().Success).Render("ok")
	down := lipgloss.NewStyle().Foreground(theme.Current().Error).Render("down")
	s5, h := down, down
	if s.Status != nil {
		if s.Status.SocksOK {
			s5 = ok
		}
		if s.Status.HTTPOK {
			h = ok
		}
	}
	sep := lipgloss.NewStyle().Foreground(theme.Current().Muted).Render(" · ")
	return "SOCKS " + s5 + sep + "HTTP " + h
}

func (s *StatusPanel) trafficCell() string {
	up := lipgloss.NewStyle().Foreground(theme.Current().Upload).Render("↑")
	dn := lipgloss.NewStyle().Foreground(theme.Current().Accent).Render("↓")
	cell := fmt.Sprintf("%s %s %s %s", dn, core.FormatBytes(s.Traffic.Downlink), up, core.FormatBytes(s.Traffic.Uplink))
	if s.TodayBytes > 0 {
		sep := lipgloss.NewStyle().Foreground(theme.Current().Muted).Render(" · ")
		cell += sep + "today " + core.FormatBytes(s.TodayBytes)
	}
	return cell
}

func (s *StatusPanel) sparklineLines() []dashLine {
	width := s.Width - 4
	if width < 8 {
		width = 8
	}
	var ring *sparkline.Ring
	var label, value, other string
	barStyle := lipgloss.NewStyle().Foreground(theme.Current().Accent)
	switch s.Metric {
	case MetricLatency:
		ring, label, other, value = s.LatencyRing, "Latency", "speed", s.Latency
		switch ms := ring.Last(); {
		case ms < 100:
			barStyle = lipgloss.NewStyle().Foreground(theme.Current().Success)
		case ms < 300:
			barStyle = lipgloss.NewStyle().Foreground(theme.Current().Selected)
		default:
			barStyle = lipgloss.NewStyle().Foreground(theme.Current().Error)
		}
	default:
		ring, label, other, value = s.SpeedRing, "Download", "latency", s.DnSpeed
	}

	mk := s.MetricKey
	if mk == "" {
		mk = "m"
	}
	hint := mk + " → " + other
	if !s.Focused {
		hint = "Tab → " + mk
	}
	header := label
	if value != "" {
		header += "  " + value
	}
	hintStyled := lipgloss.NewStyle().Foreground(theme.Current().Muted).Render(hint)
	gap := width - lipgloss.Width(header) - lipgloss.Width(hint)
	if gap < 1 {
		gap = 1
	}
	headerLine := "  " + header + strings.Repeat(" ", gap) + hintStyled
	barLine := "  " + barStyle.Render(sparkline.Render(ring.Values(), width))
	return []dashLine{
		{headerLine, grpSpark, 20},
		{barLine, grpSpark, 20},
	}
}

func (s *StatusPanel) renderTopology() string {
	if s.ActiveProfile == nil {
		return ""
	}
	node := lipgloss.NewStyle().Foreground(theme.Current().Accent)
	arrow := lipgloss.NewStyle().Foreground(theme.Current().Muted).Render(" → ")
	socks := 10808
	if s.Settings != nil {
		socks = s.Settings.Local.SocksPort
	}
	segs := []string{node.Render("you"), node.Render(fmt.Sprintf(":%d", socks))}
	if s.ActiveProfile.IsChained() {
		for _, srv := range s.ActiveProfile.ChainServers() {
			segs = append(segs, node.Render(fmt.Sprintf("%s:%d", srv.Address, srv.Port)))
		}
	} else {
		segs = append(segs,
			node.Render("["+strings.ToUpper(s.ActiveProfile.Server.GetProtocol())+"]"),
			node.Render(fmt.Sprintf("%s:%d", s.ActiveProfile.Server.Address, s.ActiveProfile.Server.Port)))
	}
	exit := "exit"
	if s.ExitIP != "" {
		exit = s.ExitIP
	}
	segs = append(segs, node.Render(exit))
	strip := "  " + strings.Join(segs, arrow)
	if s.Width > 0 && s.Width < 28 && len(segs) >= 2 {
		// keep the endpoint (last meaningful node) when there is no room
		short := segs[len(segs)-2] // [PROTO] or last hop
		return "  …" + arrow + short
	}
	return strip
}

// formatLatencyIndicator returns a color-coded latency string.
func formatLatencyIndicator(latency string) string {
	ms := parseLatencyMS(latency)
	green := lipgloss.NewStyle().Foreground(theme.Current().Success)
	yellow := lipgloss.NewStyle().Foreground(theme.Current().Selected)
	red := lipgloss.NewStyle().Foreground(theme.Current().Error)

	switch {
	case ms < 100:
		return green.Render("● " + latency)
	case ms < 300:
		return yellow.Render("◐ " + latency)
	default:
		return red.Render("○ " + latency)
	}
}

// parseLatencyMS extracts milliseconds from a latency string like "45ms".
func parseLatencyMS(s string) int {
	s = strings.TrimSuffix(s, "ms")
	var ms int
	fmt.Sscanf(s, "%d", &ms)
	return ms
}
