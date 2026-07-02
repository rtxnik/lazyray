package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/rtxnik/lazyray/internal/tui/theme"
)

// Theme-derived color shortcuts for the tui root package. panels/ and modals/
// read theme.Current() directly; these locals back app.go's status/hotkey bars
// and the panel border/title helpers. Refreshed by applyTheme() after SetTheme().
var (
	colorYellow   = theme.Current().Yellow
	colorAquaBr   = theme.Current().Accent
	colorOrangeBr = theme.Current().Orange
	colorGray     = theme.Current().Muted

	// Panel borders
	activeBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(theme.Current().BorderActive)

	inactiveBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(theme.Current().BorderInactive)

	// Status bar
	styleStatusBar = lipgloss.NewStyle().
			Foreground(theme.Current().Fg).
			Background(theme.Current().BgAlt).
			Padding(0, 1)
)

// SetTheme switches the active theme and refreshes tui-local derived styles.
func SetTheme(name string) {
	theme.Set(name)
	applyTheme()
}

// applyTheme refreshes the tui-local derived styles from the active theme.
// Field mapping mirrors the original implementation exactly.
func applyTheme() {
	t := theme.Current()
	colorYellow = t.Yellow
	colorAquaBr = t.Accent
	colorOrangeBr = t.Orange
	colorGray = t.Muted

	activeBorder = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.BorderActive)
	inactiveBorder = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.BorderInactive)
	styleStatusBar = lipgloss.NewStyle().
		Foreground(t.Fg).
		Background(t.BgAlt).
		Padding(0, 1)
}

func panelStyle(active bool, width, height int) lipgloss.Style {
	style := inactiveBorder
	if active {
		style = activeBorder
	}
	return style.Width(width).Height(height)
}

// renderPanelWithTitle rebuilds the top border of a rendered panel to include a title.
// The title is placed after the top-left corner: ╭─ Title ─────╮
// Instead of extracting pieces from the ANSI-styled border (which causes [0m artifacts
// due to byte/visual width mismatch in ansi.Truncate), we rebuild the top line from
// scratch using known border characters and colors.
func renderPanelWithTitle(rendered, title string, active bool) string {
	if title == "" {
		return rendered
	}

	lines := strings.Split(rendered, "\n")
	if len(lines) == 0 {
		return rendered
	}

	// Style the title text
	var titleStyle lipgloss.Style
	var borderColor lipgloss.Color
	if active {
		titleStyle = lipgloss.NewStyle().Bold(true).Foreground(colorAquaBr)
		borderColor = colorAquaBr
	} else {
		titleStyle = lipgloss.NewStyle().Foreground(colorGray)
		borderColor = colorGray
	}

	styledTitle := titleStyle.Render(title)
	titleVisualWidth := lipgloss.Width(styledTitle)

	topWidth := lipgloss.Width(lines[0])

	// We need at least: corner(1) + dash(1) + space(1) + title + space(1) + dash(0+) + corner(1)
	if titleVisualWidth+5 > topWidth {
		return rendered
	}

	border := lipgloss.RoundedBorder()
	dashStyle := lipgloss.NewStyle().Foreground(borderColor)

	// Layout: corner + dash + space + title + space + remaining_dashes + corner
	// remaining_dashes = topWidth - 5 - titleVisualWidth
	remainingDashes := topWidth - 5 - titleVisualWidth
	if remainingDashes < 0 {
		remainingDashes = 0
	}

	newTop := dashStyle.Render(border.TopLeft+border.Top+" ") +
		styledTitle +
		dashStyle.Render(" "+strings.Repeat(border.Top, remainingDashes)+border.TopRight)

	lines[0] = newTop
	return strings.Join(lines, "\n")
}
