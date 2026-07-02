package theme

import "github.com/charmbracelet/lipgloss"

// Theme defines a complete color palette for the TUI.
type Theme struct {
	Name string

	// Base colors
	Fg     lipgloss.Color
	Bg     lipgloss.Color
	BgAlt  lipgloss.Color
	Gray   lipgloss.Color
	Red    lipgloss.Color
	Green  lipgloss.Color
	Yellow lipgloss.Color
	Blue   lipgloss.Color
	Purple lipgloss.Color
	Aqua   lipgloss.Color
	Orange lipgloss.Color

	// Semantic aliases
	Success        lipgloss.Color
	Error          lipgloss.Color
	Warning        lipgloss.Color
	Info           lipgloss.Color
	Accent         lipgloss.Color
	Muted          lipgloss.Color
	BorderActive   lipgloss.Color
	BorderInactive lipgloss.Color

	// Bright UI roles — de-hardcoded from panels/modals; no exact base/semantic
	// home, so they live as their own fields to keep gruvbox output byte-identical.
	Selected lipgloss.Color // selection / highlight (was hardcoded #fabd2f)
	Chain    lipgloss.Color // proxy-chain / topology link (was hardcoded #83a598)
	Upload   lipgloss.Color // upload / up-traffic indicator (was hardcoded #d3869b)
}

// Built-in themes
var themes = map[string]Theme{
	"gruvbox-dark": {
		Name:           "gruvbox-dark",
		Fg:             lipgloss.Color("#ebdbb2"),
		Bg:             lipgloss.Color("#282828"),
		BgAlt:          lipgloss.Color("#3c3836"),
		Gray:           lipgloss.Color("#928374"),
		Red:            lipgloss.Color("#fb4934"),
		Green:          lipgloss.Color("#b8bb26"),
		Yellow:         lipgloss.Color("#d79921"),
		Blue:           lipgloss.Color("#458588"),
		Purple:         lipgloss.Color("#b16286"),
		Aqua:           lipgloss.Color("#8ec07c"),
		Orange:         lipgloss.Color("#fe8019"),
		Success:        lipgloss.Color("#b8bb26"),
		Error:          lipgloss.Color("#fb4934"),
		Warning:        lipgloss.Color("#d79921"),
		Info:           lipgloss.Color("#458588"),
		Accent:         lipgloss.Color("#8ec07c"),
		Muted:          lipgloss.Color("#928374"),
		BorderActive:   lipgloss.Color("#8ec07c"),
		BorderInactive: lipgloss.Color("#928374"),
		Selected:       lipgloss.Color("#fabd2f"),
		Chain:          lipgloss.Color("#83a598"),
		Upload:         lipgloss.Color("#d3869b"),
	},
	"nord": {
		Name:           "nord",
		Fg:             lipgloss.Color("#ECEFF4"),
		Bg:             lipgloss.Color("#2E3440"),
		BgAlt:          lipgloss.Color("#3B4252"),
		Gray:           lipgloss.Color("#4C566A"),
		Red:            lipgloss.Color("#BF616A"),
		Green:          lipgloss.Color("#A3BE8C"),
		Yellow:         lipgloss.Color("#EBCB8B"),
		Blue:           lipgloss.Color("#81A1C1"),
		Purple:         lipgloss.Color("#B48EAD"),
		Aqua:           lipgloss.Color("#88C0D0"),
		Orange:         lipgloss.Color("#D08770"),
		Success:        lipgloss.Color("#A3BE8C"),
		Error:          lipgloss.Color("#BF616A"),
		Warning:        lipgloss.Color("#EBCB8B"),
		Info:           lipgloss.Color("#81A1C1"),
		Accent:         lipgloss.Color("#88C0D0"),
		Muted:          lipgloss.Color("#4C566A"),
		BorderActive:   lipgloss.Color("#88C0D0"),
		BorderInactive: lipgloss.Color("#4C566A"),
		Selected:       lipgloss.Color("#EBCB8B"),
		Chain:          lipgloss.Color("#88C0D0"),
		Upload:         lipgloss.Color("#B48EAD"),
	},
	"catppuccin": {
		Name:           "catppuccin",
		Fg:             lipgloss.Color("#CDD6F4"),
		Bg:             lipgloss.Color("#1E1E2E"),
		BgAlt:          lipgloss.Color("#313244"),
		Gray:           lipgloss.Color("#6C7086"),
		Red:            lipgloss.Color("#F38BA8"),
		Green:          lipgloss.Color("#A6E3A1"),
		Yellow:         lipgloss.Color("#F9E2AF"),
		Blue:           lipgloss.Color("#89B4FA"),
		Purple:         lipgloss.Color("#CBA6F7"),
		Aqua:           lipgloss.Color("#94E2D5"),
		Orange:         lipgloss.Color("#FAB387"),
		Success:        lipgloss.Color("#A6E3A1"),
		Error:          lipgloss.Color("#F38BA8"),
		Warning:        lipgloss.Color("#F9E2AF"),
		Info:           lipgloss.Color("#74C7EC"),
		Accent:         lipgloss.Color("#89B4FA"),
		Muted:          lipgloss.Color("#6C7086"),
		BorderActive:   lipgloss.Color("#89B4FA"),
		BorderInactive: lipgloss.Color("#6C7086"),
		Selected:       lipgloss.Color("#F9E2AF"),
		Chain:          lipgloss.Color("#94E2D5"),
		Upload:         lipgloss.Color("#CBA6F7"),
	},
	"solarized": {
		Name:           "solarized",
		Fg:             lipgloss.Color("#839496"),
		Bg:             lipgloss.Color("#002b36"),
		BgAlt:          lipgloss.Color("#073642"),
		Gray:           lipgloss.Color("#586e75"),
		Red:            lipgloss.Color("#dc322f"),
		Green:          lipgloss.Color("#859900"),
		Yellow:         lipgloss.Color("#b58900"),
		Blue:           lipgloss.Color("#268bd2"),
		Purple:         lipgloss.Color("#6c71c4"),
		Aqua:           lipgloss.Color("#2aa198"),
		Orange:         lipgloss.Color("#cb4b16"),
		Success:        lipgloss.Color("#859900"),
		Error:          lipgloss.Color("#dc322f"),
		Warning:        lipgloss.Color("#b58900"),
		Info:           lipgloss.Color("#268bd2"),
		Accent:         lipgloss.Color("#2aa198"),
		Muted:          lipgloss.Color("#586e75"),
		BorderActive:   lipgloss.Color("#2aa198"),
		BorderInactive: lipgloss.Color("#586e75"),
		Selected:       lipgloss.Color("#b58900"),
		Chain:          lipgloss.Color("#2aa198"),
		Upload:         lipgloss.Color("#6c71c4"),
	},
}

// active is the currently active theme (process-wide; single active theme by design).
var active = themes["gruvbox-dark"]

// Set switches to the named theme. Unknown names leave the active theme unchanged;
// the legacy alias "dark" maps to gruvbox-dark.
func Set(name string) {
	if t, ok := themes[name]; ok {
		active = t
	} else if name == "dark" {
		active = themes["gruvbox-dark"]
	}
}

// Current returns the active theme.
func Current() Theme { return active }

// Names returns all available theme names.
func Names() []string {
	return []string{"gruvbox-dark", "nord", "catppuccin", "solarized"}
}
