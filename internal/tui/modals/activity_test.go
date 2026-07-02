package modals

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/rtxnik/lazyray/internal/tui/notify"
)

func TestActivityModalRendersNewestFirstWithTagsAndHint(t *testing.T) {
	entries := []notify.Notice{
		{ID: 2, Time: time.Date(2026, 6, 23, 12, 4, 31, 0, time.UTC), Severity: notify.Error, Message: "connect failed", Hint: "free port 1080"},
		{ID: 1, Time: time.Date(2026, 6, 23, 12, 3, 50, 0, time.UTC), Severity: notify.Success, Message: "profile active"},
	}
	out := NewActivityModal(entries, 80, 24).View()
	if !strings.Contains(out, "ERROR") || !strings.Contains(out, "connect failed") {
		t.Errorf("view missing error entry:\n%s", out)
	}
	if !strings.Contains(out, "free port 1080") {
		t.Errorf("view missing hint line:\n%s", out)
	}
	if !strings.Contains(out, "Activity") {
		t.Errorf("view missing title:\n%s", out)
	}
	// Newest-first: the error (ID 2) must render before the success (ID 1).
	if strings.Index(out, "connect failed") > strings.Index(out, "profile active") {
		t.Errorf("entries not rendered newest-first:\n%s", out)
	}
}

func TestActivityModalEmpty(t *testing.T) {
	out := NewActivityModal(nil, 80, 24).View()
	if !strings.Contains(out, "No activity yet.") {
		t.Errorf("empty view should say no activity:\n%s", out)
	}
}

func TestActivityModalClosesOnEscAndQ(t *testing.T) {
	for _, k := range []tea.KeyMsg{
		{Type: tea.KeyEsc},
		{Type: tea.KeyRunes, Runes: []rune{'q'}},
		{Type: tea.KeyRunes, Runes: []rune{'n'}},
	} {
		m := NewActivityModal(nil, 80, 24)
		m.Update(k)
		if !m.Done {
			t.Errorf("key %v should set Done", k)
		}
	}
}
