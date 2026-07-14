package tui

import (
	"testing"

	"github.com/atotto/clipboard"
)

func TestClipboardAutoClearGuard(t *testing.T) {
	var store string
	clipboardWrite = func(s string) error { store = s; return nil }
	clipboardRead = func() (string, error) { return store, nil }
	t.Cleanup(func() {
		clipboardWrite = clipboard.WriteAll
		clipboardRead = clipboard.ReadAll
	})

	a := &App{}
	a.copyToClipboard("vless://tok@h") // gen 1
	a.copyToClipboard("vless://tok@h") // gen 2 (re-copy of the same value)

	// The stale gen-1 tick must NOT clear the re-copied value.
	a.handleClearClipboard(clearClipboardMsg{gen: 1})
	if store == "" {
		t.Fatal("stale generation cleared the re-copied value")
	}
	// The current gen-2 tick clears it.
	a.handleClearClipboard(clearClipboardMsg{gen: 2})
	if store != "" {
		t.Fatalf("current generation did not clear: %q", store)
	}
}

func TestClipboardAutoClearLeavesChangedValue(t *testing.T) {
	var store string
	clipboardWrite = func(s string) error { store = s; return nil }
	clipboardRead = func() (string, error) { return store, nil }
	t.Cleanup(func() {
		clipboardWrite = clipboard.WriteAll
		clipboardRead = clipboard.ReadAll
	})

	a := &App{}
	a.copyToClipboard("secret") // gen 1
	store = "user typed something else"
	a.handleClearClipboard(clearClipboardMsg{gen: 1})
	if store != "user typed something else" {
		t.Fatalf("clobbered a value the user copied since: %q", store)
	}
}
