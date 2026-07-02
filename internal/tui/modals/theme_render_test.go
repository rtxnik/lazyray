package modals

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
	"github.com/rtxnik/lazyray/internal/tui/commands"
	"github.com/rtxnik/lazyray/internal/tui/theme"
)

// A modal rendered under a non-default theme must use that theme's accent and
// must NOT leak the gruvbox accent — proving the theme wiring is live end-to-end.
func TestHelpModalRespectsActiveTheme(t *testing.T) {
	old := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() {
		lipgloss.SetColorProfile(old)
		theme.Set("gruvbox-dark")
	})

	theme.Set("nord")
	nord := NewHelpModal(commands.New(commands.DefaultKeyMap()), 80, 24).View()
	if !strings.Contains(nord, "136;192;208") {
		t.Error("nord help modal missing nord accent ansi (136;192;208)")
	}
	if strings.Contains(nord, "142;192;124") {
		t.Error("nord help modal leaks gruvbox accent ansi (142;192;124)")
	}

	theme.Set("gruvbox-dark")
	gruv := NewHelpModal(commands.New(commands.DefaultKeyMap()), 80, 24).View()
	if !strings.Contains(gruv, "142;192;124") {
		t.Error("gruvbox help modal missing gruvbox accent ansi (142;192;124)")
	}
}
