package modals

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/rtxnik/lazyray/internal/tui/commands"
)

func launchableFixture() []commands.Command {
	return commands.New(commands.DefaultKeyMap()).Launchable()
}

func typeRunes(m *PaletteModal, s string) {
	for _, r := range s {
		model, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = model.(*PaletteModal)
	}
}

func TestPaletteEmptyQueryListsAllLaunchable(t *testing.T) {
	m := NewPaletteModal(launchableFixture(), 80, 24)
	if len(m.filtered) != len(launchableFixture()) {
		t.Errorf("empty query filtered = %d, want %d (all launchable)", len(m.filtered), len(launchableFixture()))
	}
}

func TestPaletteFuzzyFilterNarrows(t *testing.T) {
	m := NewPaletteModal(launchableFixture(), 80, 24)
	typeRunes(m, "diag")
	if len(m.filtered) != 1 {
		t.Fatalf("query \"diag\" filtered = %d, want 1", len(m.filtered))
	}
	if m.filtered[0].ID != "Doctor" {
		t.Errorf("top match = %q, want Doctor", m.filtered[0].ID)
	}
}

func TestPaletteEnterSelectsCursor(t *testing.T) {
	m := NewPaletteModal(launchableFixture(), 80, 24)
	typeRunes(m, "diag")
	model, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = model.(*PaletteModal)
	if !m.Done {
		t.Fatal("Enter must set Done")
	}
	if m.Selected == nil || m.Selected.ID != "Doctor" {
		t.Fatalf("Selected = %v, want Doctor", m.Selected)
	}
}

func TestPaletteDownMovesCursor(t *testing.T) {
	m := NewPaletteModal(launchableFixture(), 80, 24)
	model, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = model.(*PaletteModal)
	if m.cursor != 1 {
		t.Errorf("cursor after Down = %d, want 1", m.cursor)
	}
}

func TestPaletteEscLeavesSelectionNil(t *testing.T) {
	m := NewPaletteModal(launchableFixture(), 80, 24)
	model, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = model.(*PaletteModal)
	if !m.Done {
		t.Fatal("Esc must set Done")
	}
	if m.Selected != nil {
		t.Errorf("Esc must leave Selected nil, got %v", m.Selected)
	}
}

func TestPaletteViewRendersKeyColumn(t *testing.T) {
	m := NewPaletteModal(launchableFixture(), 80, 24)
	typeRunes(m, "latency")
	out := m.View()
	if !strings.Contains(out, "test all latency") {
		t.Error("View must show the command title")
	}
	// TestAll's bound key is uppercase "T", which does not occur in the
	// lowercase title — so this genuinely asserts the key column is rendered,
	// not merely a letter that the title happens to share.
	if !strings.Contains(out, "T") {
		t.Error("View must show the bound key column")
	}
}

// TestHighlightTitleMultiByteRunesPreserved guards the byte-offset walk in
// highlightTitle: fuzzy.Match.MatchedIndexes are byte offsets, so the function
// must decode runes by byte offset. With a multi-byte rune ('é' = 2 bytes) the
// byte offsets diverge from rune indices; either way every rune must be emitted
// exactly once (no drop/dup/panic). Styling is stripped in the test env, so the
// rendered output equals the plain title.
func TestHighlightTitleMultiByteRunesPreserved(t *testing.T) {
	title := "café ping" // 'é' is 2 bytes; 'p' sits at byte offset 6, rune index 5
	plain := lipgloss.NewStyle()
	out := highlightTitle(title, []int{0, 6}, plain, plain)
	if out != title {
		t.Errorf("highlightTitle altered runes: got %q, want %q", out, title)
	}
}

// TestPaletteWindowScrollsWithCursor covers clampWindow: with more launchable
// commands (23) than the visible window (8), moving the cursor past the window
// must advance offset and keep the cursor inside [offset, offset+window).
func TestPaletteWindowScrollsWithCursor(t *testing.T) {
	m := NewPaletteModal(launchableFixture(), 80, 24)
	for i := 0; i < 9; i++ {
		model, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
		m = model.(*PaletteModal)
	}
	if m.cursor != 9 {
		t.Fatalf("cursor = %d, want 9 after 9 Downs", m.cursor)
	}
	if m.offset == 0 {
		t.Errorf("offset must advance once the cursor passes the window; got offset=0 with cursor=%d window=%d", m.cursor, paletteWindow)
	}
	if m.cursor < m.offset || m.cursor >= m.offset+paletteWindow {
		t.Errorf("cursor %d outside visible window [%d,%d)", m.cursor, m.offset, m.offset+paletteWindow)
	}
}
