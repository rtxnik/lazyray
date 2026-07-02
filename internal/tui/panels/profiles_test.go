package panels

import (
	"strings"
	"testing"

	"github.com/rtxnik/lazyray/internal/config"
)

func TestProfilesPanel_SetProfiles(t *testing.T) {
	p := NewProfilesPanel()
	profiles := []config.Profile{
		{Name: "server-1"},
		{Name: "server-2"},
		{Name: "server-3"},
	}

	p.SetProfiles(profiles)
	if len(p.Profiles) != 3 {
		t.Errorf("Profiles count = %d, want 3", len(p.Profiles))
	}
}

func TestProfilesPanel_SetProfiles_AdjustsSelected(t *testing.T) {
	p := NewProfilesPanel()
	p.Selected = 5

	profiles := []config.Profile{
		{Name: "server-1"},
		{Name: "server-2"},
	}
	p.SetProfiles(profiles)

	if p.Selected != 1 {
		t.Errorf("Selected = %d, should be adjusted to 1 (last index)", p.Selected)
	}
}

func TestProfilesPanel_MoveUp(t *testing.T) {
	p := NewProfilesPanel()
	p.SetProfiles([]config.Profile{{Name: "a"}, {Name: "b"}, {Name: "c"}})
	p.Selected = 2

	p.MoveUp()
	if p.Selected != 1 {
		t.Errorf("after MoveUp: Selected = %d, want 1", p.Selected)
	}

	p.MoveUp()
	if p.Selected != 0 {
		t.Errorf("after MoveUp: Selected = %d, want 0", p.Selected)
	}

	// Should not go below 0
	p.MoveUp()
	if p.Selected != 0 {
		t.Errorf("at top, MoveUp: Selected = %d, want 0", p.Selected)
	}
}

func TestProfilesPanel_MoveDown(t *testing.T) {
	p := NewProfilesPanel()
	p.SetProfiles([]config.Profile{{Name: "a"}, {Name: "b"}, {Name: "c"}})
	p.Selected = 0

	p.MoveDown()
	if p.Selected != 1 {
		t.Errorf("after MoveDown: Selected = %d, want 1", p.Selected)
	}

	p.MoveDown()
	if p.Selected != 2 {
		t.Errorf("after MoveDown: Selected = %d, want 2", p.Selected)
	}

	// Should not exceed last index
	p.MoveDown()
	if p.Selected != 2 {
		t.Errorf("at bottom, MoveDown: Selected = %d, want 2", p.Selected)
	}
}

func TestProfilesPanel_SelectedProfile(t *testing.T) {
	p := NewProfilesPanel()
	p.SetProfiles([]config.Profile{{Name: "a"}, {Name: "b"}})
	p.Selected = 1

	profile := p.SelectedProfile()
	if profile == nil {
		t.Fatal("SelectedProfile() returned nil")
	}
	if profile.Name != "b" {
		t.Errorf("SelectedProfile().Name = %q, want %q", profile.Name, "b")
	}
}

func TestProfilesPanel_SelectedProfile_Empty(t *testing.T) {
	p := NewProfilesPanel()
	profile := p.SelectedProfile()
	if profile != nil {
		t.Error("SelectedProfile() should return nil when no profiles")
	}
}

func TestProfilesPanel_ViewEmpty(t *testing.T) {
	p := NewProfilesPanel()
	p.Width = 30
	view := p.View()

	if !strings.Contains(view, "No profiles") {
		t.Error("empty view should show 'No profiles' message")
	}
}

func TestProfilesPanel_ViewShowsProfiles(t *testing.T) {
	p := NewProfilesPanel()
	p.Width = 30
	p.SetProfiles([]config.Profile{
		{Name: "server-1", Default: true},
		{Name: "server-2"},
	})

	view := p.View()
	if !strings.Contains(view, "server-1") {
		t.Error("view should contain first profile name")
	}
	if !strings.Contains(view, "server-2") {
		t.Error("view should contain second profile name")
	}
}

func TestProfilesPanel_ViewDefaultMarker(t *testing.T) {
	p := NewProfilesPanel()
	p.Width = 30
	p.Selected = 1 // select second profile, first is default
	p.SetProfiles([]config.Profile{
		{Name: "default-server", Default: true},
		{Name: "other-server"},
	})

	view := p.View()
	// Default profile should have * marker
	if !strings.Contains(view, "*") {
		t.Error("default profile should have * marker")
	}
}

func TestProfilesPanel_ViewSelectedHighlight(t *testing.T) {
	p := NewProfilesPanel()
	p.Width = 30
	p.SetProfiles([]config.Profile{
		{Name: "server-1"},
		{Name: "server-2"},
	})
	p.Selected = 0

	view := p.View()
	// Selected profile should have > prefix
	if !strings.Contains(view, ">") {
		t.Error("selected profile should have > prefix")
	}
}

func TestProfilesBadge_SkippedIsNeutralNotFail(t *testing.T) {
	p := ProfilesPanel{Width: 40}
	p.SetProfiles([]config.Profile{{Name: "hy2", Latency: -2}})
	view := p.View()
	if !strings.Contains(view, "n/a") {
		t.Errorf("skipped profile must show 'n/a', got:\n%s", view)
	}
	if strings.Contains(view, "FAIL") {
		t.Errorf("skipped profile must NOT show FAIL, got:\n%s", view)
	}
}

func TestProfilesBadge_FailStillShown(t *testing.T) {
	p := ProfilesPanel{Width: 40}
	p.SetProfiles([]config.Profile{{Name: "dead", Latency: -1}})
	if !strings.Contains(p.View(), "FAIL") {
		t.Errorf("failed profile must still show FAIL")
	}
}
