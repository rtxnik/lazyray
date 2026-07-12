package panels

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
	"github.com/rtxnik/lazyray/internal/config"
	"github.com/rtxnik/lazyray/internal/core"
	"github.com/rtxnik/lazyray/internal/tui/theme"
)

// ProfilesPanel displays the list of server profiles.
type ProfilesPanel struct {
	Profiles []config.Profile
	Selected int
	Width    int
	Height   int

	// Teaching empty-state keys, resolved from the registry by the app.
	ImportKey string
	SubsKey   string

	// Inline rename state
	Renaming    bool
	RenameInput textinput.Model
	RenameIdx   int

	// Collapsible groups: tracks which groups are collapsed
	CollapsedGroups map[string]bool

	// Search/filter state
	Searching   bool
	SearchInput textinput.Model
	SearchQuery string

	// Moving state — true when profile is being repositioned with K/J
	Moving bool
}

// NewProfilesPanel creates a new profiles panel.
func NewProfilesPanel() ProfilesPanel {
	return ProfilesPanel{
		CollapsedGroups: make(map[string]bool),
	}
}

// SetProfiles updates the profile list.
func (p *ProfilesPanel) SetProfiles(profiles []config.Profile) {
	p.Profiles = profiles
	if p.Selected >= len(profiles) {
		p.Selected = max(0, len(profiles)-1)
	}
}

// MoveUp moves selection up.
func (p *ProfilesPanel) MoveUp() {
	if p.Selected > 0 {
		p.Selected--
	}
}

// MoveDown moves selection down.
func (p *ProfilesPanel) MoveDown() {
	if p.Selected < len(p.Profiles)-1 {
		p.Selected++
	}
}

// SelectedProfile returns the currently selected profile.
func (p *ProfilesPanel) SelectedProfile() *config.Profile {
	if len(p.Profiles) == 0 || p.Selected >= len(p.Profiles) {
		return nil
	}
	return &p.Profiles[p.Selected]
}

// MoveProfileUp swaps the selected profile with the one above.
func (p *ProfilesPanel) MoveProfileUp() bool {
	if p.Selected <= 0 || len(p.Profiles) < 2 {
		return false
	}
	p.Profiles[p.Selected], p.Profiles[p.Selected-1] = p.Profiles[p.Selected-1], p.Profiles[p.Selected]
	p.Selected--
	return true
}

// MoveProfileDown swaps the selected profile with the one below.
func (p *ProfilesPanel) MoveProfileDown() bool {
	if p.Selected >= len(p.Profiles)-1 || len(p.Profiles) < 2 {
		return false
	}
	p.Profiles[p.Selected], p.Profiles[p.Selected+1] = p.Profiles[p.Selected+1], p.Profiles[p.Selected]
	p.Selected++
	return true
}

// StartRename activates inline rename for the selected profile.
func (p *ProfilesPanel) StartRename() {
	if len(p.Profiles) == 0 || p.Selected >= len(p.Profiles) {
		return
	}
	ti := textinput.New()
	ti.SetValue(p.Profiles[p.Selected].Name)
	ti.Focus()
	ti.CharLimit = 64
	ti.Width = p.Width - 8
	if ti.Width < 10 {
		ti.Width = 10
	}
	p.Renaming = true
	p.RenameInput = ti
	p.RenameIdx = p.Selected
}

// ConfirmRename returns the new name and cancels rename mode.
func (p *ProfilesPanel) ConfirmRename() (string, bool) {
	if !p.Renaming {
		return "", false
	}
	name := strings.TrimSpace(p.RenameInput.Value())
	p.Renaming = false
	if name == "" {
		return "", false
	}
	return name, true
}

// CancelRename exits rename mode without changes.
func (p *ProfilesPanel) CancelRename() {
	p.Renaming = false
}

// ToggleGroupCollapse collapses or expands the group of the currently selected profile.
func (p *ProfilesPanel) ToggleGroupCollapse() {
	if len(p.Profiles) == 0 || p.Selected >= len(p.Profiles) {
		return
	}
	group := p.Profiles[p.Selected].Group
	if group == "" {
		return
	}
	if p.CollapsedGroups == nil {
		p.CollapsedGroups = make(map[string]bool)
	}
	p.CollapsedGroups[group] = !p.CollapsedGroups[group]
}

// isCollapsed returns true if the given profile's group is collapsed and
// the profile is not the first in its group (group headers stay visible).
func (p *ProfilesPanel) isCollapsed(idx int) bool {
	if p.CollapsedGroups == nil {
		return false
	}
	prof := p.Profiles[idx]
	if prof.Group == "" || !p.CollapsedGroups[prof.Group] {
		return false
	}
	// The first profile of each group stays visible (acts as the group header row)
	if idx > 0 && p.Profiles[idx-1].Group == prof.Group {
		return true // there's a previous profile in same group → collapse this one
	}
	return false
}

// View renders the profiles panel content.
func (p *ProfilesPanel) View() string {
	if len(p.Profiles) == 0 {
		dim := lipgloss.NewStyle().Foreground(theme.Current().Muted)
		imp := p.ImportKey
		if imp == "" {
			imp = "i"
		}
		sub := p.SubsKey
		if sub == "" {
			sub = "S"
		}
		return dim.Render("No profiles yet.\n" +
			"Profiles are saved connections. Import one with [" + imp + "],\n" +
			"or add a subscription with [" + sub + "].")
	}

	var lines []string
	contentWidth := p.Width - 4 // Account for borders and padding

	currentGroup := ""
	for i, profile := range p.Profiles {
		// Show group header if group changes
		if profile.Group != "" && profile.Group != currentGroup {
			currentGroup = profile.Group
			collapsed := p.CollapsedGroups[currentGroup]
			groupStyle := lipgloss.NewStyle().
				Foreground(theme.Current().Selected).
				Bold(true)
			arrow := "▼"
			if collapsed {
				arrow = "▶"
			}
			// Count profiles in this group
			count := 0
			for _, pp := range p.Profiles {
				if pp.Group == currentGroup {
					count++
				}
			}
			header := fmt.Sprintf("%s %s (%d)", arrow, currentGroup, count)
			lines = append(lines, groupStyle.Render(header))
		} else if profile.Group == "" && currentGroup != "" {
			currentGroup = ""
		}

		// Skip hidden profiles (collapsed group or search-filtered)
		if p.isHidden(i) {
			continue
		}

		// Show textinput for the profile being renamed
		if p.Renaming && i == p.RenameIdx {
			lines = append(lines, "> "+p.RenameInput.View())
			continue
		}

		marker := "  "
		if profile.Default {
			marker = "* "
		}

		// Latency indicator dot
		indicator := latencyIndicator(profile.Latency)

		var line string
		name := core.StripControl(profile.Name)
		latencySuffix := ""
		if profile.Latency > 0 {
			latencySuffix = fmt.Sprintf(" %dms", profile.Latency)
		} else if profile.Latency == -2 {
			latencySuffix = " n/a"
		} else if profile.Latency < 0 {
			latencySuffix = " FAIL"
		}

		// Account for indicator + space in width calculation
		maxNameLen := contentWidth - 2 - len(latencySuffix) - 2
		if maxNameLen < 4 {
			maxNameLen = 4
		}
		if contentWidth > 0 && len(name) > maxNameLen {
			name = name[:maxNameLen]
		}

		if i == p.Selected {
			if p.Moving {
				// Moving mode: inverted colors for visual feedback
				style := lipgloss.NewStyle().
					Background(theme.Current().Selected).
					Foreground(theme.Current().Bg).
					Bold(true)
				line = style.Render(fmt.Sprintf("↕ %s %s", indicator, name))
			} else {
				style := lipgloss.NewStyle().
					Foreground(theme.Current().Accent).
					Bold(true)
				line = fmt.Sprintf("> %s %s", indicator, style.Render(name))
			}
		} else {
			line = fmt.Sprintf("%s%s %s", marker, indicator, name)
		}

		if latencySuffix != "" {
			latStyle := lipgloss.NewStyle().Foreground(theme.Current().Muted)
			if profile.Latency == -1 {
				latStyle = lipgloss.NewStyle().Foreground(theme.Current().Error)
			}
			line += latStyle.Render(latencySuffix)
		}

		lines = append(lines, line)
	}

	// Show search bar at the bottom when active
	if p.Searching {
		searchStyle := lipgloss.NewStyle().Foreground(theme.Current().Selected)
		lines = append(lines, searchStyle.Render("/")+p.SearchInput.View())
	}

	return strings.Join(lines, "\n")
}

// StartSearch activates the profile search/filter.
func (p *ProfilesPanel) StartSearch() {
	ti := textinput.New()
	ti.Placeholder = "search profiles..."
	ti.Focus()
	ti.CharLimit = 64
	ti.Width = p.Width - 8
	if ti.Width < 10 {
		ti.Width = 10
	}
	p.Searching = true
	p.SearchInput = ti
	p.SearchQuery = ""
}

// ApplySearch updates the search query from the input.
func (p *ProfilesPanel) ApplySearch() {
	p.SearchQuery = strings.ToLower(strings.TrimSpace(p.SearchInput.Value()))
	// If current selection is filtered out, move to first visible
	if p.SearchQuery != "" && p.isSearchFiltered(p.Selected) {
		for i := range p.Profiles {
			if !p.isSearchFiltered(i) && !p.isCollapsed(i) {
				p.Selected = i
				return
			}
		}
	}
}

// CancelSearch exits search mode and clears the filter.
func (p *ProfilesPanel) CancelSearch() {
	p.Searching = false
	p.SearchQuery = ""
}

// isSearchFiltered returns true if the profile at idx doesn't match
// the current search query and should be hidden.
func (p *ProfilesPanel) isSearchFiltered(idx int) bool {
	if p.SearchQuery == "" {
		return false
	}
	name := strings.ToLower(p.Profiles[idx].Name)
	return !strings.Contains(name, p.SearchQuery)
}

// isHidden returns true if a profile should not be displayed
// (filtered by search or collapsed group).
func (p *ProfilesPanel) isHidden(idx int) bool {
	return p.isCollapsed(idx) || p.isSearchFiltered(idx)
}

// MoveUpVisible moves selection to the previous visible profile.
func (p *ProfilesPanel) MoveUpVisible() {
	for i := p.Selected - 1; i >= 0; i-- {
		if !p.isHidden(i) {
			p.Selected = i
			return
		}
	}
}

// MoveDownVisible moves selection to the next visible profile.
func (p *ProfilesPanel) MoveDownVisible() {
	for i := p.Selected + 1; i < len(p.Profiles); i++ {
		if !p.isHidden(i) {
			p.Selected = i
			return
		}
	}
}

// latencyIndicator returns a colored dot based on latency value.
func latencyIndicator(latency int64) string {
	green := lipgloss.NewStyle().Foreground(theme.Current().Success)
	yellow := lipgloss.NewStyle().Foreground(theme.Current().Selected)
	red := lipgloss.NewStyle().Foreground(theme.Current().Error)
	grey := lipgloss.NewStyle().Foreground(theme.Current().Muted)

	switch {
	case latency == 0 || latency == -2:
		return grey.Render("○")
	case latency < 0:
		return red.Render("●")
	case latency < 100:
		return green.Render("●")
	case latency < 300:
		return yellow.Render("●")
	default:
		return red.Render("●")
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
