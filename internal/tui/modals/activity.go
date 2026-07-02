package modals

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/rtxnik/lazyray/internal/tui/notify"
	"github.com/rtxnik/lazyray/internal/tui/theme"
)

// ActivityModal shows the durable in-memory notice log, newest first.
type ActivityModal struct {
	Done     bool
	entries  []notify.Notice
	viewport viewport.Model
	width    int
	height   int
}

// NewActivityModal builds the overlay from a newest-first slice of notices.
func NewActivityModal(entries []notify.Notice, width, height int) *ActivityModal {
	mw := 64
	if width > 0 && width-4 < mw {
		mw = width - 4
	}
	if mw < 30 {
		mw = 30
	}
	mh := 14
	if height > 0 && height-8 < mh {
		mh = height - 8
	}
	if mh < 4 {
		mh = 4
	}
	m := &ActivityModal{entries: entries, viewport: viewport.New(mw, mh), width: width, height: height}
	m.viewport.SetContent(m.renderEntries())
	return m
}

func (m *ActivityModal) Init() tea.Cmd { return nil }

func (m *ActivityModal) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if km, ok := msg.(tea.KeyMsg); ok {
		switch km.String() {
		case "esc", "q", "n", "enter":
			m.Done = true
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

// severityStyle maps a notice severity to a theme style (color only, no glyph).
func severityStyle(s notify.Severity) lipgloss.Style {
	st := theme.CurrentStyles()
	switch s {
	case notify.Error:
		return st.Error
	case notify.Warning:
		return st.Warning
	case notify.Success:
		return st.Success
	case notify.Info:
		return st.Info
	}
	return st.Muted
}

func (m *ActivityModal) renderEntries() string {
	if len(m.entries) == 0 {
		return theme.CurrentStyles().Muted.Render("No activity yet.")
	}
	hintStyle := theme.CurrentStyles().Muted
	var b strings.Builder
	for i, n := range m.entries {
		if i > 0 {
			b.WriteString("\n")
		}
		tag := severityStyle(n.Severity).Render(fmt.Sprintf("%-5s", n.Severity.Tag()))
		line := fmt.Sprintf("%s  %s  %s", n.Time.Format("15:04:05"), tag, n.Message)
		if n.Count > 1 {
			line += fmt.Sprintf(" ×%d", n.Count)
		}
		b.WriteString(line)
		if n.Hint != "" {
			b.WriteString("\n")
			b.WriteString(hintStyle.Render("            ↳ " + n.Hint))
		}
	}
	return b.String()
}

func (m *ActivityModal) View() string {
	title := theme.CurrentStyles().Title.Render("Activity")
	hint := theme.CurrentStyles().Muted.Render("[↑↓] scroll   [Esc] close")
	body := lipgloss.JoinVertical(lipgloss.Left, title, "", m.viewport.View(), "", hint)
	return lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(theme.Current().Accent).
		Padding(1, 2).
		Render(body)
}
