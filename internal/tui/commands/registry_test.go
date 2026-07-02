package commands

import (
	"reflect"
	"testing"

	"github.com/charmbracelet/bubbles/key"
)

func ids(cmds []Command) []string {
	out := make([]string, 0, len(cmds))
	for _, c := range cmds {
		out = append(out, c.ID)
	}
	return out
}

// New must source each Binding from the (possibly overridden) KeyMap, so a
// keys.yaml rebind shows up everywhere — proven without touching the filesystem.
func TestNewReflectsOverriddenBinding(t *testing.T) {
	km := DefaultKeyMap()
	km.Start = key.NewBinding(key.WithKeys("z"), key.WithHelp("z", "start/stop"))
	reg := New(km)
	for _, c := range reg {
		if c.ID == "Start" {
			if got := KeyDisplay(c.Binding); got != "z" {
				t.Fatalf("Start KeyDisplay = %q, want %q", got, "z")
			}
			return
		}
	}
	t.Fatal("Start command not found")
}

func TestKeyDisplayMultiKeyAndSpace(t *testing.T) {
	reg := New(DefaultKeyMap())
	got := map[string]string{}
	for _, c := range reg {
		got[c.ID] = KeyDisplay(c.Binding)
	}
	want := map[string]string{
		"Up":             "up/k",
		"ShiftUp":        "shift+up/K",
		"ToggleCollapse": "space",
		"Start":          "s",
	}
	for id, w := range want {
		if got[id] != w {
			t.Errorf("%s KeyDisplay = %q, want %q", id, got[id], w)
		}
	}
}

func TestBarItemsPreserveCuratedOrder(t *testing.T) {
	reg := New(DefaultKeyMap())
	wantWide := []string{"Start", "Restart", "Doctor", "Import", "Subscriptions", "TestAll", "Duplicate", "FilterGroup", "RoutingEdit", "Help", "Quit"}
	if got := ids(reg.BarItems(false)); !reflect.DeepEqual(got, wantWide) {
		t.Errorf("wide bar = %v, want %v", got, wantWide)
	}
	wantNarrow := []string{"Start", "Restart", "Doctor", "Help", "Quit"}
	if got := ids(reg.BarItems(true)); !reflect.DeepEqual(got, wantNarrow) {
		t.Errorf("narrow bar = %v, want %v", got, wantNarrow)
	}
}

func TestByCategoryOrderAndCompleteness(t *testing.T) {
	reg := New(DefaultKeyMap())
	groups := reg.ByCategory()
	wantTitles := []string{"General & Navigation", "Connection", "Profiles", "Inspect & Export", "Config & Logs"}
	if len(groups) != len(wantTitles) {
		t.Fatalf("got %d groups, want %d", len(groups), len(wantTitles))
	}
	total := 0
	for i, g := range groups {
		if g.Title != wantTitles[i] {
			t.Errorf("group %d title = %q, want %q", i, g.Title, wantTitles[i])
		}
		total += len(g.Commands)
	}
	if total != len(reg) {
		t.Errorf("ByCategory covers %d commands, want %d", total, len(reg))
	}
}

func TestLaunchableExcludesNavigation(t *testing.T) {
	reg := New(DefaultKeyMap())
	launch := reg.Launchable()
	// Derive the count from the registry so it self-maintains as commands are
	// added, and so a stale entry in paletteExclude (an ID not in the registry)
	// is caught as a length mismatch rather than silently ignored.
	if want := len(reg) - len(paletteExclude); len(launch) != want {
		t.Fatalf("Launchable() returned %d commands, want %d (registry %d - excluded %d)", len(launch), want, len(reg), len(paletteExclude))
	}
	excluded := map[string]bool{
		"Tab": true, "ShiftTab": true, "Up": true, "Down": true,
		"Enter": true, "Escape": true, "ToggleCollapse": true,
		"ShiftUp": true, "ShiftDown": true, "Palette": true,
	}
	for _, c := range launch {
		if excluded[c.ID] {
			t.Errorf("Launchable() must not contain navigation/self command %q", c.ID)
		}
	}
}

// Guard G3 (single-rune launch key). Documented in docs/ARCHITECTURE.md
// (Invariants & Guards). Keep that section and this test in sync.
func TestLaunchablePrimaryKeyIsSingleRune(t *testing.T) {
	reg := New(DefaultKeyMap())
	for _, c := range reg.Launchable() {
		keys := c.Binding.Keys()
		if len(keys) == 0 {
			t.Fatalf("%s has no bound keys", c.ID)
		}
		if r := []rune(keys[0]); len(r) != 1 {
			t.Errorf("%s primary key %q is not a single rune; palette re-injection would break", c.ID, keys[0])
		}
	}
}

func TestPaletteCommandRegistered(t *testing.T) {
	reg := New(DefaultKeyMap())
	for _, c := range reg.All() {
		if c.ID == "Palette" {
			if KeyDisplay(c.Binding) != ":" {
				t.Errorf("Palette key = %q, want \":\"", KeyDisplay(c.Binding))
			}
			return
		}
	}
	t.Fatal("Palette command not found in registry")
}
