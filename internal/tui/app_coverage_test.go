package tui

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/rtxnik/lazyray/internal/config"
	"github.com/rtxnik/lazyray/internal/tui/notify"
)

// --- Init ---

func TestApp_Init(t *testing.T) {
	app := NewApp("test")
	cmd := app.Init()
	if cmd == nil {
		t.Error("Init() should return a command batch")
	}
}

// --- startLoading ---

func TestApp_StartLoading(t *testing.T) {
	app := newTestApp()
	cmd := app.startLoading("Testing...")

	if !app.loading {
		t.Error("loading should be true after startLoading")
	}
	if app.loadingMsg != "Testing..." {
		t.Errorf("loadingMsg = %q, want Testing...", app.loadingMsg)
	}
	if cmd == nil {
		t.Error("startLoading should return a command")
	}
}

func TestApp_StopLoading(t *testing.T) {
	app := newTestApp()
	app.loading = true
	app.loadingMsg = "busy"

	app.stopLoading()
	if app.loading {
		t.Error("loading should be false after stopLoading")
	}
	if app.loadingMsg != "" {
		t.Errorf("loadingMsg = %q, want empty", app.loadingMsg)
	}
}

// --- updateLayoutFull ---

func TestApp_UpdateLayoutFull(t *testing.T) {
	app := newTestApp()
	app.updateLayoutFull()

	if app.profiles.Width <= 0 {
		t.Error("profiles width should be set after updateLayoutFull")
	}
	if app.profiles.Height <= 0 {
		t.Error("profiles height should be set after updateLayoutFull")
	}
}

// --- show modal methods ---

func TestApp_ShowImportModal(t *testing.T) {
	app := newTestApp()
	cmd := app.showImportModal()

	if app.modalID != ModalImport {
		t.Errorf("modalID = %d, want ModalImport", app.modalID)
	}
	if app.modal == nil {
		t.Error("modal should be set")
	}
	if cmd == nil {
		t.Error("showImportModal should return a command")
	}
}

func TestApp_ShowHelpModal(t *testing.T) {
	app := newTestApp()
	cmd := app.showHelpModal()

	if app.modalID != ModalHelp {
		t.Errorf("modalID = %d, want ModalHelp", app.modalID)
	}
	if app.modal == nil {
		t.Error("modal should be set")
	}
	_ = cmd
}

func TestApp_ShowSubscriptionModal(t *testing.T) {
	app := newTestApp()
	app.settings = config.DefaultSettings()
	cmd := app.showSubscriptionModal()

	if app.modalID != ModalSubscription {
		t.Errorf("modalID = %d, want ModalSubscription", app.modalID)
	}
	if app.modal == nil {
		t.Error("modal should be set")
	}
	_ = cmd
}

func TestApp_ShowConfirmModal(t *testing.T) {
	app := newTestApp()
	cmd := app.showConfirmModal("Test", "Are you sure?", "test-action")

	if app.modalID != ModalConfirm {
		t.Errorf("modalID = %d, want ModalConfirm", app.modalID)
	}
	if app.modal == nil {
		t.Error("modal should be set")
	}
	_ = cmd
}

func TestApp_ShowTunnelModal(t *testing.T) {
	app := newTestApp()
	cmd := app.showTunnelModal()

	if app.modalID != ModalTunnel {
		t.Errorf("modalID = %d, want ModalTunnel", app.modalID)
	}
	_ = cmd
}

func TestApp_ShowEditModal(t *testing.T) {
	app := newTestApp()
	profile := &app.servers.Profiles[0]
	cmd := app.showEditModal(profile)

	if app.modalID != ModalEdit {
		t.Errorf("modalID = %d, want ModalEdit", app.modalID)
	}
	if app.modal == nil {
		t.Error("modal should be set")
	}
	_ = cmd
}

func TestApp_ShowUpdateModal(t *testing.T) {
	app := newTestApp()
	cmd := app.showUpdateModal()

	if app.modalID != ModalUpdate {
		t.Errorf("modalID = %d, want ModalUpdate", app.modalID)
	}
	_ = cmd
}

func TestApp_ShowDoctorModal(t *testing.T) {
	app := newTestApp()
	cmd := app.showDoctorModal()

	if app.modalID != ModalDoctor {
		t.Errorf("modalID = %d, want ModalDoctor", app.modalID)
	}
	if app.modal == nil {
		t.Error("modal should be set")
	}
	_ = cmd
}

func TestApp_ShowQRModal(t *testing.T) {
	app := newTestApp()
	profile := &app.servers.Profiles[0]
	cmd := app.showQRModal(profile)

	if app.modalID != ModalQR {
		t.Errorf("modalID = %d, want ModalQR", app.modalID)
	}
	if app.modal == nil {
		t.Error("modal should be set")
	}
	_ = cmd
}

func TestApp_ShowDiffModal(t *testing.T) {
	app := newTestApp()
	profile := &app.servers.Profiles[0]
	cmd := app.showDiffModal(profile)

	if app.modalID != ModalDiff {
		t.Errorf("modalID = %d, want ModalDiff", app.modalID)
	}
	if app.modal == nil {
		t.Error("modal should be set")
	}
	_ = cmd
}

// --- Key press handlers ---

func TestApp_KeyPress_Help(t *testing.T) {
	app := newTestApp()
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}}
	model, _ := app.Update(msg)
	a := model.(*App)
	if a.modalID != ModalHelp {
		t.Errorf("after '?' press: modalID = %d, want ModalHelp", a.modalID)
	}
}

func TestApp_KeyPress_Import(t *testing.T) {
	app := newTestApp()
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}}
	model, _ := app.Update(msg)
	a := model.(*App)
	if a.modalID != ModalImport {
		t.Errorf("after 'i' press: modalID = %d, want ModalImport", a.modalID)
	}
}

func TestApp_KeyPress_Subscriptions(t *testing.T) {
	app := newTestApp()
	app.settings = config.DefaultSettings()
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'S'}}
	model, _ := app.Update(msg)
	a := model.(*App)
	if a.modalID != ModalSubscription {
		t.Errorf("after 'S' press: modalID = %d, want ModalSubscription", a.modalID)
	}
}

func TestApp_KeyPress_GroupFilter(t *testing.T) {
	app := newTestApp()
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}}
	model, _ := app.Update(msg)
	a := model.(*App)
	if a.groupFilter != "group1" {
		t.Errorf("after 'g' press: filter = %q, want group1", a.groupFilter)
	}
}

func TestApp_KeyPress_Rename(t *testing.T) {
	app := newTestApp()
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'R'}}
	model, _ := app.Update(msg)
	a := model.(*App)
	if !a.profiles.Renaming {
		t.Error("after 'R' press: should be renaming")
	}
}

func TestApp_KeyPress_Tab(t *testing.T) {
	app := newTestApp()
	msg := tea.KeyMsg{Type: tea.KeyTab}
	model, _ := app.Update(msg)
	a := model.(*App)
	if a.activePanel != PanelStatus {
		t.Errorf("after Tab: panel = %d, want PanelStatus", a.activePanel)
	}
}

func TestApp_KeyPress_Tunnel(t *testing.T) {
	app := newTestApp()
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}}
	model, _ := app.Update(msg)
	a := model.(*App)
	if a.modalID != ModalTunnel {
		t.Errorf("after 't' press: modalID = %d, want ModalTunnel", a.modalID)
	}
}

func TestApp_KeyPress_Update(t *testing.T) {
	app := newTestApp()
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'u'}}
	model, _ := app.Update(msg)
	a := model.(*App)
	if a.modalID != ModalUpdate {
		t.Errorf("after 'u' press: modalID = %d, want ModalUpdate", a.modalID)
	}
}

func TestApp_KeyPress_FilterLog(t *testing.T) {
	app := newTestApp()
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}}
	_, _ = app.Update(msg)
}

func TestApp_KeyPress_ToggleLog_LogPanel(t *testing.T) {
	app := newTestApp()
	app.activePanel = PanelLogs
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}}
	_, _ = app.Update(msg)
}

func TestApp_KeyPress_EditProfile(t *testing.T) {
	app := newTestApp()
	app.activePanel = PanelProfiles
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}}
	model, _ := app.Update(msg)
	a := model.(*App)
	if a.modalID != ModalEdit {
		t.Errorf("after 'e' on profiles: modalID = %d, want ModalEdit", a.modalID)
	}
}

func TestApp_KeyPress_Start(t *testing.T) {
	app := newTestApp()
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}}
	model, _ := app.Update(msg)
	a := model.(*App)
	if !a.loading {
		t.Error("after 's' press: should be loading (starting xray)")
	}
}

func TestApp_KeyPress_Restart(t *testing.T) {
	app := newTestApp()
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}}
	model, _ := app.Update(msg)
	a := model.(*App)
	if !a.loading {
		t.Error("after 'r' press: should be loading (restarting xray)")
	}
}

func TestApp_KeyPress_Doctor(t *testing.T) {
	app := newTestApp()
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}}
	model, _ := app.Update(msg)
	a := model.(*App)
	if a.modalID != ModalDoctor {
		t.Errorf("after 'h' press: modalID = %d, want ModalDoctor", a.modalID)
	}
}

func TestApp_KeyPress_TestAll(t *testing.T) {
	app := newTestApp()
	app.activePanel = PanelProfiles
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'T'}}
	model, _ := app.Update(msg)
	a := model.(*App)
	if !a.loading {
		t.Error("after 'T' press: should be loading (testing all)")
	}
}

func TestApp_KeyPress_Duplicate(t *testing.T) {
	app := newTestApp()
	app.activePanel = PanelProfiles
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'Y'}}
	_, _ = app.Update(msg)
}

// --- handleRenameKey ---

func TestApp_HandleRenameKey_Esc(t *testing.T) {
	app := newTestApp()
	app.profiles.StartRename()
	if !app.profiles.Renaming {
		t.Fatal("should be renaming")
	}

	msg := tea.KeyMsg{Type: tea.KeyEsc}
	model, _ := app.Update(msg)
	a := model.(*App)
	if a.profiles.Renaming {
		t.Error("Esc should cancel rename")
	}
}

// --- Window resize ---

func TestApp_WindowResize(t *testing.T) {
	app := newTestApp()
	msg := tea.WindowSizeMsg{Width: 200, Height: 60}
	model, _ := app.Update(msg)
	a := model.(*App)
	if a.width != 200 {
		t.Errorf("width = %d, want 200", a.width)
	}
	if a.height != 60 {
		t.Errorf("height = %d, want 60", a.height)
	}
}

// --- actionResultMsg ---

func TestApp_ActionResultMsg_Success(t *testing.T) {
	app := newTestApp()
	app.loading = true

	model, _ := app.Update(actionResultMsg{message: "done"})
	a := model.(*App)
	if a.loading {
		t.Error("loading should be false")
	}
	if a.activeTail == nil || a.activeTail.Message != "done" {
		t.Errorf("activeTail = %v, want message 'done'", a.activeTail)
	}
}

func TestApp_ActionResultMsg_Error(t *testing.T) {
	app := newTestApp()
	app.loading = true

	model, _ := app.Update(actionResultMsg{err: errTest})
	a := model.(*App)
	if a.loading {
		t.Error("loading should be false")
	}
	if a.activeTail == nil || a.activeTail.Severity != notify.Error {
		t.Error("activeTail should be an Error notice")
	}
}

// --- View with modal overlay ---

func TestApp_View_WithModal(t *testing.T) {
	app := newTestApp()
	app.showHelpModal()
	view := app.View()
	if view == "" {
		t.Error("View with modal should not be empty")
	}
}

// --- DuplicateSelectedProfile ---

func TestApp_DuplicateSelectedProfile(t *testing.T) {
	app := newTestApp()
	initialCount := len(app.servers.Profiles)
	app.duplicateSelectedProfile()

	if len(app.servers.Profiles) != initialCount+1 {
		t.Errorf("profile count = %d, want %d", len(app.servers.Profiles), initialCount+1)
	}
	dup := app.servers.Profiles[len(app.servers.Profiles)-1]
	if dup.Name != "Profile A (copy)" {
		t.Errorf("dup name = %q, want 'Profile A (copy)'", dup.Name)
	}
	if dup.Default {
		t.Error("duplicated profile should not be default")
	}
}

// --- ExportProfile ---

func TestApp_ExportProfile(t *testing.T) {
	app := newTestApp()
	cmd := app.exportProfile()
	_ = cmd
}

// --- Delete key ---

func TestApp_KeyPress_Delete(t *testing.T) {
	app := newTestApp()
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}}
	model, _ := app.Update(msg)
	a := model.(*App)
	if a.modalID != ModalConfirm {
		t.Errorf("after 'd' press: modalID = %d, want ModalConfirm", a.modalID)
	}
}

// --- QR export key ---

func TestApp_KeyPress_QRExport(t *testing.T) {
	app := newTestApp()
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'Q'}}
	model, _ := app.Update(msg)
	a := model.(*App)
	if a.modalID != ModalQR {
		t.Errorf("after 'Q' press: modalID = %d, want ModalQR", a.modalID)
	}
}

// --- Config diff key ---

func TestApp_KeyPress_ConfigDiff(t *testing.T) {
	app := newTestApp()
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'D'}}
	model, _ := app.Update(msg)
	a := model.(*App)
	if a.modalID != ModalDiff {
		t.Errorf("after 'D' press: modalID = %d, want ModalDiff", a.modalID)
	}
}

// --- Enter on profiles ---

func TestApp_KeyPress_Enter_Profiles(t *testing.T) {
	app := newTestApp()
	app.activePanel = PanelProfiles
	msg := tea.KeyMsg{Type: tea.KeyEnter}
	model, _ := app.Update(msg)
	a := model.(*App)
	if !a.loading {
		t.Error("after Enter on profiles: should be loading (testing connection)")
	}
}

// --- Logs panel Up/Down ---

func TestApp_LogsPanelNavigation(t *testing.T) {
	app := newTestApp()
	app.activePanel = PanelLogs

	downMsg := tea.KeyMsg{Type: tea.KeyDown}
	_, _ = app.Update(downMsg)

	upMsg := tea.KeyMsg{Type: tea.KeyUp}
	_, _ = app.Update(upMsg)
}

// --- handleLogInputKey ---

func TestApp_HandleLogInputKey(t *testing.T) {
	app := newTestApp()
	app.logs.ToggleSearch()

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}}
	_, _ = app.Update(msg)

	escMsg := tea.KeyMsg{Type: tea.KeyEsc}
	_, _ = app.Update(escMsg)
}

func TestApp_HandleLogInputKey_FilterEnter(t *testing.T) {
	app := newTestApp()
	app.logs.ToggleFilter()

	enterMsg := tea.KeyMsg{Type: tea.KeyEnter}
	_, _ = app.Update(enterMsg)
}

// --- Modal dismiss with Esc ---

func TestApp_ModalDismiss_Esc(t *testing.T) {
	app := newTestApp()
	app.showHelpModal()

	if app.modalID != ModalHelp {
		t.Fatal("should have help modal")
	}

	msg := tea.KeyMsg{Type: tea.KeyEsc}
	model, _ := app.Update(msg)
	a := model.(*App)
	if a.modalID != ModalNone {
		t.Errorf("after Esc: modalID = %d, want ModalNone", a.modalID)
	}
	if a.modal != nil {
		t.Error("modal should be nil after Esc")
	}
}

// --- setMessage / setError ---

func TestApp_SetMessage(t *testing.T) {
	app := newTestApp()

	cmd := app.setMessage("hello")
	if app.activeTail == nil || app.activeTail.Message != "hello" {
		t.Errorf("activeTail = %v, want message 'hello'", app.activeTail)
	}
	if app.activeTail.Severity != notify.Success {
		t.Errorf("severity = %v, want Success", app.activeTail.Severity)
	}
	if cmd == nil {
		t.Error("setMessage should return a dwell-clear cmd")
	}
}

func TestApp_SetError(t *testing.T) {
	app := newTestApp()

	cmd := app.setError(errTest)
	if app.activeTail == nil || app.activeTail.Message != "test error" {
		t.Errorf("activeTail = %v, want message 'test error'", app.activeTail)
	}
	if app.activeTail.Severity != notify.Error {
		t.Errorf("severity = %v, want Error", app.activeTail.Severity)
	}
	if cmd != nil {
		t.Error("setError should return nil cmd (Error is sticky)")
	}
}

func TestDwellForSeverities(t *testing.T) {
	if d, sticky := dwellFor(notify.Error); !sticky || d != 0 {
		t.Errorf("Error dwell = %v,%v; want 0,true", d, sticky)
	}
	if d, sticky := dwellFor(notify.Warning); sticky || d != 15*time.Second {
		t.Errorf("Warning dwell = %v,%v; want 15s,false", d, sticky)
	}
	for _, s := range []notify.Severity{notify.Info, notify.Success} {
		if d, sticky := dwellFor(s); sticky || d != 6*time.Second {
			t.Errorf("%v dwell = %v,%v; want 6s,false", s, d, sticky)
		}
	}
}

func TestStaleClearDoesNotZeroNewerTail(t *testing.T) {
	app := newTestApp()
	app.activeTail = &notify.Notice{ID: 5, Severity: notify.Info, Message: "new"}
	app.Update(clearTailMsg{id: 4}) // stale id
	if app.activeTail == nil {
		t.Error("stale clearTailMsg cleared a newer tail")
	}
}

// --- resizeDebounceMsg ---

func TestApp_ResizeDebounceMsg(t *testing.T) {
	app := newTestApp()
	app.resizeSeq = 5

	// Matching seq should trigger updateLayoutFull
	model, _ := app.Update(resizeDebounceMsg{seq: 5})
	_ = model.(*App)

	// Non-matching seq should be ignored
	model, _ = app.Update(resizeDebounceMsg{seq: 3})
	_ = model.(*App)
}

// --- Panel ID constants ---

func TestPanelConstants(t *testing.T) {
	if PanelProfiles != 0 {
		t.Errorf("PanelProfiles = %d, want 0", PanelProfiles)
	}
	if PanelStatus != 1 {
		t.Errorf("PanelStatus = %d, want 1", PanelStatus)
	}
	if PanelLogs != 2 {
		t.Errorf("PanelLogs = %d, want 2", PanelLogs)
	}
	if panelCount != 3 {
		t.Errorf("panelCount = %d, want 3", panelCount)
	}
}

// --- Modal ID constants ---

func TestModalConstants(t *testing.T) {
	if ModalNone != 0 {
		t.Errorf("ModalNone = %d, want 0", ModalNone)
	}
	if ModalImport != 1 {
		t.Errorf("ModalImport = %d, want 1", ModalImport)
	}
	if ModalHelp != 3 {
		t.Errorf("ModalHelp = %d, want 3", ModalHelp)
	}
	if ModalSubscription != 10 {
		t.Errorf("ModalSubscription = %d, want 10", ModalSubscription)
	}
}

// --- refreshStatus ---

func TestApp_RefreshStatus(t *testing.T) {
	app := newTestApp()
	app.refreshStatus()

	// Should set profile name from default profile
	if app.status.Profile != "Profile A" {
		t.Errorf("status.Profile = %q, want 'Profile A'", app.status.Profile)
	}
}

// --- editorFinishedMsg ---

func TestApp_EditorFinishedMsg_Error(t *testing.T) {
	app := newTestApp()
	model, _ := app.Update(editorFinishedMsg{err: errTest})
	a := model.(*App)
	if a.activeTail == nil || a.activeTail.Severity != notify.Error {
		t.Error("activeTail should be an Error notice after editor error")
	}
}

// --- lifecycle poll-diff ---

func TestLifecycleNoticesOnTransitions(t *testing.T) {
	app := NewApp("test")

	// First call establishes the baseline; emits nothing.
	if cmds := app.diffSnapshot(statusSnapshot{alive: true, xrayPID: 100}); len(cmds) != 0 {
		t.Fatalf("baseline emitted %d notices, want 0", len(cmds))
	}

	// up -> down => Error "proxy stopped"
	app.diffSnapshot(statusSnapshot{alive: false})
	tail := app.activeTail
	if tail == nil || tail.Severity != notify.Error || tail.Message != "proxy stopped" {
		t.Fatalf("down transition tail = %v, want Error 'proxy stopped'", tail)
	}

	// down -> up => Info "proxy started"
	app.diffSnapshot(statusSnapshot{alive: true, xrayPID: 200})
	if app.activeTail == nil || app.activeTail.Severity != notify.Info || app.activeTail.Message != "proxy started" {
		t.Fatalf("up transition tail = %v, want Info 'proxy started'", app.activeTail)
	}

	// xrayPID change under a live supervisor => Warning "xray restarted (self-heal)"
	app.diffSnapshot(statusSnapshot{alive: true, xrayPID: 201})
	if app.activeTail == nil || app.activeTail.Severity != notify.Warning || app.activeTail.Message != "xray restarted (self-heal)" {
		t.Fatalf("pid change tail = %v, want Warning 'xray restarted (self-heal)'", app.activeTail)
	}
}

func TestApp_EditorFinishedMsg_Success(t *testing.T) {
	app := newTestApp()
	model, _ := app.Update(editorFinishedMsg{})
	a := model.(*App)
	if a.activeTail == nil || a.activeTail.Message != "config saved" {
		t.Errorf("activeTail = %v, want message 'config saved'", a.activeTail)
	}
}
