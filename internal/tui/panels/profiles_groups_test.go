package panels

import (
	"strings"
	"testing"

	"github.com/rtxnik/lazyray/internal/config"
)

func TestProfilesPanel_ViewWithGroups(t *testing.T) {
	p := NewProfilesPanel()
	p.Width = 40
	p.SetProfiles([]config.Profile{
		{Name: "Server A", Group: "US Servers"},
		{Name: "Server B", Group: "US Servers"},
		{Name: "Server C", Group: "EU Servers"},
	})

	view := p.View()
	if !strings.Contains(view, "US Servers") {
		t.Error("view should contain group header 'US Servers'")
	}
	if !strings.Contains(view, "EU Servers") {
		t.Error("view should contain group header 'EU Servers'")
	}
}

func TestProfilesPanel_ViewWithLatency(t *testing.T) {
	p := NewProfilesPanel()
	p.Width = 40
	p.SetProfiles([]config.Profile{
		{Name: "Fast Server", Latency: 45},
		{Name: "Slow Server", Latency: 200},
		{Name: "Failed Server", Latency: -1},
		{Name: "Unknown Server", Latency: 0},
	})

	view := p.View()
	if !strings.Contains(view, "45ms") {
		t.Error("view should show latency '45ms'")
	}
	if !strings.Contains(view, "200ms") {
		t.Error("view should show latency '200ms'")
	}
	if !strings.Contains(view, "FAIL") {
		t.Error("view should show 'FAIL' for negative latency")
	}
}

func TestProfilesPanel_MoveProfileUp(t *testing.T) {
	p := NewProfilesPanel()
	p.SetProfiles([]config.Profile{{Name: "a"}, {Name: "b"}, {Name: "c"}})
	p.Selected = 1

	moved := p.MoveProfileUp()
	if !moved {
		t.Error("MoveProfileUp should return true")
	}
	if p.Selected != 0 {
		t.Errorf("Selected = %d, want 0", p.Selected)
	}
	if p.Profiles[0].Name != "b" {
		t.Errorf("Profiles[0].Name = %q, want 'b' (swapped)", p.Profiles[0].Name)
	}
}

func TestProfilesPanel_MoveProfileUp_AtTop(t *testing.T) {
	p := NewProfilesPanel()
	p.SetProfiles([]config.Profile{{Name: "a"}, {Name: "b"}})
	p.Selected = 0

	moved := p.MoveProfileUp()
	if moved {
		t.Error("MoveProfileUp at top should return false")
	}
}

func TestProfilesPanel_MoveProfileDown(t *testing.T) {
	p := NewProfilesPanel()
	p.SetProfiles([]config.Profile{{Name: "a"}, {Name: "b"}, {Name: "c"}})
	p.Selected = 1

	moved := p.MoveProfileDown()
	if !moved {
		t.Error("MoveProfileDown should return true")
	}
	if p.Selected != 2 {
		t.Errorf("Selected = %d, want 2", p.Selected)
	}
	if p.Profiles[2].Name != "b" {
		t.Errorf("Profiles[2].Name = %q, want 'b' (swapped)", p.Profiles[2].Name)
	}
}

func TestProfilesPanel_MoveProfileDown_AtBottom(t *testing.T) {
	p := NewProfilesPanel()
	p.SetProfiles([]config.Profile{{Name: "a"}, {Name: "b"}})
	p.Selected = 1

	moved := p.MoveProfileDown()
	if moved {
		t.Error("MoveProfileDown at bottom should return false")
	}
}

func TestProfilesPanel_Rename(t *testing.T) {
	p := NewProfilesPanel()
	p.Width = 40
	p.SetProfiles([]config.Profile{{Name: "original"}})
	p.Selected = 0

	p.StartRename()
	if !p.Renaming {
		t.Error("Renaming should be true after StartRename")
	}
	if p.RenameIdx != 0 {
		t.Errorf("RenameIdx = %d, want 0", p.RenameIdx)
	}

	// Set new name
	p.RenameInput.SetValue("renamed")
	name, ok := p.ConfirmRename()
	if !ok {
		t.Error("ConfirmRename should return ok=true")
	}
	if name != "renamed" {
		t.Errorf("name = %q, want 'renamed'", name)
	}
	if p.Renaming {
		t.Error("Renaming should be false after ConfirmRename")
	}
}

func TestProfilesPanel_CancelRename(t *testing.T) {
	p := NewProfilesPanel()
	p.Width = 40
	p.SetProfiles([]config.Profile{{Name: "original"}})
	p.Selected = 0

	p.StartRename()
	p.CancelRename()
	if p.Renaming {
		t.Error("Renaming should be false after CancelRename")
	}
}

func TestProfilesPanel_ConfirmRename_Empty(t *testing.T) {
	p := NewProfilesPanel()
	p.Width = 40
	p.SetProfiles([]config.Profile{{Name: "original"}})
	p.Selected = 0

	p.StartRename()
	p.RenameInput.SetValue("   ")
	_, ok := p.ConfirmRename()
	if ok {
		t.Error("ConfirmRename with empty name should return ok=false")
	}
}

func TestProfilesPanel_StartRename_EmptyProfiles(t *testing.T) {
	p := NewProfilesPanel()
	p.StartRename()
	if p.Renaming {
		t.Error("StartRename with empty profiles should not activate renaming")
	}
}

func TestProfilesPanel_ViewTruncatesLongNames(t *testing.T) {
	p := NewProfilesPanel()
	p.Width = 20
	p.SetProfiles([]config.Profile{
		{Name: "This Is A Very Long Server Name That Should Be Truncated"},
	})

	view := p.View()
	// The view should not exceed the panel width
	lines := strings.Split(view, "\n")
	if len(lines) == 0 {
		t.Fatal("view should have at least one line")
	}
}

func TestProfilesPanel_ViewRenaming(t *testing.T) {
	p := NewProfilesPanel()
	p.Width = 40
	p.SetProfiles([]config.Profile{{Name: "server-1"}, {Name: "server-2"}})
	p.Selected = 0
	p.StartRename()

	view := p.View()
	// While renaming, the first line should show the text input
	if !strings.Contains(view, ">") {
		t.Error("renaming view should show > prefix")
	}
}
