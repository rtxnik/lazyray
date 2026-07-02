package commands

import (
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/key"
)

// Category groups commands for the help overlay and the future palette.
type Category int

const (
	CatNavigation Category = iota
	CatConnection
	CatProfiles
	CatInspect
	CatConfigLogs
)

func (c Category) String() string {
	switch c {
	case CatNavigation:
		return "General & Navigation"
	case CatConnection:
		return "Connection"
	case CatProfiles:
		return "Profiles"
	case CatInspect:
		return "Inspect & Export"
	case CatConfigLogs:
		return "Config & Logs"
	}
	return ""
}

// Scope describes where a command applies (descriptive metadata for help/palette).
type Scope int

const (
	ScopeGlobal Scope = iota
	ScopeProfiles
	ScopeLogs
	ScopeStatus
)

// String returns a human label for the scope (used by help/reference docs).
func (s Scope) String() string {
	switch s {
	case ScopeGlobal:
		return "Global"
	case ScopeProfiles:
		return "Profiles"
	case ScopeLogs:
		return "Logs"
	case ScopeStatus:
		return "Status"
	}
	return ""
}

// Dispatch records where the key is handled: the handleKeyPress switch, or the
// Update priority chain (esc). Descriptive; the dispatch logic itself is unchanged.
type Dispatch int

const (
	DispatchSwitch Dispatch = iota
	DispatchPriorityChain
)

// Command is metadata only. The dispatch stays in app.go's handleKeyPress switch.
type Command struct {
	ID         string      // equals the KeyMap field name (Guard 1 bijection)
	Binding    key.Binding // taken from the already-overridden KeyMap
	Title      string      // canonical verbose label (help / palette)
	ShortLabel string      // terse label (hotkey bar)
	Category   Category
	Scope      Scope
	Dispatch   Dispatch
	BarWide    int // 1-based position in the wide bar; 0 = absent
	BarNarrow  int // 1-based position in the narrow bar; 0 = absent
}

// Registry is the single ordered source every display surface reads.
type Registry []Command

// New builds the registry from an already-resolved KeyMap (post loadCustomKeys),
// so keys.yaml overrides are reflected in help/bar/palette automatically.
func New(km KeyMap) Registry {
	return Registry{
		{ID: "Quit", Binding: km.Quit, Title: "quit", ShortLabel: "quit", Category: CatNavigation, Scope: ScopeGlobal, Dispatch: DispatchSwitch, BarWide: 11, BarNarrow: 5},
		{ID: "Help", Binding: km.Help, Title: "help", ShortLabel: "help", Category: CatNavigation, Scope: ScopeGlobal, Dispatch: DispatchSwitch, BarWide: 10, BarNarrow: 4},
		{ID: "Activity", Binding: km.Activity, Title: "activity log", ShortLabel: "activity", Category: CatNavigation, Scope: ScopeGlobal, Dispatch: DispatchSwitch},
		{ID: "Palette", Binding: km.Palette, Title: "command palette", ShortLabel: "palette", Category: CatNavigation, Scope: ScopeGlobal, Dispatch: DispatchSwitch},
		{ID: "Tab", Binding: km.Tab, Title: "next panel", Category: CatNavigation, Scope: ScopeGlobal, Dispatch: DispatchSwitch},
		{ID: "ShiftTab", Binding: km.ShiftTab, Title: "prev panel", Category: CatNavigation, Scope: ScopeGlobal, Dispatch: DispatchSwitch},
		{ID: "Up", Binding: km.Up, Title: "move up / scroll", Category: CatNavigation, Scope: ScopeGlobal, Dispatch: DispatchSwitch},
		{ID: "Down", Binding: km.Down, Title: "move down / scroll", Category: CatNavigation, Scope: ScopeGlobal, Dispatch: DispatchSwitch},
		{ID: "Enter", Binding: km.Enter, Title: "select / activate", Category: CatNavigation, Scope: ScopeProfiles, Dispatch: DispatchSwitch},
		{ID: "Escape", Binding: km.Escape, Title: "close / cancel", Category: CatNavigation, Scope: ScopeGlobal, Dispatch: DispatchPriorityChain},
		{ID: "ToggleCollapse", Binding: km.ToggleCollapse, Title: "collapse / expand group", Category: CatNavigation, Scope: ScopeProfiles, Dispatch: DispatchSwitch},
		{ID: "ToggleMetric", Binding: km.ToggleMetric, Title: "toggle dashboard metric", Category: CatNavigation, Scope: ScopeStatus, Dispatch: DispatchSwitch},
		{ID: "Start", Binding: km.Start, Title: "start / stop", ShortLabel: "start", Category: CatConnection, Scope: ScopeGlobal, Dispatch: DispatchSwitch, BarWide: 1, BarNarrow: 1},
		{ID: "Restart", Binding: km.Restart, Title: "restart", ShortLabel: "restart", Category: CatConnection, Scope: ScopeGlobal, Dispatch: DispatchSwitch, BarWide: 2, BarNarrow: 2},
		{ID: "Doctor", Binding: km.Doctor, Title: "diagnostics", ShortLabel: "doctor", Category: CatInspect, Scope: ScopeGlobal, Dispatch: DispatchSwitch, BarWide: 3, BarNarrow: 3},
		{ID: "Tunnel", Binding: km.Tunnel, Title: "SSH tunnel", Category: CatConnection, Scope: ScopeGlobal, Dispatch: DispatchSwitch},
		{ID: "TestAll", Binding: km.TestAll, Title: "test all latency", ShortLabel: "test", Category: CatConnection, Scope: ScopeProfiles, Dispatch: DispatchSwitch, BarWide: 6},
		{ID: "Import", Binding: km.Import, Title: "import config", ShortLabel: "import", Category: CatProfiles, Scope: ScopeGlobal, Dispatch: DispatchSwitch, BarWide: 4},
		{ID: "Subscriptions", Binding: km.Subscriptions, Title: "subscriptions", ShortLabel: "subs", Category: CatProfiles, Scope: ScopeGlobal, Dispatch: DispatchSwitch, BarWide: 5},
		{ID: "Duplicate", Binding: km.Duplicate, Title: "duplicate profile", ShortLabel: "dup", Category: CatProfiles, Scope: ScopeProfiles, Dispatch: DispatchSwitch, BarWide: 7},
		{ID: "Rename", Binding: km.Rename, Title: "rename profile", Category: CatProfiles, Scope: ScopeProfiles, Dispatch: DispatchSwitch},
		{ID: "Delete", Binding: km.Delete, Title: "delete profile", Category: CatProfiles, Scope: ScopeProfiles, Dispatch: DispatchSwitch},
		{ID: "FilterGroup", Binding: km.FilterGroup, Title: "filter group", ShortLabel: "group", Category: CatProfiles, Scope: ScopeProfiles, Dispatch: DispatchSwitch, BarWide: 8},
		{ID: "ShiftUp", Binding: km.ShiftUp, Title: "move profile up", Category: CatProfiles, Scope: ScopeProfiles, Dispatch: DispatchSwitch},
		{ID: "ShiftDown", Binding: km.ShiftDown, Title: "move profile down", Category: CatProfiles, Scope: ScopeProfiles, Dispatch: DispatchSwitch},
		{ID: "Export", Binding: km.Export, Title: "export VLESS URL", Category: CatInspect, Scope: ScopeGlobal, Dispatch: DispatchSwitch},
		{ID: "QRExport", Binding: km.QRExport, Title: "QR code export", Category: CatInspect, Scope: ScopeProfiles, Dispatch: DispatchSwitch},
		{ID: "ConfigDiff", Binding: km.ConfigDiff, Title: "config diff", Category: CatInspect, Scope: ScopeProfiles, Dispatch: DispatchSwitch},
		{ID: "RoutingEdit", Binding: km.RoutingEdit, Title: "routing rules", ShortLabel: "routing", Category: CatInspect, Scope: ScopeProfiles, Dispatch: DispatchSwitch, BarWide: 9},
		{ID: "EditConfig", Binding: km.EditConfig, Title: "edit config", Category: CatConfigLogs, Scope: ScopeGlobal, Dispatch: DispatchSwitch},
		{ID: "Update", Binding: km.Update, Title: "update xray", Category: CatConfigLogs, Scope: ScopeGlobal, Dispatch: DispatchSwitch},
		{ID: "ToggleLog", Binding: km.ToggleLog, Title: "edit profile / toggle log", Category: CatConfigLogs, Scope: ScopeProfiles, Dispatch: DispatchSwitch},
		{ID: "FilterLog", Binding: km.FilterLog, Title: "filter logs", Category: CatConfigLogs, Scope: ScopeLogs, Dispatch: DispatchSwitch},
		{ID: "SearchLog", Binding: km.SearchLog, Title: "search", Category: CatConfigLogs, Scope: ScopeLogs, Dispatch: DispatchSwitch},
	}
}

// All returns the commands in declaration order.
func (r Registry) All() []Command { return []Command(r) }

// CategoryGroup is one help/palette section.
type CategoryGroup struct {
	Category Category
	Title    string
	Commands []Command
}

// ByCategory returns the commands grouped in fixed category order.
func (r Registry) ByCategory() []CategoryGroup {
	order := []Category{CatNavigation, CatConnection, CatProfiles, CatInspect, CatConfigLogs}
	groups := make([]CategoryGroup, 0, len(order))
	for _, cat := range order {
		var cmds []Command
		for _, c := range r {
			if c.Category == cat {
				cmds = append(cmds, c)
			}
		}
		groups = append(groups, CategoryGroup{Category: cat, Title: cat.String(), Commands: cmds})
	}
	return groups
}

// BarItems returns the curated hotkey-bar commands in their fixed bar order.
func (r Registry) BarItems(narrow bool) []Command {
	pos := func(c Command) int {
		if narrow {
			return c.BarNarrow
		}
		return c.BarWide
	}
	var out []Command
	for _, c := range r {
		if pos(c) > 0 {
			out = append(out, c)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return pos(out[i]) < pos(out[j]) })
	return out
}

// paletteExclude lists commands the palette does NOT launch: pure
// navigation/motion driven by live keys, plus the palette key itself.
var paletteExclude = map[string]bool{
	"Tab": true, "ShiftTab": true, "Up": true, "Down": true,
	"Enter": true, "Escape": true, "ToggleCollapse": true,
	"ShiftUp": true, "ShiftDown": true, "Palette": true,
}

// Launchable returns the commands offered in the command palette — every
// registry command except pure navigation/motion and the palette key itself,
// in registry declaration order. The palette re-injects each command's primary
// key, which the single-rune invariant (see registry_test.go) guarantees safe.
func (r Registry) Launchable() []Command {
	out := make([]Command, 0, len(r))
	for _, c := range r {
		if !paletteExclude[c.ID] {
			out = append(out, c)
		}
	}
	return out
}

// KeyDisplay formats a binding's keys for display from Keys() (not Help().Key,
// which collapses to parts[0] after a keys.yaml override), preserving "up/k".
func KeyDisplay(b key.Binding) string {
	keys := b.Keys()
	out := make([]string, 0, len(keys))
	for _, k := range keys {
		if k == " " {
			k = "space"
		}
		out = append(out, k)
	}
	return strings.Join(out, "/")
}
