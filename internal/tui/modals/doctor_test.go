package modals

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
	"github.com/rtxnik/lazyray/internal/doctor"
	"github.com/rtxnik/lazyray/internal/tui/theme"
)

// keyMsg builds a tea.KeyMsg for the given key string. Named keys map to their
// special tea.KeyType; everything else is treated as literal runes.
func keyMsg(s string) tea.KeyMsg {
	switch s {
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	default:
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
	}
}

func sampleReport() *doctor.Report {
	return &doctor.Report{
		Checks: []doctor.Result{
			{Group: "install", Name: "xray binary", Severity: doctor.SeverityOK, Detail: "/usr/bin/xray"},
			{Group: "config", Name: "local port", Severity: doctor.SeverityWarn, Detail: "8080 in use", Hint: "choose a free port"},
			{Group: "connectivity", Name: "exit IP", Severity: doctor.SeverityFail, Detail: "timeout", Hint: "check the server"},
		},
		Summary: doctor.Summary{OK: 1, Warn: 1, Fail: 1},
	}
}

func TestDoctorModalRendersGroupsHintsSummary(t *testing.T) {
	rep := sampleReport()
	m := NewDoctorModal(80, 24, func() *doctor.Report { return rep })
	m.report = rep
	m.loading = false

	body := m.renderReport()
	for _, want := range []string{"install", "config", "connectivity", "xray binary", "local port", "exit IP"} {
		if !strings.Contains(body, want) {
			t.Errorf("renderReport missing %q\n%s", want, body)
		}
	}
	// Hint lines only under WARN/FAIL, never under OK.
	if !strings.Contains(body, "→ try: choose a free port") || !strings.Contains(body, "→ try: check the server") {
		t.Errorf("renderReport missing hint lines\n%s", body)
	}
	if n := strings.Count(body, "→ try:"); n != 2 {
		t.Errorf("expected exactly 2 hint lines, got %d\n%s", n, body)
	}
	if s := m.summary(); s != "1 OK · 1 WARN · 1 FAIL" {
		t.Errorf("summary = %q, want %q", s, "1 OK · 1 WARN · 1 FAIL")
	}
}

func TestDoctorModalLoadingThenClose(t *testing.T) {
	m := NewDoctorModal(80, 24, func() *doctor.Report { return sampleReport() })
	if !m.loading {
		t.Fatal("new doctor modal should start loading")
	}
	if !strings.Contains(m.View(), "Running diagnostics") {
		t.Errorf("loading view missing spinner text\n%s", m.View())
	}
	// Delivering the report clears loading.
	m.Update(doctorDoneMsg{report: sampleReport()})
	if m.loading || m.report == nil {
		t.Fatal("doctorDoneMsg should clear loading and set report")
	}
	// esc closes.
	m.Update(keyMsg("esc"))
	if !m.Done {
		t.Error("esc should set Done")
	}
}

func TestDoctorModalRespectsActiveTheme(t *testing.T) {
	old := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() {
		lipgloss.SetColorProfile(old)
		theme.Set("gruvbox-dark")
	})
	render := func() string {
		m := NewDoctorModal(80, 24, func() *doctor.Report { return sampleReport() })
		m.report = sampleReport()
		m.loading = false
		m.viewport.SetContent(m.renderReport())
		return m.View()
	}
	theme.Set("nord")
	if nord := render(); !strings.Contains(nord, "136;192;208") {
		t.Errorf("nord doctor modal missing nord accent ansi\n%s", nord)
	}
	theme.Set("gruvbox-dark")
	if gruv := render(); !strings.Contains(gruv, "142;192;124") {
		t.Errorf("gruvbox doctor modal missing gruvbox accent ansi\n%s", gruv)
	}
}
