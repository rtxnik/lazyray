package theme

import "github.com/charmbracelet/lipgloss"

// Styles is a snapshot of ready-made semantic styles for a theme. New TUI
// surfaces read these so color stays single-sourced through the theme package.
type Styles struct {
	Title          lipgloss.Style
	Key            lipgloss.Style
	Desc           lipgloss.Style
	Hint           lipgloss.Style
	Border         lipgloss.Style
	BorderInactive lipgloss.Style
	Error          lipgloss.Style
	Success        lipgloss.Style
	Warning        lipgloss.Style
	Info           lipgloss.Style
	Accent         lipgloss.Style
	Muted          lipgloss.Style
	Selected       lipgloss.Style
}

// Styles builds the semantic style snapshot for this theme.
func (t Theme) Styles() Styles {
	return Styles{
		Title:          lipgloss.NewStyle().Bold(true).Foreground(t.Accent),
		Key:            lipgloss.NewStyle().Bold(true).Foreground(t.Accent),
		Desc:           lipgloss.NewStyle().Foreground(t.Fg),
		Hint:           lipgloss.NewStyle().Foreground(t.Muted),
		Border:         lipgloss.NewStyle().Foreground(t.BorderActive),
		BorderInactive: lipgloss.NewStyle().Foreground(t.BorderInactive),
		Error:          lipgloss.NewStyle().Foreground(t.Error),
		Success:        lipgloss.NewStyle().Foreground(t.Success),
		Warning:        lipgloss.NewStyle().Foreground(t.Warning),
		Info:           lipgloss.NewStyle().Foreground(t.Info),
		Accent:         lipgloss.NewStyle().Foreground(t.Accent),
		Muted:          lipgloss.NewStyle().Foreground(t.Muted),
		Selected:       lipgloss.NewStyle().Bold(true).Foreground(t.Selected),
	}
}

// CurrentStyles returns the semantic style snapshot for the active theme.
func CurrentStyles() Styles { return active.Styles() }
