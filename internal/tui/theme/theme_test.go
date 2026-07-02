package theme

import (
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestCurrentDefaultsToGruvbox(t *testing.T) {
	t.Cleanup(func() { Set("gruvbox-dark") })
	if got := Current().Name; got != "gruvbox-dark" {
		t.Fatalf("default theme = %q, want gruvbox-dark", got)
	}
	if got := Current().Accent; got != lipgloss.Color("#8ec07c") {
		t.Errorf("gruvbox Accent = %v, want #8ec07c", got)
	}
}

func TestSetSwitchesTheme(t *testing.T) {
	t.Cleanup(func() { Set("gruvbox-dark") })
	Set("nord")
	if got := Current().Accent; got != lipgloss.Color("#88C0D0") {
		t.Errorf("nord Accent = %v, want #88C0D0", got)
	}
	Set("unknown-xyz") // unknown name must NOT change the active theme
	if got := Current().Name; got != "nord" {
		t.Errorf("unknown theme changed active to %q, want nord unchanged", got)
	}
	Set("dark") // legacy alias
	if got := Current().Name; got != "gruvbox-dark" {
		t.Errorf("'dark' alias = %q, want gruvbox-dark", got)
	}
}

func TestNewBrightFieldsPresentForEveryTheme(t *testing.T) {
	want := map[string][3]lipgloss.Color{ // {Selected, Chain, Upload}
		"gruvbox-dark": {"#fabd2f", "#83a598", "#d3869b"},
		"nord":         {"#EBCB8B", "#88C0D0", "#B48EAD"},
		"catppuccin":   {"#F9E2AF", "#94E2D5", "#CBA6F7"},
		"solarized":    {"#b58900", "#2aa198", "#6c71c4"},
	}
	t.Cleanup(func() { Set("gruvbox-dark") })
	for name, w := range want {
		Set(name)
		c := Current()
		if c.Selected != w[0] || c.Chain != w[1] || c.Upload != w[2] {
			t.Errorf("%s bright fields = {%v,%v,%v}, want {%v,%v,%v}",
				name, c.Selected, c.Chain, c.Upload, w[0], w[1], w[2])
		}
	}
}

func TestNamesListsAllFour(t *testing.T) {
	if got := Names(); len(got) != 4 {
		t.Fatalf("Names() = %v, want 4 entries", got)
	}
}

func TestAllThemesHaveInfo(t *testing.T) {
	for _, name := range Names() {
		Set(name)
		if Current().Info == "" {
			t.Errorf("theme %q has empty Info color", name)
		}
	}
	Set("gruvbox-dark")
}

func TestInfoIsDistinctFromAccent(t *testing.T) {
	// Info must be its own token, not aliased to Accent, in every theme.
	for _, name := range Names() {
		Set(name)
		if Current().Info == Current().Accent {
			t.Errorf("theme %q: Info equals Accent (%v); use a distinct blue", name, Current().Info)
		}
	}
	Set("gruvbox-dark")
}

func TestStylesInfoUsesThemeInfo(t *testing.T) {
	Set("nord")
	defer Set("gruvbox-dark")
	if got := CurrentStyles().Info.GetForeground(); got != Current().Info {
		t.Errorf("Styles().Info fg = %v, want %v", got, Current().Info)
	}
}
