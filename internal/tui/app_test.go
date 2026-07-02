package tui

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/rtxnik/lazyray/internal/core"
	"github.com/rtxnik/lazyray/internal/tui/commands"
	"github.com/rtxnik/lazyray/internal/tui/modals"
	"github.com/rtxnik/lazyray/internal/tui/notify"
	"github.com/rtxnik/lazyray/internal/tui/panels"
)

func TestApp_TrafficResultMsg_SetsTraffic(t *testing.T) {
	app := NewApp("test")
	app.width = 120
	app.height = 40
	app.updateLayoutQuick()

	stats := &core.TrafficStats{Uplink: 1024, Downlink: 2048}
	msg := trafficResultMsg{stats: stats}

	model, _ := app.Update(msg)
	a := model.(*App)

	if a.status.Traffic == nil {
		t.Fatal("Traffic should not be nil after trafficResultMsg")
	}
	if a.status.Traffic.Uplink != 1024 {
		t.Errorf("Uplink = %d, want 1024", a.status.Traffic.Uplink)
	}
	if a.status.Traffic.Downlink != 2048 {
		t.Errorf("Downlink = %d, want 2048", a.status.Traffic.Downlink)
	}
}

func TestApp_TrafficResultMsg_SpeedCalculation(t *testing.T) {
	app := NewApp("test")
	app.width = 120
	app.height = 40
	app.updateLayoutQuick()

	// Tick 1: set initial traffic
	stats1 := &core.TrafficStats{Uplink: 1000, Downlink: 2000}
	app.Update(trafficResultMsg{stats: stats1})

	// Simulate time passing
	app.status.PrevTrafficAt = time.Now().Unix() - 5

	// Tick 2: update traffic — speed should be calculated
	stats2 := &core.TrafficStats{Uplink: 6000, Downlink: 12000}
	model, _ := app.Update(trafficResultMsg{stats: stats2})
	a := model.(*App)

	if a.status.UpSpeed == "" {
		t.Error("UpSpeed should be set after 2nd traffic tick")
	}
	if a.status.DnSpeed == "" {
		t.Error("DnSpeed should be set after 2nd traffic tick")
	}
}

func TestApp_TrafficResultMsg_NilStats(t *testing.T) {
	app := NewApp("test")
	app.width = 120
	app.height = 40
	app.updateLayoutQuick()

	// Set some traffic first
	app.status.Traffic = &core.TrafficStats{Uplink: 100, Downlink: 200}

	// Send nil stats (xray stopped)
	model, _ := app.Update(trafficResultMsg{stats: nil})
	a := model.(*App)

	if a.status.Traffic != nil {
		t.Error("Traffic should be nil when nil stats received")
	}
}

func TestApp_ExitIPResultMsg_Success(t *testing.T) {
	app := NewApp("test")
	app.width = 120
	app.height = 40
	app.updateLayoutQuick()
	app.fetchingExitIP = true

	msg := exitIPResultMsg{ip: "1.2.3.4", latency: 150 * time.Millisecond}
	model, _ := app.Update(msg)
	a := model.(*App)

	if a.status.ExitIP != "1.2.3.4" {
		t.Errorf("ExitIP = %q, want %q", a.status.ExitIP, "1.2.3.4")
	}
	if a.status.Latency != "150ms" {
		t.Errorf("Latency = %q, want %q", a.status.Latency, "150ms")
	}
	if a.fetchingExitIP {
		t.Error("fetchingExitIP should be false after result")
	}
}

func TestApp_ExitIPResultMsg_Error(t *testing.T) {
	app := NewApp("test")
	app.width = 120
	app.height = 40
	app.updateLayoutQuick()
	app.fetchingExitIP = true
	app.status.ExitIP = "old-ip"

	msg := exitIPResultMsg{err: errTest}
	model, _ := app.Update(msg)
	a := model.(*App)

	// On error, IP should remain unchanged
	if a.status.ExitIP != "old-ip" {
		t.Errorf("ExitIP = %q, should remain 'old-ip' on error", a.status.ExitIP)
	}
	if a.fetchingExitIP {
		t.Error("fetchingExitIP should be false even on error")
	}
}

func TestApp_ClearTailMsg(t *testing.T) {
	app := NewApp("test")
	app.width = 120
	app.height = 40
	app.updateLayoutQuick()

	app.activeTail = &notify.Notice{ID: 1, Severity: notify.Error, Message: "some error"}

	model, _ := app.Update(clearTailMsg{id: 1})
	a := model.(*App)

	if a.activeTail != nil {
		t.Errorf("activeTail = %v, want nil after matching clearTailMsg", a.activeTail)
	}
}

func TestApp_WindowSizeMsg(t *testing.T) {
	app := NewApp("test")
	msg := tea.WindowSizeMsg{Width: 100, Height: 50}

	model, _ := app.Update(msg)
	a := model.(*App)

	if a.width != 100 {
		t.Errorf("width = %d, want 100", a.width)
	}
	if a.height != 50 {
		t.Errorf("height = %d, want 50", a.height)
	}
}

func TestApp_TabNavigation(t *testing.T) {
	app := NewApp("test")
	app.width = 120
	app.height = 40
	app.updateLayoutQuick()

	if app.activePanel != PanelProfiles {
		t.Fatalf("initial panel = %d, want PanelProfiles", app.activePanel)
	}

	// Tab forward
	tabMsg := tea.KeyMsg{Type: tea.KeyTab}
	model, _ := app.Update(tabMsg)
	a := model.(*App)
	if a.activePanel != PanelStatus {
		t.Errorf("after Tab: panel = %d, want PanelStatus", a.activePanel)
	}

	// Tab again
	model, _ = a.Update(tabMsg)
	a = model.(*App)
	if a.activePanel != PanelLogs {
		t.Errorf("after Tab+Tab: panel = %d, want PanelLogs", a.activePanel)
	}

	// Tab wraps around
	model, _ = a.Update(tabMsg)
	a = model.(*App)
	if a.activePanel != PanelProfiles {
		t.Errorf("after Tab+Tab+Tab: panel = %d, want PanelProfiles (wrap)", a.activePanel)
	}
}

var errTest = &testError{}

type testError struct{}

func (e *testError) Error() string { return "test error" }

func TestActivityKeyOpensOverlay(t *testing.T) {
	app := NewApp("test")
	app.width, app.height = 80, 24
	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	a := model.(*App)
	if a.modalID != ModalActivity {
		t.Fatalf("modalID = %d, want ModalActivity", a.modalID)
	}
}

func TestUnreadCountAndReset(t *testing.T) {
	app := NewApp("test")
	app.width, app.height = 80, 24
	app.notify(notify.Notice{Severity: notify.Info, Message: "a"})
	app.notify(notify.Notice{Severity: notify.Info, Message: "b"})
	if got := app.unreadCount(); got != 2 {
		t.Fatalf("unreadCount = %d, want 2", got)
	}
	app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}}) // opens Activity, marks read
	if got := app.unreadCount(); got != 0 {
		t.Fatalf("unreadCount after open = %d, want 0", got)
	}
}

func findLaunchable(t *testing.T, id string) commands.Command {
	t.Helper()
	for _, c := range commands.New(commands.DefaultKeyMap()).Launchable() {
		if c.ID == id {
			return c
		}
	}
	t.Fatalf("launchable command %q not found", id)
	return commands.Command{}
}

func TestPaletteKeyOpensOverlay(t *testing.T) {
	app := NewApp("test")
	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{':'}})
	if model.(*App).modalID != ModalPalette {
		t.Fatalf("modalID = %v, want ModalPalette", model.(*App).modalID)
	}
}

func TestPaletteFuzzyLaunchTriggersAction(t *testing.T) {
	app := NewApp("test")
	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{':'}})
	a := model.(*App)
	for _, r := range "diag" {
		model, _ = a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		a = model.(*App)
	}
	model, _ = a.Update(tea.KeyMsg{Type: tea.KeyEnter})
	a = model.(*App)
	if a.modalID != ModalDoctor {
		t.Fatalf("modalID = %v, want ModalDoctor after launching diagnostics from palette", a.modalID)
	}
	found := false
	for _, e := range a.notices.Entries() {
		if e.Source == "palette" {
			found = true
		}
	}
	if !found {
		t.Error("expected a palette-sourced launch notice in the log")
	}
}

// Guard G5 (scope auto-focus on palette launch). Documented in
// docs/ARCHITECTURE.md (Invariants & Guards). Keep that section and this test in sync.
func TestLaunchCommandAutoFocusesScopePanel(t *testing.T) {
	app := NewApp("test")
	app.activePanel = PanelLogs
	app.launchCommand(findLaunchable(t, "Duplicate")) // ScopeProfiles
	if app.activePanel != PanelProfiles {
		t.Errorf("activePanel = %v, want PanelProfiles after launching a ScopeProfiles command", app.activePanel)
	}
}

func TestLaunchCommandEmitsInfoNotice(t *testing.T) {
	app := NewApp("test")
	app.launchCommand(findLaunchable(t, "Doctor"))
	entries := app.notices.Entries()
	if len(entries) == 0 {
		t.Fatal("expected a launch notice")
	}
	top := entries[0] // newest first
	if top.Severity != notify.Info || top.Source != "palette" {
		t.Errorf("notice = {sev:%v src:%q}, want {Info palette}", top.Severity, top.Source)
	}
	// Derive the expected message from the registry so the assertion tracks the
	// canonical title rather than a hardcoded copy that drifts silently.
	want := "→ " + findLaunchable(t, "Doctor").Title
	if top.Message != want {
		t.Errorf("message = %q, want %q", top.Message, want)
	}
}

// TestPaletteEscDispatchesNothing guards the result-arm's `m.Selected != nil`
// gate: closing the palette with Esc must run no command and leave no notice.
func TestPaletteEscDispatchesNothing(t *testing.T) {
	app := NewApp("test")
	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{':'}})
	a := model.(*App)
	if a.modalID != ModalPalette {
		t.Fatalf("palette did not open: modalID = %v", a.modalID)
	}
	model, _ = a.Update(tea.KeyMsg{Type: tea.KeyEsc})
	a = model.(*App)
	if a.modalID != ModalNone {
		t.Errorf("Esc should close the palette to ModalNone, got %v", a.modalID)
	}
	if n := len(a.notices.Entries()); n != 0 {
		t.Errorf("Esc must dispatch nothing; got %d notice(s)", n)
	}
}

func TestApp_ToggleMetricKey_ScopedToStatus(t *testing.T) {
	app := NewApp("test")
	app.width = 120
	app.height = 40
	app.updateLayoutQuick()

	// Profiles focused: "m" is a no-op for the dashboard.
	app.activePanel = PanelProfiles
	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("m")})
	app = model.(*App)
	if app.status.Metric != panels.MetricSpeed {
		t.Errorf("m on Profiles toggled the metric; want no-op")
	}

	// Status focused: "m" toggles to latency.
	app.activePanel = PanelStatus
	model, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("m")})
	app = model.(*App)
	if app.status.Metric != panels.MetricLatency {
		t.Errorf("m on Status did not toggle to latency")
	}
}

func TestApp_ExitIPResultMsg_FeedsLatencyRing(t *testing.T) {
	app := NewApp("test")
	app.width = 120
	app.height = 40
	app.updateLayoutQuick()

	model, _ := app.Update(exitIPResultMsg{ip: "1.2.3.4", latency: 42 * time.Millisecond})
	a := model.(*App)
	if a.status.LatencyRing.Len() != 1 {
		t.Fatalf("LatencyRing.Len = %d, want 1", a.status.LatencyRing.Len())
	}
	if a.status.LatencyRing.Last() != 42 {
		t.Errorf("LatencyRing.Last = %v, want 42", a.status.LatencyRing.Last())
	}
}

func TestApp_TrafficResultMsg_FeedsSpeedRing(t *testing.T) {
	app := NewApp("test")
	app.width = 120
	app.height = 40
	app.updateLayoutQuick()

	app.Update(trafficResultMsg{stats: &core.TrafficStats{Uplink: 1000, Downlink: 2000}})
	app.status.PrevTrafficAt = time.Now().Unix() - 5
	model, _ := app.Update(trafficResultMsg{stats: &core.TrafficStats{Uplink: 6000, Downlink: 12000}})
	a := model.(*App)
	if a.status.SpeedRing.Len() == 0 {
		t.Error("SpeedRing should have a sample after the speed is computed")
	}
}

func TestApp_MetricKeyIsSet(t *testing.T) {
	app := NewApp("test")
	if app.status.MetricKey != "m" {
		t.Errorf("MetricKey = %q, want \"m\"", app.status.MetricKey)
	}
}

func TestImportSubscriptionCmd_ReturnsCommand(t *testing.T) {
	app := NewApp("test")
	cmd := app.importSubscriptionCmd("https://example.com/sub", "test-sub")
	if cmd == nil {
		t.Fatal("importSubscriptionCmd should return a non-nil command")
	}
}

func TestApp_WizardSubscriptionResultDispatches(t *testing.T) {
	app := NewApp("test")
	app.activePanel = PanelStatus // ensure the result block actually moves focus

	// A finished wizard carrying a subscription result. Build via the real
	// constructor so the modal is fully initialized; the exported result
	// fields are the contract the app's result block consumes.
	m := modals.NewWizardModal(app.width, app.height)
	m.Done = true
	m.SubURL = "https://example.com/sub"
	m.SubName = "s1"
	app.modal = m
	app.modalID = ModalWizard

	// Any subsequent message routes through the wizard result block.
	_, cmd := app.Update(trafficResultMsg{})

	if app.modalID != ModalNone {
		t.Errorf("wizard should be dismissed, modalID=%v", app.modalID)
	}
	if app.activePanel != PanelProfiles {
		t.Errorf("expected focus on PanelProfiles, got %v", app.activePanel)
	}
	if cmd == nil {
		t.Error("expected a subscription-import command to be dispatched")
	}
}

func TestApp_ShowWizardInjectsStartKey(t *testing.T) {
	app := NewApp("test")
	app.showWizardModal()
	wiz, ok := app.modal.(*modals.WizardModal)
	if !ok {
		t.Fatal("showWizardModal should set a *WizardModal")
	}
	if wiz.StartKey != commands.KeyDisplay(app.keys.Start) {
		t.Errorf("StartKey = %q, want %q", wiz.StartKey, commands.KeyDisplay(app.keys.Start))
	}
}
