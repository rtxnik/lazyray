package tui

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	appsvc "github.com/rtxnik/lazyray/internal/app"
	"github.com/rtxnik/lazyray/internal/clihint"
	"github.com/rtxnik/lazyray/internal/config"
	"github.com/rtxnik/lazyray/internal/core"
	"github.com/rtxnik/lazyray/internal/doctor"
	"github.com/rtxnik/lazyray/internal/lifecycle"
	"github.com/rtxnik/lazyray/internal/tui/commands"
	"github.com/rtxnik/lazyray/internal/tui/modals"
	"github.com/rtxnik/lazyray/internal/tui/notify"
	"github.com/rtxnik/lazyray/internal/tui/panels"
	"github.com/rtxnik/lazyray/internal/tui/theme"
)

// PanelID identifies which panel is active.
type PanelID int

const (
	PanelProfiles PanelID = iota
	PanelStatus
	PanelLogs
	panelCount
)

// ModalID identifies which modal is shown.
type ModalID int

const (
	ModalNone ModalID = iota
	ModalImport
	ModalDoctor
	ModalHelp
	ModalConfirm
	ModalUpdate
	ModalTunnel
	ModalEdit
	ModalQR
	ModalDiff
	ModalSubscription
	ModalRouting
	ModalWizard
	ModalActivity
	ModalPalette
)

// Messages
type statusTickMsg time.Time
type logTickMsg time.Time
type exitIPResultMsg struct {
	ip      string
	latency time.Duration
	err     error
}
type actionResultMsg struct {
	message string
	err     error
}
type trafficResultMsg struct {
	stats *core.TrafficStats
}
type updateCheckResultMsg struct {
	release *core.ReleaseInfo
	err     error
}
type trafficTickMsg time.Time
type clearTailMsg struct{ id uint64 }
type editorFinishedMsg struct{ err error }
type resizeDebounceMsg struct{ seq int }
type shutdownCompleteMsg struct{}
type testAllResultMsg struct {
	results []profileTestResult
	err     error
}
type subscriptionResultMsg struct {
	added   int
	updated int
	err     error
}
type subscriptionAutoRefreshMsg struct{}
type subscriptionAutoRefreshResultMsg struct {
	totalAdded   int
	totalUpdated int
	err          error
}

// profileTestResult holds the latency test result for a single profile.
type profileTestResult struct {
	name    string
	latency time.Duration
	skipped bool
	err     error
}

// App is the main TUI model.
type App struct {
	// Panels
	profiles panels.ProfilesPanel
	status   panels.StatusPanel
	logs     panels.LogsPanel

	// State
	activePanel PanelID
	modalID     ModalID
	modal       tea.Model

	// Data
	servers  *config.ServersConfig
	settings *config.Settings
	xray     *core.XrayProcess
	tunnels  *core.TunnelManager
	version  string
	svc      *appsvc.Service

	// Dimensions
	width    int
	height   int
	keys     commands.KeyMap
	registry commands.Registry

	// Notification pipeline (E2c): durable in-memory ring + the status-bar tail.
	notices    *notify.Log
	activeTail *notify.Notice  // currently shown tail; nil = none
	notifSeen  uint64          // highest notice ID seen when Activity was last opened
	prevSnap   *statusSnapshot // previous lifecycle snapshot for poll-diff

	// Status tracking
	statusTickCount int
	fetchingExitIP  bool

	// Spinner for async operations
	spinner    spinner.Model
	loading    bool
	loadingMsg string

	// Update notification
	availableUpdate string

	// Resize debounce
	resizeSeq int

	// Group filter state
	groupFilter string
	allGroups   []string

	// Shutdown state
	shuttingDown bool
}

// NewApp creates a new TUI application.
func NewApp(version string) *App {
	servers, _ := config.LoadServers()
	if servers == nil {
		servers = &config.ServersConfig{}
	}

	settings, _ := config.LoadSettings()
	if settings == nil {
		settings = config.DefaultSettings()
	}

	// Apply theme from settings
	SetTheme(settings.UI.Theme)

	profilesPanel := panels.NewProfilesPanel()
	profilesPanel.SetProfiles(servers.Profiles)

	statusPanel := panels.NewStatusPanel()
	statusPanel.Settings = settings

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(colorAquaBr)

	km := commands.DefaultKeyMap()
	statusPanel.MetricKey = commands.KeyDisplay(km.ToggleMetric)
	profilesPanel.ImportKey = commands.KeyDisplay(km.Import)
	profilesPanel.SubsKey = commands.KeyDisplay(km.Subscriptions)
	statusPanel.ConnectKey = commands.KeyDisplay(km.Enter)
	statusPanel.DoctorKey = commands.KeyDisplay(km.Doctor)

	return &App{
		profiles:    profilesPanel,
		status:      statusPanel,
		logs:        panels.NewLogsPanel(settings.UI.LogLines),
		activePanel: PanelProfiles,
		servers:     servers,
		settings:    settings,
		xray:        core.NewXrayProcess(),
		tunnels:     core.NewTunnelManager(),
		version:     version,
		keys:        km,
		registry:    commands.New(km),
		notices:     notify.New(200),
		spinner:     s,
		svc:         appsvc.NewService(),
	}
}

// Init initializes the TUI.
func (a *App) Init() tea.Cmd {
	// Refresh status immediately so we show Running if xray is already up
	a.refreshStatus()
	a.logs.Refresh()

	// Check xray version compatibility
	a.status.VersionWarning = core.CheckXrayVersionCompat()

	cmds := []tea.Cmd{
		a.statusTick(),
		trafficTick(),
		logTick(),
		a.spinner.Tick,
	}

	if a.settings.Update.AutoCheck {
		cmds = append(cmds, checkForUpdate())
	}

	if a.hasAutoRefreshSubscriptions() {
		cmds = append(cmds, subscriptionAutoRefreshTick())
	}

	// Show onboarding wizard if no profiles exist and servers.yaml doesn't exist
	if len(a.servers.Profiles) == 0 {
		if _, err := os.Stat(config.ServersPath()); os.IsNotExist(err) {
			cmds = append(cmds, a.showWizardModal())
		}
	}

	return tea.Batch(cmds...)
}

func checkForUpdate() tea.Cmd {
	return func() tea.Msg {
		settings, _ := config.LoadSettings()
		if settings == nil {
			settings = config.DefaultSettings()
		}
		release, err := core.CheckUpdate(settings.Update.XrayVersion)
		return updateCheckResultMsg{release: release, err: err}
	}
}

func (a *App) statusTick() tea.Cmd {
	interval := time.Duration(a.settings.UI.RefreshInterval) * time.Second
	if interval <= 0 {
		interval = 5 * time.Second
	}
	return tea.Tick(interval, func(t time.Time) tea.Msg {
		return statusTickMsg(t)
	})
}

func trafficTick() tea.Cmd {
	return tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
		return trafficTickMsg(t)
	})
}

func logTick() tea.Cmd {
	return tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
		return logTickMsg(t)
	})
}

func subscriptionAutoRefreshTick() tea.Cmd {
	// Check every hour; the actual refresh decision is per-subscription interval
	return tea.Tick(1*time.Hour, func(t time.Time) tea.Msg {
		return subscriptionAutoRefreshMsg{}
	})
}

// notify appends a notice to the ring, makes it the status-bar tail, and
// schedules the severity-tiered dwell clear (Success/Info ~6s, Warning ~15s,
// Error sticky). The clear is keyed to the notice ID so a stale timer never
// clears a newer tail.
func (a *App) notify(n notify.Notice) tea.Cmd {
	n.Time = time.Now()
	if n.Severity == notify.Error && n.Hint == "" {
		n.Hint = teachHint(n.Source, commands.KeyDisplay(a.keys.Doctor))
	}
	stored := a.notices.Add(n)
	a.activeTail = &stored
	d, sticky := dwellFor(stored.Severity)
	if sticky {
		return nil
	}
	id := stored.ID
	return tea.Tick(d, func(time.Time) tea.Msg { return clearTailMsg{id: id} })
}

// dwellFor returns how long the tail shows a severity, and whether it is sticky.
func dwellFor(s notify.Severity) (time.Duration, bool) {
	switch s {
	case notify.Error:
		return 0, true
	case notify.Warning:
		return 15 * time.Second, false
	default: // Info, Success
		return 6 * time.Second, false
	}
}

func (a *App) setMessage(msg string) tea.Cmd {
	return a.notify(notify.Notice{Severity: notify.Success, Message: msg})
}

func (a *App) setError(err error) tea.Cmd {
	return a.notify(notify.Notice{Severity: notify.Error, Message: err.Error(), Hint: extractHint(err)})
}

func (a *App) setWarning(msg string) tea.Cmd {
	return a.notify(notify.Notice{Severity: notify.Warning, Message: msg})
}

// setInfo is part of the notify seam reserved for later stages (E2e/E2f); it is
// not yet called from a key path, so it is intentionally retained unused here.
//
//nolint:unused
func (a *App) setInfo(msg string) tea.Cmd {
	return a.notify(notify.Notice{Severity: notify.Info, Message: msg})
}

// extractHint surfaces a clihint.Error's actionable next step, if any.
func extractHint(err error) string {
	var he *clihint.Error
	if errors.As(err, &he) {
		return he.Hint
	}
	return ""
}

// teachHint returns a doctor-vocabulary fallback hint for an error whose source
// is known, used when the error carries no clihint.Error hint of its own.
func teachHint(source, doctorKey string) string {
	switch source {
	case "profile":
		return "verify the profile, then [" + doctorKey + "] to diagnose"
	case "subscription":
		return "check the subscription URL, then [" + doctorKey + "]"
	case "update":
		return "see logs, or retry"
	default: // "", "lifecycle", anything else
		return "press [" + doctorKey + "] to diagnose"
	}
}

// Update handles messages.
func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// During shutdown, only process the completion message
	if a.shuttingDown {
		if _, ok := msg.(shutdownCompleteMsg); ok {
			return a, tea.Quit
		}
		return a, nil
	}

	// Handle modal first
	if a.modalID != ModalNone && a.modal != nil {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			// Wizard handles esc internally (back navigation)
			if key.Matches(msg, a.keys.Escape) && a.modalID != ModalWizard {
				a.modalID = ModalNone
				a.modal = nil
				return a, nil
			}
		}

		newModal, cmd := a.modal.Update(msg)
		a.modal = newModal

		// Check for modal results
		switch m := a.modal.(type) {
		case *modals.ImportModal:
			if m.Done {
				a.modalID = ModalNone
				a.modal = nil
				if m.Profile != nil {
					importCmd := a.handleImportResult(m.Profile)
					return a, tea.Batch(cmd, importCmd)
				}
			}
		case *modals.DoctorModal:
			if m.Done {
				a.modalID = ModalNone
				a.modal = nil
			}
		case *modals.ActivityModal:
			if m.Done {
				a.modalID = ModalNone
				a.modal = nil
			}
		case *modals.PaletteModal:
			if m.Done {
				a.modalID = ModalNone
				a.modal = nil
				if m.Selected != nil {
					return a, tea.Batch(cmd, a.launchCommand(*m.Selected))
				}
			}
		case *modals.ConfirmModal:
			if m.Done {
				a.modalID = ModalNone
				a.modal = nil
				if m.Confirmed {
					confirmCmd := a.handleConfirmResult(m.Action)
					return a, tea.Batch(cmd, confirmCmd)
				}
			}
		case *modals.EditModal:
			if m.Done {
				a.modalID = ModalNone
				a.modal = nil
				if m.Profile != nil {
					editCmd := a.handleEditResult(m.Profile)
					return a, tea.Batch(cmd, editCmd)
				}
			}
		case *modals.QRModal:
			if m.Done {
				a.modalID = ModalNone
				a.modal = nil
			}
		case *modals.DiffModal:
			if m.Done {
				a.modalID = ModalNone
				a.modal = nil
			}
		case *modals.SubscriptionModal:
			if m.Done {
				a.modalID = ModalNone
				a.modal = nil
				if m.Action != modals.SubActionNone {
					subCmd := a.handleSubscriptionResult(m)
					return a, tea.Batch(cmd, subCmd)
				}
			}
		case *modals.RoutingModal:
			if m.Done {
				a.modalID = ModalNone
				a.modal = nil
				if m.Routing != nil {
					routingCmd := a.handleRoutingResult(m.Routing)
					return a, tea.Batch(cmd, routingCmd)
				}
			}
		case *modals.WizardModal:
			if m.Done {
				a.modalID = ModalNone
				a.modal = nil
				a.activePanel = PanelProfiles
				if m.Skipped {
					// Create empty servers.yaml so wizard won't show again
					_ = config.SaveServers(&config.ServersConfig{})
				} else if m.Profile != nil {
					wizardCmd := a.handleImportResult(m.Profile)
					return a, tea.Batch(cmd, wizardCmd)
				} else if m.SubURL != "" {
					subCmd := a.importSubscriptionCmd(m.SubURL, m.SubName)
					return a, tea.Batch(cmd, subCmd)
				}
			}
		}

		return a, cmd
	}

	switch msg := msg.(type) {
	case spinner.TickMsg:
		if a.loading {
			var cmd tea.Cmd
			a.spinner, cmd = a.spinner.Update(msg)
			return a, cmd
		}
		return a, nil

	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a.updateLayoutQuick()
		a.resizeSeq++
		seq := a.resizeSeq
		return a, tea.Tick(100*time.Millisecond, func(time.Time) tea.Msg {
			return resizeDebounceMsg{seq: seq}
		})

	case resizeDebounceMsg:
		if msg.seq == a.resizeSeq {
			a.updateLayoutFull()
		}
		return a, nil

	case statusTickMsg:
		a.refreshStatus()
		a.statusTickCount++
		var cmds []tea.Cmd
		cmds = append(cmds, a.statusTick())
		cmds = append(cmds, a.lifecycleNotices()...)
		if lifecycle.SupervisorAlive() {
			// Fetch exit IP every 6 ticks (30 seconds)
			if a.statusTickCount%6 == 1 && !a.fetchingExitIP {
				a.fetchingExitIP = true
				cmds = append(cmds, a.fetchExitIP())
			}
		}
		return a, tea.Batch(cmds...)

	case trafficTickMsg:
		var cmds []tea.Cmd
		cmds = append(cmds, trafficTick())
		if lifecycle.SupervisorAlive() {
			cmds = append(cmds, fetchTraffic())
		}
		return a, tea.Batch(cmds...)

	case exitIPResultMsg:
		a.fetchingExitIP = false
		if msg.err == nil {
			a.status.ExitIP = msg.ip
			a.status.Latency = fmt.Sprintf("%dms", msg.latency.Milliseconds())
			a.status.LatencyRing.Push(float64(msg.latency.Milliseconds()))
		}
		return a, nil

	case trafficResultMsg:
		if msg.stats != nil {
			if a.status.PrevTraffic != nil && a.status.PrevTrafficAt > 0 {
				now := time.Now().Unix()
				elapsed := now - a.status.PrevTrafficAt
				if elapsed > 0 {
					upDelta := msg.stats.Uplink - a.status.PrevTraffic.Uplink
					dnDelta := msg.stats.Downlink - a.status.PrevTraffic.Downlink
					if upDelta < 0 {
						upDelta = 0
					}
					if dnDelta < 0 {
						dnDelta = 0
					}
					a.status.UpSpeed = core.FormatBytes(upDelta/elapsed) + "/s"
					a.status.DnSpeed = core.FormatBytes(dnDelta/elapsed) + "/s"
					a.status.SpeedRing.Push(float64(dnDelta / elapsed))
				}
			}
			a.status.PrevTraffic = msg.stats
			a.status.PrevTrafficAt = time.Now().Unix()

			// Record traffic for persistent history
			sm := core.GetStatsManager()
			sm.RecordTraffic(msg.stats)
			today := sm.TodayStats()
			a.status.TodayBytes = today.Uplink + today.Downlink
			// Save periodically (every 12 ticks = ~1 minute)
			if a.statusTickCount%12 == 0 {
				_ = sm.Save()
			}
		}
		a.status.Traffic = msg.stats
		return a, nil

	case updateCheckResultMsg:
		if msg.err == nil && msg.release != nil {
			current := core.GetXrayVersion()
			latest := msg.release.TagName
			if latest != "" && latest != current {
				a.availableUpdate = latest
			}
		}
		return a, nil

	case connectionTestMsg:
		a.stopLoading()
		if msg.err != nil {
			// Show warning but still switch
			warnCmd := a.setWarning(fmt.Sprintf("%s (switching anyway)", msg.err))
			switchCmd := a.finalizeProfileSwitch(msg.profileName)
			return a, tea.Batch(warnCmd, switchCmd)
		}
		return a, a.finalizeProfileSwitch(msg.profileName)

	case testAllResultMsg:
		a.stopLoading()
		if msg.err != nil {
			return a, a.setError(msg.err)
		}
		a.applyTestAllResults(msg.results)
		return a, a.setMessage(fmt.Sprintf("tested %d profiles", len(msg.results)))

	case subscriptionAutoRefreshMsg:
		return a, a.autoRefreshSubscriptions()

	case subscriptionAutoRefreshResultMsg:
		var cmds []tea.Cmd
		cmds = append(cmds, subscriptionAutoRefreshTick())
		if msg.err != nil {
			cmds = append(cmds, a.setError(msg.err))
			return a, tea.Batch(cmds...)
		}
		if msg.totalAdded > 0 || msg.totalUpdated > 0 {
			// Reload servers
			servers, _ := config.LoadServers()
			if servers != nil {
				a.servers = servers
				a.refreshGroups()
				a.applyGroupFilter()
			}
			settings, _ := config.LoadSettings()
			if settings != nil {
				a.settings = settings
			}
			cmds = append(cmds, a.setMessage(fmt.Sprintf("auto-refresh: %d added, %d updated", msg.totalAdded, msg.totalUpdated)))
		}
		return a, tea.Batch(cmds...)

	case subscriptionResultMsg:
		a.stopLoading()
		if msg.err != nil {
			return a, a.setError(msg.err)
		}
		// Reload servers
		servers, _ := config.LoadServers()
		if servers != nil {
			a.servers = servers
			a.refreshGroups()
			a.applyGroupFilter()
		}
		// Reload settings for updated subscription list
		settings, _ := config.LoadSettings()
		if settings != nil {
			a.settings = settings
		}
		return a, a.setMessage(fmt.Sprintf("subscription: %d added, %d updated", msg.added, msg.updated))

	case actionResultMsg:
		a.stopLoading()
		a.refreshStatus()
		var clearCmd tea.Cmd
		if msg.err != nil {
			clearCmd = a.setError(msg.err)
		} else if msg.message != "" {
			clearCmd = a.setMessage(msg.message)
		}
		return a, clearCmd

	case clearTailMsg:
		if a.activeTail != nil && a.activeTail.ID == msg.id {
			a.activeTail = nil
		}
		return a, nil

	case editorFinishedMsg:
		if msg.err != nil {
			return a, a.setError(fmt.Errorf("editor: %w", msg.err))
		}
		if lifecycle.SupervisorAlive() {
			return a, a.restartXray()
		}
		return a, a.setMessage("config saved")

	case logTickMsg:
		a.logs.Refresh()
		return a, logTick()

	case tea.KeyMsg:
		// If profiles panel is in rename mode, handle that first
		if a.profiles.Renaming {
			return a.handleRenameKey(msg)
		}
		// If profiles panel is in search mode, handle that first
		if a.profiles.Searching {
			return a.handleProfileSearchKey(msg)
		}
		// If logs panel is in filter/search input mode, handle that first
		if a.logs.Filtering || a.logs.Searching {
			return a.handleLogInputKey(msg)
		}
		return a.handleKeyPress(msg)
	}

	return a, nil
}

func (a *App) handleRenameKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, a.keys.Enter):
		newName, ok := a.profiles.ConfirmRename()
		if !ok {
			return a, nil
		}
		idx := a.profiles.RenameIdx
		if idx < 0 || idx >= len(a.servers.Profiles) {
			return a, nil
		}
		a.servers.Profiles[idx].Name = newName
		if err := config.SaveServers(a.servers); err != nil {
			return a, a.setError(fmt.Errorf("saving profile: %w", err))
		}
		a.profiles.SetProfiles(a.servers.Profiles)
		return a, a.setMessage(fmt.Sprintf("renamed to %s", newName))
	case key.Matches(msg, a.keys.Escape):
		a.profiles.CancelRename()
		return a, nil
	}

	var cmd tea.Cmd
	a.profiles.RenameInput, cmd = a.profiles.RenameInput.Update(msg)
	return a, cmd
}

func (a *App) handleProfileSearchKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, a.keys.Enter):
		// Confirm search (keep filter, exit search input mode)
		a.profiles.Searching = false
		return a, nil
	case key.Matches(msg, a.keys.Escape):
		a.profiles.CancelSearch()
		return a, nil
	case key.Matches(msg, a.keys.Up):
		a.profiles.MoveUpVisible()
		return a, nil
	case key.Matches(msg, a.keys.Down):
		a.profiles.MoveDownVisible()
		return a, nil
	}

	var cmd tea.Cmd
	a.profiles.SearchInput, cmd = a.profiles.SearchInput.Update(msg)
	a.profiles.ApplySearch()
	return a, cmd
}

func (a *App) handleLogInputKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, a.keys.Enter):
		if a.logs.Filtering {
			a.logs.ApplyFilter()
		} else if a.logs.Searching {
			a.logs.ApplySearch()
		}
		return a, nil
	case key.Matches(msg, a.keys.Escape):
		a.logs.CancelFilterSearch()
		return a, nil
	}

	// Forward to the active text input
	var cmd tea.Cmd
	if a.logs.Filtering {
		a.logs.FilterInput, cmd = a.logs.FilterInput.Update(msg)
		a.logs.ApplyFilter()
	} else if a.logs.Searching {
		a.logs.SearchInput, cmd = a.logs.SearchInput.Update(msg)
		a.logs.ApplySearch()
	}
	return a, cmd
}

func (a *App) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Clear move highlighting for any key that isn't a move key
	if a.profiles.Moving && !key.Matches(msg, a.keys.ShiftUp) && !key.Matches(msg, a.keys.ShiftDown) {
		a.profiles.Moving = false
	}

	switch {
	case key.Matches(msg, a.keys.Quit):
		if a.shuttingDown {
			return a, nil
		}
		a.shuttingDown = true
		return a, a.performShutdown()

	case key.Matches(msg, a.keys.Escape):
		a.activeTail = nil
		return a, nil

	case key.Matches(msg, a.keys.Tab):
		a.activePanel = (a.activePanel + 1) % panelCount
		return a, nil

	case key.Matches(msg, a.keys.ShiftTab):
		a.activePanel = (a.activePanel - 1 + panelCount) % panelCount
		return a, nil

	case key.Matches(msg, a.keys.Up):
		switch a.activePanel {
		case PanelProfiles:
			if a.profiles.SearchQuery != "" {
				a.profiles.MoveUpVisible()
			} else {
				a.profiles.MoveUp()
			}
		case PanelLogs:
			a.logs.Viewport.ScrollUp(1)
		}
		return a, nil

	case key.Matches(msg, a.keys.Down):
		switch a.activePanel {
		case PanelProfiles:
			if a.profiles.SearchQuery != "" {
				a.profiles.MoveDownVisible()
			} else {
				a.profiles.MoveDown()
			}
		case PanelLogs:
			a.logs.Viewport.ScrollDown(1)
		}
		return a, nil

	case key.Matches(msg, a.keys.Enter):
		if a.activePanel == PanelProfiles {
			return a, a.activateSelectedProfile()
		}
		return a, nil

	case key.Matches(msg, a.keys.Start):
		if lifecycle.SupervisorAlive() {
			return a, a.showConfirmModal("Disconnect", "Disconnect the session?", "stop")
		}
		return a, a.toggleXray()

	case key.Matches(msg, a.keys.Restart):
		return a, a.restartXray()

	case key.Matches(msg, a.keys.Doctor):
		return a, a.showDoctorModal()

	case key.Matches(msg, a.keys.Import):
		return a, a.showImportModal()

	case key.Matches(msg, a.keys.Help):
		return a, a.showHelpModal()

	case key.Matches(msg, a.keys.Activity):
		return a, a.showActivityModal()

	case key.Matches(msg, a.keys.Palette):
		return a, a.showPaletteModal()

	case key.Matches(msg, a.keys.ToggleLog):
		if a.activePanel == PanelProfiles {
			profile := a.profiles.SelectedProfile()
			if profile != nil {
				return a, a.showEditModal(profile)
			}
			return a, nil
		}
		a.logs.ToggleLogType()
		return a, nil

	case key.Matches(msg, a.keys.FilterLog):
		a.logs.ToggleFilter()
		return a, nil

	case key.Matches(msg, a.keys.SearchLog):
		if a.activePanel == PanelProfiles {
			a.profiles.StartSearch()
			return a, textinput.Blink
		}
		a.logs.ToggleSearch()
		return a, nil

	case key.Matches(msg, a.keys.Tunnel):
		return a, a.showTunnelModal()

	case key.Matches(msg, a.keys.Update):
		return a, a.showUpdateModal()

	case key.Matches(msg, a.keys.EditConfig):
		return a, a.editConfig()

	case key.Matches(msg, a.keys.ConfigDiff):
		if a.activePanel == PanelProfiles {
			profile := a.profiles.SelectedProfile()
			if profile != nil {
				return a, a.showDiffModal(profile)
			}
		}
		return a, nil

	case key.Matches(msg, a.keys.QRExport):
		if a.activePanel == PanelProfiles {
			profile := a.profiles.SelectedProfile()
			if profile != nil {
				return a, a.showQRModal(profile)
			}
		}
		return a, nil

	case key.Matches(msg, a.keys.Export):
		return a, a.exportProfile()

	case key.Matches(msg, a.keys.ShiftUp):
		if a.activePanel == PanelProfiles {
			if a.profiles.MoveProfileUp() {
				a.profiles.Moving = true
				a.servers.Profiles = a.profiles.Profiles
				if err := config.SaveServers(a.servers); err != nil {
					return a, a.setError(fmt.Errorf("saving profile order: %w", err))
				}
			}
		}
		return a, nil

	case key.Matches(msg, a.keys.ShiftDown):
		if a.activePanel == PanelProfiles {
			if a.profiles.MoveProfileDown() {
				a.profiles.Moving = true
				a.servers.Profiles = a.profiles.Profiles
				if err := config.SaveServers(a.servers); err != nil {
					return a, a.setError(fmt.Errorf("saving profile order: %w", err))
				}
			}
		}
		return a, nil

	case key.Matches(msg, a.keys.Rename):
		if a.activePanel == PanelProfiles && len(a.profiles.Profiles) > 0 {
			a.profiles.StartRename()
			return a, textinput.Blink
		}
		return a, nil

	case key.Matches(msg, a.keys.Subscriptions):
		return a, a.showSubscriptionModal()

	case key.Matches(msg, a.keys.FilterGroup):
		if a.activePanel == PanelProfiles {
			a.cycleGroupFilter()
			return a, a.setMessage(a.groupFilterLabel())
		}
		return a, nil

	case key.Matches(msg, a.keys.TestAll):
		if a.activePanel == PanelProfiles {
			return a, a.testAllProfiles()
		}
		return a, nil

	case key.Matches(msg, a.keys.Duplicate):
		if a.activePanel == PanelProfiles {
			return a, a.duplicateSelectedProfile()
		}
		return a, nil

	case key.Matches(msg, a.keys.ToggleCollapse):
		if a.activePanel == PanelProfiles {
			a.profiles.ToggleGroupCollapse()
		}
		return a, nil

	case key.Matches(msg, a.keys.ToggleMetric):
		if a.activePanel == PanelStatus {
			a.status.ToggleMetric()
		}
		return a, nil

	case key.Matches(msg, a.keys.RoutingEdit):
		if a.activePanel == PanelProfiles {
			profile := a.profiles.SelectedProfile()
			if profile != nil {
				return a, a.showRoutingModal(profile)
			}
		}
		return a, nil

	case key.Matches(msg, a.keys.Delete):
		if a.activePanel == PanelProfiles {
			profile := a.profiles.SelectedProfile()
			if profile != nil {
				return a, a.showConfirmModal("Delete Profile",
					fmt.Sprintf("Delete profile %q?", profile.Name), "delete:"+profile.Name)
			}
		}
		return a, nil
	}

	return a, nil
}

func (a *App) startLoading(msg string) tea.Cmd {
	a.loading = true
	a.loadingMsg = msg
	return a.spinner.Tick
}

func (a *App) stopLoading() {
	a.loading = false
	a.loadingMsg = ""
}

func (a *App) toggleXray() tea.Cmd {
	if lifecycle.SupervisorAlive() {
		loadCmd := a.startLoading("Disconnecting...")
		return tea.Batch(loadCmd, func() tea.Msg {
			if err := a.disconnectSession(); err != nil {
				return actionResultMsg{err: fmt.Errorf("disconnect failed: %w", err)}
			}
			return actionResultMsg{message: "disconnected"}
		})
	}
	loadCmd := a.startLoading("Connecting...")
	return tea.Batch(loadCmd, func() tea.Msg {
		if a.servers.DefaultProfile() == nil {
			return actionResultMsg{err: fmt.Errorf("no profile selected")}
		}
		if err := a.svc.Connect([]string{"--owner", string(lifecycle.OwnerTUI)}); err != nil {
			return actionResultMsg{err: fmt.Errorf("connect failed: %w", err)}
		}
		return actionResultMsg{message: "connected"}
	})
}

// disconnectSession delegates graceful teardown to the app service.
func (a *App) disconnectSession() error {
	return a.svc.Disconnect()
}

func (a *App) performShutdown() tea.Cmd {
	return func() tea.Msg {
		// Quitting the dashboard does NOT disconnect the session.
		// The supervisor owns xray + routing and keeps running.
		sm := core.GetStatsManager()
		_ = sm.Save()
		return shutdownCompleteMsg{}
	}
}

func (a *App) restartXray() tea.Cmd {
	loadCmd := a.startLoading("Reconnecting...")
	return tea.Batch(loadCmd, func() tea.Msg {
		_ = a.disconnectSession()
		if err := a.svc.Connect([]string{"--owner", string(lifecycle.OwnerTUI)}); err != nil {
			return actionResultMsg{err: fmt.Errorf("reconnect failed: %w", err)}
		}
		return actionResultMsg{message: "reconnected"}
	})
}

type connectionTestMsg struct {
	profileName string
	err         error
}

func (a *App) activateSelectedProfile() tea.Cmd {
	profile := a.profiles.SelectedProfile()
	if profile == nil {
		return nil
	}

	// If a session is running with the same profile, do nothing
	if lifecycle.SupervisorAlive() {
		current := a.servers.DefaultProfile()
		if current != nil && current.Name == profile.Name {
			return a.setMessage("already active")
		}
		// Running with different profile — confirm before switching
		return a.showConfirmModal(
			"Switch Profile",
			fmt.Sprintf("Switch to %s and restart?", profile.Name),
			"switch:"+profile.Name,
		)
	}

	// Not running — test connection, then switch + start
	name := profile.Name
	loadCmd := a.startLoading(fmt.Sprintf("testing %s...", name))
	settings := a.settings
	return tea.Batch(loadCmd, func() tea.Msg {
		r := core.ProbeProfile(*profile, lifecycle.ProbeContextFor(name, settings))
		var err error
		if r.Status == core.LivenessFail {
			err = r.Err
		}
		return connectionTestMsg{profileName: name, err: err}
	})
}

func (a *App) finalizeProfileSwitch(profileName string) tea.Cmd {
	var profile *config.Profile
	for i := range a.servers.Profiles {
		if a.servers.Profiles[i].Name == profileName {
			profile = &a.servers.Profiles[i]
			break
		}
	}
	if profile == nil {
		return a.setError(fmt.Errorf("profile %q not found", profileName))
	}

	for i := range a.servers.Profiles {
		a.servers.Profiles[i].Default = (a.servers.Profiles[i].Name == profileName)
	}
	if err := config.SaveServers(a.servers); err != nil {
		return a.setError(fmt.Errorf("saving profile: %w", err))
	}
	a.profiles.SetProfiles(a.servers.Profiles)

	if err := a.svc.WriteActiveConfig(profile, a.settings); err != nil {
		return a.setError(fmt.Errorf("config generation failed: %w", err))
	}

	if lifecycle.SupervisorAlive() {
		return a.restartXray()
	}
	// Auto-connect when selecting a profile
	return a.toggleXray()
}

// statusSnapshot is the lifecycle state compared between status ticks.
type statusSnapshot struct {
	alive        bool
	xrayPID      int
	startupErrAt time.Time // zero when there is no persisted startup error
}

// snapshotStatus reads the current lifecycle state from disk artifacts.
func (a *App) snapshotStatus() statusSnapshot {
	s := statusSnapshot{alive: lifecycle.SupervisorAlive()}
	if st, err := lifecycle.ReadState(); err == nil && st != nil {
		s.xrayPID = st.XrayPID
	}
	if se, err := lifecycle.ReadStartupError(); err == nil && se != nil {
		s.startupErrAt = se.Time
	}
	return s
}

// lifecycleNotices snapshots the current state and diffs it against the previous.
func (a *App) lifecycleNotices() []tea.Cmd {
	return a.diffSnapshot(a.snapshotStatus())
}

// diffSnapshot emits notices for transitions between the previous snapshot and
// cur, then stores cur as the new baseline. The first call only establishes the
// baseline (no notices), so a session that starts already-connected is silent.
func (a *App) diffSnapshot(cur statusSnapshot) []tea.Cmd {
	prev := a.prevSnap
	a.prevSnap = &cur
	if prev == nil {
		return nil
	}
	var cmds []tea.Cmd
	if !cur.startupErrAt.IsZero() && !cur.startupErrAt.Equal(prev.startupErrAt) {
		msg := "startup failed"
		if se, err := lifecycle.ReadStartupError(); err == nil && se != nil {
			msg = fmt.Sprintf("startup failed (%s): %s", se.Stage, se.Message)
		}
		cmds = append(cmds, a.notify(notify.Notice{Severity: notify.Error, Source: "lifecycle", Message: msg}))
	}
	switch {
	case prev.alive && !cur.alive:
		cmds = append(cmds, a.notify(notify.Notice{Severity: notify.Error, Source: "lifecycle", Message: "proxy stopped"}))
	case !prev.alive && cur.alive:
		cmds = append(cmds, a.notify(notify.Notice{Severity: notify.Info, Source: "lifecycle", Message: "proxy started"}))
	case cur.alive && prev.xrayPID != 0 && cur.xrayPID != 0 && cur.xrayPID != prev.xrayPID:
		cmds = append(cmds, a.notify(notify.Notice{Severity: notify.Warning, Source: "lifecycle", Message: "xray restarted (self-heal)"}))
	}
	return cmds
}

func (a *App) refreshStatus() {
	status := a.xray.Status()
	// The supervisor (internal/lifecycle) is the authoritative liveness source;
	// the TUI no longer owns xray, so derive "running" from it.
	status.Running = lifecycle.SupervisorAlive()
	a.status.Status = status

	a.status.StoppedReason = ""
	if !status.Running {
		if se, err := lifecycle.ReadStartupError(); err == nil && se != nil {
			a.status.StoppedReason = fmt.Sprintf("%s: %s", se.Stage, se.Message)
		}
	}

	profile := a.servers.DefaultProfile()
	if profile != nil {
		a.status.Profile = profile.Name
		a.status.ActiveProfile = profile
	}

	// Clear exit IP/latency/traffic if the session is down
	if !status.Running {
		a.status.ExitIP = ""
		a.status.Latency = ""
		a.status.Traffic = nil
	}
}

func (a *App) fetchExitIP() tea.Cmd {
	settings := a.settings
	return func() tea.Msg {
		start := time.Now()
		ip, err := core.GetExitIP(settings)
		latency := time.Since(start)
		if err != nil {
			return exitIPResultMsg{err: err}
		}
		return exitIPResultMsg{ip: strings.TrimSpace(ip), latency: latency}
	}
}

func fetchTraffic() tea.Cmd {
	return func() tea.Msg {
		stats := core.GetTrafficStats()
		return trafficResultMsg{stats: stats}
	}
}

// calcLayout returns dimensions for the 3-panel layout:
// Left column (profiles, full height), right column split (status top, logs bottom).
// Reserves 2 lines at the bottom for hotkeys bar and status bar.
// isNarrowTerminal returns true if the terminal is too narrow for side-by-side layout.
func (a *App) isNarrowTerminal() bool {
	return a.width < 80
}

// tailRows is the number of bottom rows the status tail occupies: 2 while a
// hinted Error/Warning tail is shown (message + "→ try:" line), else 1.
func (a *App) tailRows() int {
	if t := a.activeTail; t != nil && t.Hint != "" &&
		(t.Severity == notify.Error || t.Severity == notify.Warning) {
		return 2
	}
	return 1
}

func (a *App) calcLayout() (leftWidth, rightWidth, fullHeight, statusHeight, logsHeight int) {
	extra := a.tailRows() - 1
	if a.isNarrowTerminal() {
		// Single-column stacked layout
		fullWidth := a.width - 4
		if fullWidth < 10 {
			fullWidth = 10
		}
		leftWidth = fullWidth
		rightWidth = fullWidth
		// 4 lines reserved: 2 border overhead + 1 hotkeys bar + 1 status bar
		totalHeight := a.height - 4 - extra
		if totalHeight < 12 {
			totalHeight = 12
		}
		// Split: profiles gets 30%, status 30%, logs 40%
		fullHeight = totalHeight * 30 / 100
		statusHeight = totalHeight * 30 / 100
		logsHeight = totalHeight - fullHeight - statusHeight - 4
		if fullHeight < 4 {
			fullHeight = 4
		}
		if statusHeight < 3 {
			statusHeight = 3
		}
		if logsHeight < 3 {
			logsHeight = 3
		}
		return
	}

	leftWidth = a.width*30/100 - 2
	rightWidth = a.width - leftWidth - 6
	// 4 lines reserved: 2 border overhead + 1 hotkeys bar + 1 status bar
	fullHeight = a.height - 4 - extra
	statusHeight = fullHeight * 40 / 100
	logsHeight = fullHeight - statusHeight - 2

	if leftWidth < 10 {
		leftWidth = 10
	}
	if rightWidth < 20 {
		rightWidth = 20
	}
	if fullHeight < 6 {
		fullHeight = 6
	}
	if statusHeight < 3 {
		statusHeight = 3
	}
	if logsHeight < 3 {
		logsHeight = 3
	}
	return
}

// updateLayoutQuick sets panel dimensions without reinitializing the log viewport.
func (a *App) updateLayoutQuick() {
	leftWidth, rightWidth, fullHeight, statusHeight, logsHeight := a.calcLayout()

	a.profiles.Width = leftWidth
	a.profiles.Height = fullHeight
	a.status.Width = rightWidth
	a.status.Height = statusHeight
	a.status.Focused = a.activePanel == PanelStatus
	a.logs.Width = rightWidth
	a.logs.Height = logsHeight
	a.logs.Resize(rightWidth, logsHeight)
}

// updateLayoutFull reinitializes the log viewport and refreshes content.
func (a *App) updateLayoutFull() {
	_, rightWidth, _, _, logsHeight := a.calcLayout()

	if logsHeight > 0 && rightWidth > 0 {
		a.logs.Init(rightWidth, logsHeight)
		a.logs.Refresh()
	}
}

func (a *App) handleImportResult(profile *config.Profile) tea.Cmd {
	if _, err := a.svc.ImportProfile(a.servers, profile, false); err != nil {
		var dup *appsvc.DuplicateUUIDError
		if errors.As(err, &dup) {
			return a.setError(dup) // Error() == `UUID already used by profile %q`
		}
		return a.setError(fmt.Errorf("saving profile: %w", err))
	}
	a.profiles.SetProfiles(a.servers.Profiles)
	return a.setMessage(fmt.Sprintf("imported %s", profile.Name))
}

func (a *App) handleConfirmResult(action string) tea.Cmd {
	switch {
	case action == "stop":
		loadCmd := a.startLoading("Disconnecting...")
		return tea.Batch(loadCmd, func() tea.Msg {
			if err := a.disconnectSession(); err != nil {
				return actionResultMsg{err: fmt.Errorf("disconnect failed: %w", err)}
			}
			return actionResultMsg{message: "disconnected"}
		})

	case strings.HasPrefix(action, "delete:"):
		name := strings.TrimPrefix(action, "delete:")
		found := -1
		for i, p := range a.servers.Profiles {
			if p.Name == name {
				found = i
				break
			}
		}
		if found == -1 {
			return a.setError(fmt.Errorf("profile %q not found", name))
		}

		wasDefault := a.servers.Profiles[found].Default
		a.servers.Profiles = append(a.servers.Profiles[:found], a.servers.Profiles[found+1:]...)

		// If deleted profile was default, set the first remaining as default
		if wasDefault && len(a.servers.Profiles) > 0 {
			a.servers.Profiles[0].Default = true
		}

		if err := config.SaveServers(a.servers); err != nil {
			return a.setError(fmt.Errorf("saving servers: %w", err))
		}
		a.profiles.SetProfiles(a.servers.Profiles)
		return a.setMessage(fmt.Sprintf("deleted profile: %s", name))

	case strings.HasPrefix(action, "switch:"):
		name := strings.TrimPrefix(action, "switch:")
		return a.finalizeProfileSwitch(name)
	}
	return nil
}

func (a *App) showImportModal() tea.Cmd {
	m := modals.NewImportModal(a.width, a.height)
	a.modal = m
	a.modalID = ModalImport
	return m.Init()
}

func (a *App) showWizardModal() tea.Cmd {
	m := modals.NewWizardModal(a.width, a.height)
	m.StartKey = commands.KeyDisplay(a.keys.Start)
	a.modal = m
	a.modalID = ModalWizard
	return m.Init()
}

func (a *App) showDoctorModal() tea.Cmd {
	run := func() *doctor.Report {
		rep := doctor.RunAll(context.Background(), doctor.DefaultEnv())
		return &rep
	}
	m := modals.NewDoctorModal(a.width, a.height, run)
	a.modal = m
	a.modalID = ModalDoctor
	return m.Init()
}

func (a *App) showHelpModal() tea.Cmd {
	m := modals.NewHelpModal(a.registry, a.width, a.height)
	a.modal = m
	a.modalID = ModalHelp
	return m.Init()
}

func (a *App) showActivityModal() tea.Cmd {
	if n, ok := a.notices.Latest(); ok {
		a.notifSeen = n.ID // mark everything currently in the log as read
	}
	m := modals.NewActivityModal(a.notices.Entries(), a.width, a.height)
	a.modal = m
	a.modalID = ModalActivity
	return m.Init()
}

func (a *App) showPaletteModal() tea.Cmd {
	m := modals.NewPaletteModal(a.registry.Launchable(), a.width, a.height)
	a.modal = m
	a.modalID = ModalPalette
	return m.Init()
}

// launchCommand runs a palette-selected command: it records the launch in the
// notice log, focuses the command's scope panel so its handleKeyPress branch
// fires, then re-injects the command's primary key through the one dispatcher.
func (a *App) launchCommand(c commands.Command) tea.Cmd {
	infoCmd := a.notify(notify.Notice{Severity: notify.Info, Message: "→ " + c.Title, Source: "palette"})
	if p, ok := panelForScope(c.Scope); ok {
		a.activePanel = p
	}
	_, dispatchCmd := a.handleKeyPress(synthKey(c.Binding))
	return tea.Batch(infoCmd, dispatchCmd)
}

// synthKey builds a synthetic key message from a command's primary key. Safe
// for every launchable command by the single-rune registry invariant.
func synthKey(b key.Binding) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(b.Keys()[0])}
}

// panelForScope maps a command scope to the panel that must be focused for its
// handleKeyPress branch to fire; ok=false leaves the active panel unchanged.
func panelForScope(s commands.Scope) (PanelID, bool) {
	switch s {
	case commands.ScopeProfiles:
		return PanelProfiles, true
	case commands.ScopeLogs:
		return PanelLogs, true
	case commands.ScopeStatus:
		return PanelStatus, true
	}
	return PanelProfiles, false
}

// unreadCount is the number of notices added since Activity was last opened.
func (a *App) unreadCount() int {
	n := 0
	for _, e := range a.notices.Entries() { // newest-first
		if e.ID > a.notifSeen {
			n++
		} else {
			break
		}
	}
	return n
}

func (a *App) showTunnelModal() tea.Cmd {
	m := modals.NewTunnelModal(a.servers, a.tunnels, a.width, a.height)
	a.modal = m
	a.modalID = ModalTunnel
	return m.Init()
}

func (a *App) showConfirmModal(title, message, action string) tea.Cmd {
	m := modals.NewConfirmModal(title, message, action, a.width, a.height)
	a.modal = m
	a.modalID = ModalConfirm
	return m.Init()
}

func (a *App) editConfig() tea.Cmd {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi"
	}
	configPath := config.XrayConfigPath()
	c := exec.Command(editor, configPath)
	return tea.ExecProcess(c, func(err error) tea.Msg {
		return editorFinishedMsg{err: err}
	})
}

func (a *App) exportProfile() tea.Cmd {
	profile := a.servers.DefaultProfile()
	if profile == nil {
		return a.setError(fmt.Errorf("no profile to export"))
	}

	proxyURL := core.ToProxyURL(profile)
	if err := clipboard.WriteAll(proxyURL); err != nil {
		return a.setMessage(fmt.Sprintf("exported %s (clipboard unavailable)", profile.Name))
	}
	return a.setMessage(fmt.Sprintf("exported %s to clipboard", profile.Name))
}

func (a *App) showEditModal(profile *config.Profile) tea.Cmd {
	m := modals.NewEditModal(profile, a.width, a.height)
	a.modal = m
	a.modalID = ModalEdit
	return m.Init()
}

func (a *App) handleEditResult(updated *config.Profile) tea.Cmd {
	idx := a.profiles.Selected
	if idx < 0 || idx >= len(a.servers.Profiles) {
		return a.setError(fmt.Errorf("no profile selected"))
	}

	// Preserve fields not in the edit form
	updated.Default = a.servers.Profiles[idx].Default
	updated.SSH = a.servers.Profiles[idx].SSH
	updated.ExpectedExitIP = a.servers.Profiles[idx].ExpectedExitIP
	updated.Routing = a.servers.Profiles[idx].Routing

	a.servers.Profiles[idx] = *updated
	if err := config.SaveServers(a.servers); err != nil {
		return a.setError(fmt.Errorf("saving profile: %w", err))
	}
	a.profiles.SetProfiles(a.servers.Profiles)

	// Regenerate xray config and restart if this is the active profile
	if updated.Default {
		if err := a.svc.WriteActiveConfig(updated, a.settings); err != nil {
			return a.setError(fmt.Errorf("config generation failed: %w", err))
		}
		if lifecycle.SupervisorAlive() {
			return a.restartXray()
		}
	}
	return a.setMessage(fmt.Sprintf("updated profile: %s", updated.Name))
}

func (a *App) showRoutingModal(profile *config.Profile) tea.Cmd {
	m := modals.NewRoutingModal(profile, a.width, a.height)
	a.modal = m
	a.modalID = ModalRouting
	return m.Init()
}

func (a *App) handleRoutingResult(routing *config.ProfileRouting) tea.Cmd {
	idx := a.profiles.Selected
	if idx < 0 || idx >= len(a.servers.Profiles) {
		return a.setError(fmt.Errorf("no profile selected"))
	}

	a.servers.Profiles[idx].Routing = *routing
	if err := config.SaveServers(a.servers); err != nil {
		return a.setError(fmt.Errorf("saving routing rules: %w", err))
	}
	a.profiles.SetProfiles(a.servers.Profiles)

	// Regenerate xray config and restart if this is the active profile
	profile := &a.servers.Profiles[idx]
	if profile.Default {
		if err := a.svc.WriteActiveConfig(profile, a.settings); err != nil {
			return a.setError(fmt.Errorf("config generation failed: %w", err))
		}
		if lifecycle.SupervisorAlive() {
			return a.restartXray()
		}
	}
	return a.setMessage(fmt.Sprintf("routing rules updated for %s", profile.Name))
}

func (a *App) showDiffModal(profile *config.Profile) tea.Cmd {
	// Generate new config for selected profile
	newCfg, err := core.GenerateXrayConfig(profile, a.settings)
	if err != nil {
		return a.setError(fmt.Errorf("generating config: %w", err))
	}
	newJSON, _ := json.MarshalIndent(newCfg, "", "  ")
	newLines := strings.Split(string(newJSON), "\n")

	// Read current config
	var oldLines []string
	currentCfg, err := core.ReadXrayConfig()
	if err != nil {
		oldLines = []string{"(no current config)"}
	} else {
		oldJSON, _ := json.MarshalIndent(currentCfg, "", "  ")
		oldLines = strings.Split(string(oldJSON), "\n")
	}

	title := fmt.Sprintf("Config diff: → %s", profile.Name)
	m := modals.NewDiffModal(title, oldLines, newLines, a.width, a.height)
	a.modal = m
	a.modalID = ModalDiff
	return m.Init()
}

func (a *App) showQRModal(profile *config.Profile) tea.Cmd {
	url := core.ToProxyURL(profile)
	m := modals.NewQRModal(profile.Name, url, a.width, a.height)
	a.modal = m
	a.modalID = ModalQR
	return m.Init()
}

func (a *App) showUpdateModal() tea.Cmd {
	m := modals.NewUpdateModal(a.xray, a.settings, a.width, a.height)
	a.modal = m
	a.modalID = ModalUpdate
	return m.Init()
}

func (a *App) showSubscriptionModal() tea.Cmd {
	m := modals.NewSubscriptionModal(a.settings.Subscriptions, a.width, a.height)
	a.modal = m
	a.modalID = ModalSubscription
	return m.Init()
}

// importSubscriptionCmd fetches a subscription, merges its profiles, persists
// servers plus the subscription entry, and reports the outcome via
// subscriptionResultMsg. Shared by the SubscriptionModal and the first-run wizard.
func (a *App) importSubscriptionCmd(subURL, subName string) tea.Cmd {
	loadCmd := a.startLoading("Fetching subscription...")
	return tea.Batch(loadCmd, func() tea.Msg {
		servers, err := config.LoadServers()
		if err != nil {
			return subscriptionResultMsg{err: fmt.Errorf("loading servers: %w", err)}
		}
		added, updated, err := a.svc.ImportSubscription(servers, subURL, subName)
		if err != nil {
			return subscriptionResultMsg{err: err}
		}
		if err := config.SaveServers(servers); err != nil {
			return subscriptionResultMsg{err: fmt.Errorf("saving servers: %w", err)}
		}

		// Save subscription entry to settings
		settings, _ := config.LoadSettings()
		if settings != nil {
			settings.UpsertSubscription(subURL, subName)
			_ = config.SaveSettings(settings)
		}

		return subscriptionResultMsg{added: added, updated: updated}
	})
}

func (a *App) handleSubscriptionResult(m *modals.SubscriptionModal) tea.Cmd {
	switch m.Action {
	case modals.SubActionAdd, modals.SubActionUpdate:
		return a.importSubscriptionCmd(m.SubURL, m.SubName)

	case modals.SubActionDelete:
		if m.DeleteIndex >= 0 && m.DeleteIndex < len(a.settings.Subscriptions) {
			subURL := a.settings.Subscriptions[m.DeleteIndex].URL

			// Remove subscription entry from settings
			a.settings.Subscriptions = append(
				a.settings.Subscriptions[:m.DeleteIndex],
				a.settings.Subscriptions[m.DeleteIndex+1:]...,
			)
			if err := config.SaveSettings(a.settings); err != nil {
				return a.setError(fmt.Errorf("saving settings: %w", err))
			}

			// Remove profiles belonging to this subscription
			var remaining []config.Profile
			removed := 0
			for _, p := range a.servers.Profiles {
				if p.Subscription == subURL {
					removed++
					continue
				}
				remaining = append(remaining, p)
			}
			a.servers.Profiles = remaining
			if a.servers.DefaultProfile() == nil && len(a.servers.Profiles) > 0 {
				a.servers.Profiles[0].Default = true
			}
			if err := config.SaveServers(a.servers); err != nil {
				return a.setError(fmt.Errorf("saving servers: %w", err))
			}
			a.refreshGroups()
			a.applyGroupFilter()
			return a.setMessage(fmt.Sprintf("deleted subscription (%d profiles removed)", removed))
		}
	}
	return nil
}

func (a *App) testAllProfiles() tea.Cmd {
	if len(a.servers.Profiles) == 0 {
		return a.setError(fmt.Errorf("no profiles to test"))
	}

	profiles := make([]config.Profile, len(a.servers.Profiles))
	copy(profiles, a.servers.Profiles)

	loadCmd := a.startLoading("Testing all profiles...")
	settings := a.settings
	return tea.Batch(loadCmd, func() tea.Msg {
		var results []profileTestResult
		for _, p := range profiles {
			r := core.ProbeProfile(p, lifecycle.ProbeContextFor(p.Name, settings))
			switch r.Status {
			case core.LivenessOK:
				results = append(results, profileTestResult{name: p.Name, latency: r.Latency})
			case core.LivenessSkipped:
				results = append(results, profileTestResult{name: p.Name, skipped: true})
			default:
				results = append(results, profileTestResult{name: p.Name, err: r.Err})
			}
		}
		return testAllResultMsg{results: results}
	})
}

func (a *App) applyTestAllResults(results []profileTestResult) {
	for _, r := range results {
		for i := range a.servers.Profiles {
			if a.servers.Profiles[i].Name == r.name {
				switch {
				case r.err != nil:
					a.servers.Profiles[i].Latency = -1
				case r.skipped:
					a.servers.Profiles[i].Latency = -2
				default:
					a.servers.Profiles[i].Latency = r.latency.Milliseconds()
				}
				break
			}
		}
	}

	// Sort: working (asc) → skipped (-2) → untested (0) → failed (-1).
	rank := func(lat int64) int {
		switch {
		case lat > 0:
			return 0
		case lat == -2:
			return 1
		case lat == 0:
			return 2
		default:
			return 3
		}
	}
	sort.SliceStable(a.servers.Profiles, func(i, j int) bool {
		li, lj := a.servers.Profiles[i].Latency, a.servers.Profiles[j].Latency
		ri, rj := rank(li), rank(lj)
		if ri != rj {
			return ri < rj
		}
		if ri == 0 {
			return li < lj
		}
		return false
	})

	// Auto-select fastest: set the first working profile as default
	fastestIdx := -1
	for i := range a.servers.Profiles {
		if a.servers.Profiles[i].Latency > 0 {
			fastestIdx = i
			break
		}
	}
	if fastestIdx >= 0 {
		for i := range a.servers.Profiles {
			a.servers.Profiles[i].Default = (i == fastestIdx)
		}
	}

	_ = config.SaveServers(a.servers)
	a.applyGroupFilter()
}

func (a *App) duplicateSelectedProfile() tea.Cmd {
	profile := a.profiles.SelectedProfile()
	if profile == nil {
		return a.setError(fmt.Errorf("no profile selected"))
	}

	dup := *profile
	dup.Name = profile.Name + " (copy)"
	dup.Default = false
	dup.Latency = 0

	a.servers.Profiles = append(a.servers.Profiles, dup)
	if err := config.SaveServers(a.servers); err != nil {
		return a.setError(fmt.Errorf("saving servers: %w", err))
	}
	a.refreshGroups()
	a.applyGroupFilter()
	return a.setMessage(fmt.Sprintf("duplicated: %s", dup.Name))
}

// refreshGroups rebuilds the list of unique groups from all profiles.
func (a *App) refreshGroups() {
	seen := make(map[string]bool)
	var groups []string
	for _, p := range a.servers.Profiles {
		if p.Group != "" && !seen[p.Group] {
			seen[p.Group] = true
			groups = append(groups, p.Group)
		}
	}
	a.allGroups = groups
}

// cycleGroupFilter cycles through: all → group1 → group2 → ... → all
func (a *App) cycleGroupFilter() {
	a.refreshGroups()
	if len(a.allGroups) == 0 {
		a.groupFilter = ""
		return
	}

	if a.groupFilter == "" {
		a.groupFilter = a.allGroups[0]
		a.applyGroupFilter()
		return
	}

	for i, g := range a.allGroups {
		if g == a.groupFilter {
			if i+1 < len(a.allGroups) {
				a.groupFilter = a.allGroups[i+1]
			} else {
				a.groupFilter = ""
			}
			a.applyGroupFilter()
			return
		}
	}

	a.groupFilter = ""
	a.applyGroupFilter()
}

// applyGroupFilter updates the profiles panel with filtered profiles.
func (a *App) applyGroupFilter() {
	if a.groupFilter == "" {
		a.profiles.SetProfiles(a.servers.Profiles)
		return
	}

	var filtered []config.Profile
	for _, p := range a.servers.Profiles {
		if p.Group == a.groupFilter {
			filtered = append(filtered, p)
		}
	}
	a.profiles.SetProfiles(filtered)
}

func (a *App) groupFilterLabel() string {
	if a.groupFilter == "" {
		return "filter: all profiles"
	}
	return fmt.Sprintf("filter: %s", a.groupFilter)
}

// hasAutoRefreshSubscriptions returns true if any subscription has an interval > 0.
func (a *App) hasAutoRefreshSubscriptions() bool {
	for _, sub := range a.settings.Subscriptions {
		if sub.Interval > 0 && sub.URL != "" {
			return true
		}
	}
	return false
}

// autoRefreshSubscriptions checks each subscription and refreshes those past their interval.
func (a *App) autoRefreshSubscriptions() tea.Cmd {
	subs := make([]config.SubscriptionEntry, len(a.settings.Subscriptions))
	copy(subs, a.settings.Subscriptions)

	return func() tea.Msg {
		var totalAdded, totalUpdated int

		servers, err := config.LoadServers()
		if err != nil {
			return subscriptionAutoRefreshResultMsg{err: fmt.Errorf("loading servers: %w", err)}
		}

		refreshed := false
		for _, sub := range subs {
			if sub.Interval <= 0 || sub.URL == "" {
				continue
			}

			added, updated, err := a.svc.ImportSubscription(servers, sub.URL, sub.Name)
			if err != nil {
				continue
			}
			totalAdded += added
			totalUpdated += updated
			if added > 0 || updated > 0 {
				refreshed = true
			}
		}

		if refreshed {
			_ = config.SaveServers(servers)
		}

		return subscriptionAutoRefreshResultMsg{totalAdded: totalAdded, totalUpdated: totalUpdated}
	}
}

// View renders the TUI.
func (a *App) View() string {
	if a.shuttingDown {
		return "\n  Shutting down...\n"
	}
	if a.width == 0 || a.height == 0 {
		return "Loading..."
	}

	// Calculate layout
	leftWidth, rightWidth, fullHeight, statusHeight, logsHeight := a.calcLayout()

	// Render panels with titles embedded in borders
	profilesView := renderPanelWithTitle(
		panelStyle(a.activePanel == PanelProfiles, leftWidth, fullHeight).Render(a.profiles.View()),
		"Profiles", a.activePanel == PanelProfiles)

	statusView := renderPanelWithTitle(
		panelStyle(a.activePanel == PanelStatus, rightWidth, statusHeight).Render(a.status.View()),
		"Status", a.activePanel == PanelStatus)

	logsView := renderPanelWithTitle(
		panelStyle(a.activePanel == PanelLogs, rightWidth, logsHeight).Render(a.logs.View()),
		a.logs.LogTitle(), a.activePanel == PanelLogs)

	var main string
	if a.isNarrowTerminal() {
		// Stacked single-column layout
		main = lipgloss.JoinVertical(lipgloss.Left, profilesView, statusView, logsView)
	} else {
		// Side-by-side layout
		rightColumn := lipgloss.JoinVertical(lipgloss.Left, statusView, logsView)
		main = lipgloss.JoinHorizontal(lipgloss.Top, profilesView, rightColumn)
	}

	// Hotkeys bar + Status bar
	hotkeysBar := a.renderHotkeysBar()
	statusBar := a.renderStatusBar()
	result := lipgloss.JoinVertical(lipgloss.Left, main, hotkeysBar, statusBar)

	// Modal overlay
	if a.modalID != ModalNone && a.modal != nil {
		modalView := a.modal.View()
		result = a.overlayModal(result, modalView)
	}

	return result
}

func (a *App) renderHotkeysBar() string {
	keyStyle := lipgloss.NewStyle().Foreground(colorOrangeBr).Bold(true)
	descStyle := lipgloss.NewStyle().Foreground(colorGray)

	var parts []string
	for _, c := range a.registry.BarItems(a.isNarrowTerminal()) {
		parts = append(parts, keyStyle.Render(commands.KeyDisplay(c.Binding))+descStyle.Render(":"+c.ShortLabel))
	}

	bar := " " + strings.Join(parts, "  ")
	return lipgloss.NewStyle().Width(a.width).Render(bar)
}

func severityColor(s notify.Severity) lipgloss.Color {
	t := theme.Current()
	switch s {
	case notify.Error:
		return t.Error
	case notify.Warning:
		return t.Warning
	case notify.Success:
		return t.Success
	case notify.Info:
		return t.Info
	}
	return t.Muted
}

func (a *App) renderStatusBar() string {
	// Show spinner during loading
	if a.loading {
		spinnerStyle := lipgloss.NewStyle().
			Foreground(colorAquaBr).
			Padding(0, 1)
		return spinnerStyle.Width(a.width).Render(a.spinner.View() + " " + a.loadingMsg)
	}

	// Show the active notification tail (severity carried by color).
	if a.activeTail != nil {
		n := *a.activeTail
		text := n.Severity.Tag() + ": " + n.Message
		if n.Count > 1 {
			text += fmt.Sprintf(" ×%d", n.Count)
		}
		tailStyle := lipgloss.NewStyle().
			Foreground(severityColor(n.Severity)).
			Bold(true).
			Padding(0, 1)
		line := tailStyle.Width(a.width).Render(text)
		if n.Hint != "" && (n.Severity == notify.Error || n.Severity == notify.Warning) {
			hintLine := lipgloss.NewStyle().
				Foreground(theme.Current().Muted).
				Padding(0, 1).
				Width(a.width).
				Render("  → try: " + n.Hint)
			return lipgloss.JoinVertical(lipgloss.Left, line, hintLine)
		}
		return line
	}

	xrayVer := core.GetXrayVersion()
	profile := ""
	if p := a.servers.DefaultProfile(); p != nil {
		profile = p.Name
	}

	state := "stopped"
	if lifecycle.SupervisorAlive() {
		state = "running"
	}

	bar := fmt.Sprintf("  lazyray %s | xray %s | profile: %s | %s",
		a.version, xrayVer, profile, state)

	if a.availableUpdate != "" {
		updateHint := lipgloss.NewStyle().
			Foreground(colorYellow).
			Bold(true).
			Render(fmt.Sprintf(" | update available: %s [u]", a.availableUpdate))
		bar += updateHint
	}

	if !a.isNarrowTerminal() {
		if u := a.unreadCount(); u > 0 {
			bar += lipgloss.NewStyle().
				Foreground(theme.Current().Info).
				Render(fmt.Sprintf(" | activity: %d new [n]", u))
		}
	}

	return styleStatusBar.Width(a.width).Render(bar)
}

func (a *App) overlayModal(bg, modal string) string {
	bgLines := strings.Split(bg, "\n")
	modalLines := strings.Split(modal, "\n")

	modalWidth := 0
	for _, line := range modalLines {
		w := lipgloss.Width(line)
		if w > modalWidth {
			modalWidth = w
		}
	}

	startRow := (len(bgLines) - len(modalLines)) / 2
	startCol := (a.width - modalWidth) / 2

	if startRow < 0 {
		startRow = 0
	}
	if startCol < 0 {
		startCol = 0
	}

	for i, mLine := range modalLines {
		row := startRow + i
		if row >= len(bgLines) {
			break
		}

		bgLine := bgLines[row]

		// ANSI-safe: truncate background to startCol visual width
		left := ansi.Truncate(bgLine, startCol, "")
		leftW := lipgloss.Width(left)
		if leftW < startCol {
			left += strings.Repeat(" ", startCol-leftW)
		}

		bgLines[row] = left + mLine
	}

	return strings.Join(bgLines, "\n")
}
