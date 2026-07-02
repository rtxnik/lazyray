package modals

import (
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/rtxnik/lazyray/internal/tui/theme"
)

// DiffModal shows a config diff between current and new profile.
type DiffModal struct {
	viewport viewport.Model
	title    string
	Done     bool
	width    int
	height   int
	ready    bool
}

// NewDiffModal creates a diff display modal.
func NewDiffModal(title string, oldLines, newLines []string, width, height int) *DiffModal {
	m := &DiffModal{
		title:  title,
		width:  width,
		height: height,
	}

	diff := computeDiff(oldLines, newLines)

	modalWidth := width - 8
	if modalWidth > 80 {
		modalWidth = 80
	}
	if modalWidth < 30 {
		modalWidth = 30
	}

	vpHeight := height - 10
	if vpHeight < 5 {
		vpHeight = 5
	}

	vp := viewport.New(modalWidth-4, vpHeight)
	vp.Style = lipgloss.NewStyle()
	vp.SetContent(diff)

	m.viewport = vp
	m.ready = true
	return m
}

func (m *DiffModal) Init() tea.Cmd {
	return nil
}

func (m *DiffModal) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.KeyMsg); ok {
		switch msg.String() {
		case "esc", "enter", "q":
			m.Done = true
			return m, nil
		case "up", "k":
			m.viewport.ScrollUp(1)
			return m, nil
		case "down", "j":
			m.viewport.ScrollDown(1)
			return m, nil
		}
	}
	return m, nil
}

func (m *DiffModal) View() string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.Current().Accent).
		MarginBottom(1)

	modalWidth := m.width - 8
	if modalWidth > 80 {
		modalWidth = 80
	}
	if modalWidth < 30 {
		modalWidth = 30
	}

	modal := lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(theme.Current().Accent).
		Padding(1, 2).
		Width(modalWidth)

	var b strings.Builder
	b.WriteString(titleStyle.Render(m.title))
	b.WriteString("\n\n")
	b.WriteString(m.viewport.View())
	b.WriteString("\n\n")
	hint := lipgloss.NewStyle().Foreground(theme.Current().Muted)
	b.WriteString(hint.Render("[↑↓] Scroll  [Esc] Close"))

	return modal.Render(b.String())
}

// computeDiff produces a simple line-by-line diff with color coding.
func computeDiff(oldLines, newLines []string) string {
	addStyle := lipgloss.NewStyle().Foreground(theme.Current().Success)
	removeStyle := lipgloss.NewStyle().Foreground(theme.Current().Error)
	sameStyle := lipgloss.NewStyle().Foreground(theme.Current().Muted)

	// Build lookup for old lines
	oldSet := make(map[string]bool, len(oldLines))
	for _, l := range oldLines {
		oldSet[l] = true
	}
	newSet := make(map[string]bool, len(newLines))
	for _, l := range newLines {
		newSet[l] = true
	}

	var result []string

	// Show removed lines
	for _, l := range oldLines {
		if !newSet[l] {
			result = append(result, removeStyle.Render("- "+l))
		}
	}

	// Show new/same lines
	for _, l := range newLines {
		if !oldSet[l] {
			result = append(result, addStyle.Render("+ "+l))
		} else {
			result = append(result, sameStyle.Render("  "+l))
		}
	}

	if len(result) == 0 {
		return sameStyle.Render("No changes")
	}

	return strings.Join(result, "\n")
}
