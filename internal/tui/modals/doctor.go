package modals

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/rtxnik/lazyray/internal/doctor"
	"github.com/rtxnik/lazyray/internal/tui/theme"
)

type doctorDoneMsg struct{ report *doctor.Report }

// DoctorModal renders a read-only doctor.Report: severity-graded checks grouped
// by section, each with an optional actionable hint. It never mutates state;
// remediation lives in the CLI (`lzr doctor`).
type DoctorModal struct {
	Done     bool
	report   *doctor.Report
	run      func() *doctor.Report
	viewport viewport.Model
	loading  bool
	width    int
	height   int
}

// NewDoctorModal builds the modal in a loading state. run is the injectable
// diagnostic runner; the app supplies doctor.RunAll, tests supply a fake.
func NewDoctorModal(width, height int, run func() *doctor.Report) *DoctorModal {
	mw := 64
	if width > 0 && width-4 < mw {
		mw = width - 4
	}
	if mw < 30 {
		mw = 30
	}
	mh := 16
	if height > 0 && height-8 < mh {
		mh = height - 8
	}
	if mh < 4 {
		mh = 4
	}
	return &DoctorModal{
		run:      run,
		loading:  true,
		viewport: viewport.New(mw, mh),
		width:    width,
		height:   height,
	}
}

func (m *DoctorModal) Init() tea.Cmd { return m.runCmd() }

func (m *DoctorModal) runCmd() tea.Cmd {
	return func() tea.Msg { return doctorDoneMsg{report: m.run()} }
}

func (m *DoctorModal) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case doctorDoneMsg:
		m.report = msg.report
		m.loading = false
		m.viewport.SetContent(m.renderReport())
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "enter", "esc":
			m.Done = true
			return m, nil
		case "r":
			m.loading = true
			m.report = nil
			return m, m.runCmd()
		}
	}
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

// doctorSeverityStyle maps a doctor severity to a theme style (color only, no glyph).
func doctorSeverityStyle(s doctor.Severity) lipgloss.Style {
	st := theme.CurrentStyles()
	switch s {
	case doctor.SeverityOK:
		return st.Success
	case doctor.SeverityInfo:
		return st.Info
	case doctor.SeverityWarn:
		return st.Warning
	default: // SeverityFail
		return st.Error
	}
}

func (m *DoctorModal) renderReport() string {
	if m.report == nil {
		return ""
	}
	muted := theme.CurrentStyles().Muted
	header := theme.CurrentStyles().Title
	var b strings.Builder
	group := ""
	for _, r := range m.report.Checks {
		if r.Group != group {
			if group != "" {
				b.WriteString("\n")
			}
			b.WriteString(header.Render(r.Group))
			b.WriteString("\n")
			group = r.Group
		}
		tag := doctorSeverityStyle(r.Severity).Render(fmt.Sprintf("%-4s", r.Severity.String()))
		b.WriteString(fmt.Sprintf("  %s  %s", tag, r.Name))
		if r.Detail != "" {
			b.WriteString("  " + muted.Render(r.Detail))
		}
		b.WriteString("\n")
		if r.Hint != "" && (r.Severity == doctor.SeverityWarn || r.Severity == doctor.SeverityFail) {
			b.WriteString(muted.Render("        → try: " + r.Hint))
			b.WriteString("\n")
		}
	}
	return strings.TrimRight(b.String(), "\n")
}

func (m *DoctorModal) summary() string {
	s := m.report.Summary
	parts := []string{fmt.Sprintf("%d OK", s.OK)}
	if s.Info > 0 {
		parts = append(parts, fmt.Sprintf("%d INFO", s.Info))
	}
	parts = append(parts, fmt.Sprintf("%d WARN", s.Warn), fmt.Sprintf("%d FAIL", s.Fail))
	return strings.Join(parts, " · ")
}

func (m *DoctorModal) View() string {
	title := theme.CurrentStyles().Title.Render("Diagnostics")
	muted := theme.CurrentStyles().Muted
	var body string
	if m.loading || m.report == nil {
		body = lipgloss.JoinVertical(lipgloss.Left, title, "", "  Running diagnostics...", "",
			muted.Render("[Esc] close"))
	} else {
		body = lipgloss.JoinVertical(lipgloss.Left, title, "", m.viewport.View(), "",
			muted.Render(m.summary()),
			muted.Render("[↑↓] scroll   [r] re-run   [Esc] close"))
	}
	return lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(theme.Current().Accent).
		Padding(1, 2).
		Render(body)
}
