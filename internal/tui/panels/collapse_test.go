package panels

import (
	"strings"
	"testing"

	"github.com/rtxnik/lazyray/internal/config"
)

func TestProfilesPanel_ToggleGroupCollapse(t *testing.T) {
	p := NewProfilesPanel()
	p.SetProfiles([]config.Profile{
		{Name: "Server A", Group: "US"},
		{Name: "Server B", Group: "US"},
		{Name: "Server C", Group: "EU"},
	})
	p.Selected = 0

	p.ToggleGroupCollapse()
	if !p.CollapsedGroups["US"] {
		t.Error("US group should be collapsed after toggle")
	}

	p.ToggleGroupCollapse()
	if p.CollapsedGroups["US"] {
		t.Error("US group should be expanded after second toggle")
	}
}

func TestProfilesPanel_ToggleGroupCollapse_NoGroup(t *testing.T) {
	p := NewProfilesPanel()
	p.SetProfiles([]config.Profile{
		{Name: "Server A"},
	})
	p.Selected = 0

	// Should not panic for profile without group
	p.ToggleGroupCollapse()
}

func TestProfilesPanel_ToggleGroupCollapse_EmptyProfiles(t *testing.T) {
	p := NewProfilesPanel()
	p.ToggleGroupCollapse() // Should not panic
}

func TestProfilesPanel_IsCollapsed(t *testing.T) {
	p := NewProfilesPanel()
	p.SetProfiles([]config.Profile{
		{Name: "Server A", Group: "US"},
		{Name: "Server B", Group: "US"},
		{Name: "Server C", Group: "EU"},
	})

	p.CollapsedGroups["US"] = true

	// First profile of US group is NOT collapsed (it's the header)
	if p.isCollapsed(0) {
		t.Error("first profile of collapsed group should not be collapsed")
	}

	// Second profile of US group IS collapsed
	if !p.isCollapsed(1) {
		t.Error("second profile of collapsed US group should be collapsed")
	}

	// EU group is not collapsed
	if p.isCollapsed(2) {
		t.Error("EU group profile should not be collapsed")
	}
}

func TestProfilesPanel_ViewCollapsed(t *testing.T) {
	p := NewProfilesPanel()
	p.Width = 40
	p.SetProfiles([]config.Profile{
		{Name: "Server A", Group: "US"},
		{Name: "Server B", Group: "US"},
		{Name: "Server C", Group: "EU"},
	})

	p.CollapsedGroups["US"] = true

	view := p.View()

	// Should show Server A (first in US group) but not Server B
	if !strings.Contains(view, "Server A") {
		t.Error("collapsed view should show first profile of US group")
	}
	if strings.Contains(view, "Server B") {
		t.Error("collapsed view should hide Server B (second in collapsed US group)")
	}
	if !strings.Contains(view, "Server C") {
		t.Error("collapsed view should show Server C (EU group not collapsed)")
	}
}

func TestProfilesPanel_ViewCollapsedArrow(t *testing.T) {
	p := NewProfilesPanel()
	p.Width = 40
	p.SetProfiles([]config.Profile{
		{Name: "A", Group: "US"},
		{Name: "B", Group: "US"},
	})

	// Expanded: should show down arrow
	view := p.View()
	if !strings.Contains(view, "▼") {
		t.Error("expanded group should show ▼ arrow")
	}

	// Collapsed: should show right arrow
	p.CollapsedGroups["US"] = true
	view = p.View()
	if !strings.Contains(view, "▶") {
		t.Error("collapsed group should show ▶ arrow")
	}
}

func TestProfilesPanel_ViewGroupCount(t *testing.T) {
	p := NewProfilesPanel()
	p.Width = 40
	p.SetProfiles([]config.Profile{
		{Name: "A", Group: "US"},
		{Name: "B", Group: "US"},
		{Name: "C", Group: "US"},
	})

	view := p.View()
	if !strings.Contains(view, "(3)") {
		t.Error("group header should show count (3)")
	}
}

func TestMax(t *testing.T) {
	if max(3, 5) != 5 {
		t.Error("max(3,5) should be 5")
	}
	if max(5, 3) != 5 {
		t.Error("max(5,3) should be 5")
	}
	if max(4, 4) != 4 {
		t.Error("max(4,4) should be 4")
	}
}
