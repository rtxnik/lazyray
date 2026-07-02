package modals

import (
	"strings"
	"testing"

	"github.com/rtxnik/lazyray/internal/tui/commands"
)

// The help overlay is generated from the registry: every command Title and every
// category header must appear. This is the structural anti-drift guarantee for help.
func TestHelpModalRendersEveryCommandAndCategory(t *testing.T) {
	reg := commands.New(commands.DefaultKeyMap())
	out := NewHelpModal(reg, 120, 40).View()

	for _, c := range reg.All() {
		if !strings.Contains(out, c.Title) {
			t.Errorf("help modal missing command title %q", c.Title)
		}
	}
	for _, g := range reg.ByCategory() {
		if !strings.Contains(out, g.Title) {
			t.Errorf("help modal missing category header %q", g.Title)
		}
	}
}
