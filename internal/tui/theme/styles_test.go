package theme

import (
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestCurrentStylesFollowActiveTheme(t *testing.T) {
	t.Cleanup(func() { Set("gruvbox-dark") })

	Set("nord")
	if got := CurrentStyles().Title.GetForeground(); got != lipgloss.Color("#88C0D0") {
		t.Errorf("nord Title fg = %v, want nord accent #88C0D0", got)
	}
	if got := CurrentStyles().Selected.GetForeground(); got != lipgloss.Color("#EBCB8B") {
		t.Errorf("nord Selected fg = %v, want #EBCB8B", got)
	}

	Set("gruvbox-dark")
	if got := CurrentStyles().Title.GetForeground(); got != lipgloss.Color("#8ec07c") {
		t.Errorf("gruvbox Title fg = %v, want #8ec07c", got)
	}
	if !CurrentStyles().Title.GetBold() {
		t.Error("Title should be bold")
	}
}
