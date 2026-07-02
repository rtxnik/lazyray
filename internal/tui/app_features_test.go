package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/rtxnik/lazyray/internal/config"
	"github.com/rtxnik/lazyray/internal/tui/notify"
)

func newTestApp() *App {
	app := NewApp("test")
	app.width = 120
	app.height = 40
	app.updateLayoutQuick()
	app.servers = &config.ServersConfig{
		Profiles: []config.Profile{
			{Name: "Profile A", Default: true, Group: "group1",
				Server: config.ServerConfig{Address: "1.2.3.4", Port: 443, UUID: "uuid-a",
					Transport: config.TransportConfig{Network: "tcp"}, Security: config.SecurityConfig{Type: "none"}}},
			{Name: "Profile B", Group: "group1",
				Server: config.ServerConfig{Address: "5.6.7.8", Port: 443, UUID: "uuid-b",
					Transport: config.TransportConfig{Network: "tcp"}, Security: config.SecurityConfig{Type: "none"}}},
			{Name: "Profile C", Group: "group2",
				Server: config.ServerConfig{Address: "9.0.1.2", Port: 443, UUID: "uuid-c",
					Transport: config.TransportConfig{Network: "tcp"}, Security: config.SecurityConfig{Type: "none"}}},
		},
	}
	app.profiles.SetProfiles(app.servers.Profiles)
	app.refreshGroups()
	return app
}

func TestApp_RefreshGroups(t *testing.T) {
	app := newTestApp()

	if len(app.allGroups) != 2 {
		t.Fatalf("allGroups count = %d, want 2", len(app.allGroups))
	}
	if app.allGroups[0] != "group1" {
		t.Errorf("allGroups[0] = %q, want %q", app.allGroups[0], "group1")
	}
	if app.allGroups[1] != "group2" {
		t.Errorf("allGroups[1] = %q, want %q", app.allGroups[1], "group2")
	}
}

func TestApp_CycleGroupFilter(t *testing.T) {
	app := newTestApp()

	// Start: all profiles
	if app.groupFilter != "" {
		t.Fatalf("initial groupFilter = %q, want empty", app.groupFilter)
	}

	// Cycle to group1
	app.cycleGroupFilter()
	if app.groupFilter != "group1" {
		t.Errorf("after 1st cycle: groupFilter = %q, want group1", app.groupFilter)
	}
	if len(app.profiles.Profiles) != 2 {
		t.Errorf("group1 filter: %d profiles, want 2", len(app.profiles.Profiles))
	}

	// Cycle to group2
	app.cycleGroupFilter()
	if app.groupFilter != "group2" {
		t.Errorf("after 2nd cycle: groupFilter = %q, want group2", app.groupFilter)
	}
	if len(app.profiles.Profiles) != 1 {
		t.Errorf("group2 filter: %d profiles, want 1", len(app.profiles.Profiles))
	}

	// Cycle back to all
	app.cycleGroupFilter()
	if app.groupFilter != "" {
		t.Errorf("after 3rd cycle: groupFilter = %q, want empty", app.groupFilter)
	}
	if len(app.profiles.Profiles) != 3 {
		t.Errorf("all filter: %d profiles, want 3", len(app.profiles.Profiles))
	}
}

func TestApp_GroupFilterLabel(t *testing.T) {
	app := newTestApp()

	label := app.groupFilterLabel()
	if label != "filter: all profiles" {
		t.Errorf("label = %q, want 'filter: all profiles'", label)
	}

	app.groupFilter = "group1"
	label = app.groupFilterLabel()
	if label != "filter: group1" {
		t.Errorf("label = %q, want 'filter: group1'", label)
	}
}

func TestApp_ApplyGroupFilter_NoGroups(t *testing.T) {
	app := NewApp("test")
	app.width = 120
	app.height = 40
	app.updateLayoutQuick()
	app.servers = &config.ServersConfig{
		Profiles: []config.Profile{
			{Name: "P1", Server: config.ServerConfig{Address: "1.2.3.4", Port: 443}},
		},
	}
	app.profiles.SetProfiles(app.servers.Profiles)

	// No groups exist, cycling should do nothing
	app.cycleGroupFilter()
	if app.groupFilter != "" {
		t.Errorf("groupFilter = %q, want empty when no groups", app.groupFilter)
	}
}

func TestApp_TestAllResultMsg(t *testing.T) {
	app := newTestApp()
	app.loading = true

	results := []profileTestResult{
		{name: "Profile A", latency: 50000000},  // 50ms
		{name: "Profile B", latency: 100000000}, // 100ms
		{name: "Profile C", err: errTest},
	}

	model, _ := app.Update(testAllResultMsg{results: results})
	a := model.(*App)

	if a.loading {
		t.Error("loading should be false after testAllResultMsg")
	}

	// Check latency was saved to profiles
	if a.servers.Profiles[0].Latency != 50 {
		t.Errorf("Profile A latency = %d, want 50", a.servers.Profiles[0].Latency)
	}
	if a.servers.Profiles[1].Latency != 100 {
		t.Errorf("Profile B latency = %d, want 100", a.servers.Profiles[1].Latency)
	}
	if a.servers.Profiles[2].Latency != -1 {
		t.Errorf("Profile C latency = %d, want -1 (failed)", a.servers.Profiles[2].Latency)
	}
}

func TestApp_TestAllResultMsg_Error(t *testing.T) {
	app := newTestApp()
	app.loading = true

	model, _ := app.Update(testAllResultMsg{err: errTest})
	a := model.(*App)

	if a.loading {
		t.Error("loading should be false after error")
	}
	if a.activeTail == nil || a.activeTail.Severity != notify.Error {
		t.Error("activeTail should be an Error notice")
	}
}

func TestApp_SubscriptionResultMsg(t *testing.T) {
	app := newTestApp()
	app.loading = true

	model, _ := app.Update(subscriptionResultMsg{added: 3, updated: 1})
	a := model.(*App)

	if a.loading {
		t.Error("loading should be false after subscriptionResultMsg")
	}
	if a.activeTail == nil || a.activeTail.Severity != notify.Success {
		t.Error("activeTail should be a Success notice")
	}
}

func TestApp_SubscriptionResultMsg_Error(t *testing.T) {
	app := newTestApp()
	app.loading = true

	model, _ := app.Update(subscriptionResultMsg{err: errTest})
	a := model.(*App)

	if a.loading {
		t.Error("loading should be false after error")
	}
	if a.activeTail == nil || a.activeTail.Severity != notify.Error {
		t.Error("activeTail should be an Error notice")
	}
}

func TestApp_View_Renders(t *testing.T) {
	app := newTestApp()
	view := app.View()

	if view == "" {
		t.Error("View() should not return empty string")
	}
}

func TestApp_View_ZeroSize(t *testing.T) {
	app := NewApp("test")
	view := app.View()

	if view != "Loading..." {
		t.Errorf("View() with zero size should return 'Loading...', got %q", view)
	}
}

func TestApp_StatusBar_Loading(t *testing.T) {
	app := newTestApp()
	app.loading = true
	app.loadingMsg = "Testing..."

	bar := app.renderStatusBar()
	if bar == "" {
		t.Error("status bar should not be empty during loading")
	}
}

func TestApp_StatusBar_Error(t *testing.T) {
	app := newTestApp()
	app.activeTail = &notify.Notice{Severity: notify.Error, Message: "something went wrong"}

	bar := app.renderStatusBar()
	if bar == "" {
		t.Error("status bar should not be empty with error")
	}
	if !strings.Contains(bar, "ERROR") || !strings.Contains(bar, "something went wrong") {
		t.Errorf("error bar missing tag or message:\n%s", bar)
	}
}

func TestApp_StatusBar_Message(t *testing.T) {
	app := newTestApp()
	app.activeTail = &notify.Notice{Severity: notify.Success, Message: "profile imported"}

	bar := app.renderStatusBar()
	if bar == "" {
		t.Error("status bar should not be empty with message")
	}
	if !strings.Contains(bar, "profile imported") {
		t.Errorf("success bar missing message:\n%s", bar)
	}
}

func TestApp_StatusBar_UpdateAvailable(t *testing.T) {
	app := newTestApp()
	app.availableUpdate = "v1.9.0"

	bar := app.renderStatusBar()
	if bar == "" {
		t.Error("status bar should not be empty")
	}
}

func TestApp_HotkeysBar(t *testing.T) {
	app := newTestApp()
	bar := app.renderHotkeysBar()

	if bar == "" {
		t.Error("hotkeys bar should not be empty")
	}
}

func TestApp_CalcLayout(t *testing.T) {
	app := newTestApp()
	leftW, rightW, fullH, statusH, logsH := app.calcLayout()

	if leftW <= 0 {
		t.Errorf("leftWidth = %d, want > 0", leftW)
	}
	if rightW <= 0 {
		t.Errorf("rightWidth = %d, want > 0", rightW)
	}
	if fullH <= 0 {
		t.Errorf("fullHeight = %d, want > 0", fullH)
	}
	if statusH <= 0 {
		t.Errorf("statusHeight = %d, want > 0", statusH)
	}
	if logsH <= 0 {
		t.Errorf("logsHeight = %d, want > 0", logsH)
	}
}

func TestApp_CalcLayout_SmallTerminal(t *testing.T) {
	app := NewApp("test")
	app.width = 30
	app.height = 10

	leftW, rightW, fullH, statusH, logsH := app.calcLayout()

	// Narrow terminal uses stacked single-column layout
	if leftW < 10 {
		t.Errorf("leftWidth = %d, want >= 10 (minimum)", leftW)
	}
	if rightW < 10 {
		t.Errorf("rightWidth = %d, want >= 10 (minimum)", rightW)
	}
	if fullH < 4 {
		t.Errorf("fullHeight = %d, want >= 4 (minimum)", fullH)
	}
	if statusH < 3 {
		t.Errorf("statusHeight = %d, want >= 3 (minimum)", statusH)
	}
	if logsH < 3 {
		t.Errorf("logsHeight = %d, want >= 3 (minimum)", logsH)
	}
}

func TestApp_OverlayModal(t *testing.T) {
	app := newTestApp()
	bg := "line1\nline2\nline3\nline4\nline5"
	modal := "M"

	result := app.overlayModal(bg, modal)
	if result == "" {
		t.Error("overlayModal should not return empty string")
	}
}

func TestApp_ShiftTabNavigation(t *testing.T) {
	app := newTestApp()
	if app.activePanel != PanelProfiles {
		t.Fatal("should start at PanelProfiles")
	}

	// Shift+Tab should go backwards (wrap to Logs)
	shiftTabMsg := tea.KeyMsg{Type: tea.KeyShiftTab}
	model, _ := app.Update(shiftTabMsg)
	a := model.(*App)
	if a.activePanel != PanelLogs {
		t.Errorf("after Shift+Tab from Profiles: panel = %d, want PanelLogs", a.activePanel)
	}
}

func TestApp_UpDownNavigation(t *testing.T) {
	app := newTestApp()
	app.activePanel = PanelProfiles

	downMsg := tea.KeyMsg{Type: tea.KeyDown}
	model, _ := app.Update(downMsg)
	a := model.(*App)
	if a.profiles.Selected != 1 {
		t.Errorf("after Down: Selected = %d, want 1", a.profiles.Selected)
	}

	upMsg := tea.KeyMsg{Type: tea.KeyUp}
	model, _ = a.Update(upMsg)
	a = model.(*App)
	if a.profiles.Selected != 0 {
		t.Errorf("after Up: Selected = %d, want 0", a.profiles.Selected)
	}
}

func TestRenderHotkeysBarFromRegistry(t *testing.T) {
	app := newTestApp()
	app.width = 200 // wide
	app.updateLayoutQuick()
	bar := app.renderHotkeysBar()
	for _, want := range []string{"start", "restart", "doctor", "import", "subs", "test", "dup", "group", "routing", "help", "quit"} {
		if !strings.Contains(bar, want) {
			t.Errorf("wide bar missing %q", want)
		}
	}
	if strings.Contains(bar, "tunnel") {
		t.Error("wide bar should not contain non-bar command 'tunnel'")
	}
}
