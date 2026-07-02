package modals

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/rtxnik/lazyray/internal/tui/theme"
)

// ConfirmModal is a generic confirmation dialog.
type ConfirmModal struct {
	Title     string
	Message   string
	Action    string
	Confirmed bool
	Done      bool
	width     int
	height    int
}

// NewConfirmModal creates a new confirmation modal.
func NewConfirmModal(title, message, action string, width, height int) *ConfirmModal {
	return &ConfirmModal{
		Title:   title,
		Message: message,
		Action:  action,
		width:   width,
		height:  height,
	}
}

func (m *ConfirmModal) Init() tea.Cmd { return nil }

func (m *ConfirmModal) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.KeyMsg); ok {
		switch msg.String() {
		case "y", "enter":
			m.Confirmed = true
			m.Done = true
		case "n", "esc":
			m.Confirmed = false
			m.Done = true
		}
	}
	return m, nil
}

func (m *ConfirmModal) View() string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.Current().Selected).
		MarginBottom(1)

	modalWidth := 50
	if m.width > 0 && m.width-4 < modalWidth {
		modalWidth = m.width - 4
	}
	if modalWidth < 30 {
		modalWidth = 30
	}

	modal := lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(theme.Current().Selected).
		Padding(1, 2).
		Width(modalWidth)

	var b strings.Builder
	b.WriteString(titleStyle.Render(m.Title))
	b.WriteString("\n\n")
	b.WriteString(fmt.Sprintf("%s\n", m.Message))
	b.WriteString("\n")

	hint := lipgloss.NewStyle().Foreground(theme.Current().Muted)
	b.WriteString(hint.Render("[y/Enter] Confirm  [n/Esc] Cancel"))

	return modal.Render(b.String())
}
