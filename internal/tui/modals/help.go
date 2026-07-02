package modals

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/rtxnik/lazyray/internal/tui/commands"
	"github.com/rtxnik/lazyray/internal/tui/theme"
)

// HelpModal displays keybinding help, generated from the command registry.
type HelpModal struct {
	Done     bool
	registry commands.Registry
	width    int
	height   int
}

// NewHelpModal creates a new help modal driven by the command registry.
func NewHelpModal(reg commands.Registry, width, height int) *HelpModal {
	return &HelpModal{registry: reg, width: width, height: height}
}

func (m *HelpModal) Init() tea.Cmd { return nil }

func (m *HelpModal) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.KeyMsg); ok {
		switch msg.String() {
		case "enter", "esc", "?", "q":
			m.Done = true
		}
	}
	return m, nil
}

func (m *HelpModal) View() string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.Current().Accent).
		MarginBottom(1)
	catStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.Current().Accent)

	modalWidth := 50
	if m.width > 0 && m.width-4 < modalWidth {
		modalWidth = m.width - 4
	}
	if modalWidth < 30 {
		modalWidth = 30
	}

	modal := lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(theme.Current().Accent).
		Padding(1, 2).
		Width(modalWidth)

	keyStyle := lipgloss.NewStyle().
		Foreground(theme.Current().Accent).
		Bold(true).
		Width(14)

	descStyle := lipgloss.NewStyle().
		Foreground(theme.Current().Fg)

	var b strings.Builder
	b.WriteString(titleStyle.Render("Keyboard Shortcuts"))
	b.WriteString("\n\n")

	for i, g := range m.registry.ByCategory() {
		if i > 0 {
			b.WriteString("\n")
		}
		b.WriteString(catStyle.Render(g.Title))
		b.WriteString("\n")
		for _, c := range g.Commands {
			fmt.Fprintf(&b, "%s %s\n",
				keyStyle.Render(commands.KeyDisplay(c.Binding)),
				descStyle.Render(c.Title))
		}
	}

	b.WriteString("\n")
	hint := lipgloss.NewStyle().Foreground(theme.Current().Muted)
	b.WriteString(hint.Render("[Enter/Esc] Close"))

	return modal.Render(b.String())
}
