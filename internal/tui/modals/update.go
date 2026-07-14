package modals

import (
	"fmt"
	"runtime"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/rtxnik/lazyray/internal/config"
	"github.com/rtxnik/lazyray/internal/core"
	"github.com/rtxnik/lazyray/internal/tui/theme"
)

type updateCheckMsg struct {
	release *core.ReleaseInfo
	err     error
}

type updateApplyMsg struct {
	err error
}

// updateApplier is the app-service seam the modal drives (satisfied by
// *app.Service); kept as an interface so modals never imports internal/app.
type updateApplier interface {
	ApplyXrayUpdate(xray *core.XrayProcess, release *core.ReleaseInfo, url string,
		s *config.Settings, allowUnverified, allowDowngrade bool) error
}

// UpdateModal handles xray-core updates.
type UpdateModal struct {
	xray     *core.XrayProcess
	settings *config.Settings
	svc      updateApplier
	current  string
	release  *core.ReleaseInfo
	checking bool
	applying bool
	done     bool
	result   string
	err      string
	Done     bool
	width    int
	height   int
}

// NewUpdateModal creates a new update modal.
func NewUpdateModal(xray *core.XrayProcess, settings *config.Settings, width, height int, svc updateApplier) *UpdateModal {
	return &UpdateModal{
		xray:     xray,
		settings: settings,
		svc:      svc,
		current:  core.GetXrayVersion(),
		checking: true,
		width:    width,
		height:   height,
	}
}

func (m *UpdateModal) Init() tea.Cmd {
	return m.checkUpdate()
}

func (m *UpdateModal) checkUpdate() tea.Cmd {
	return func() tea.Msg {
		release, err := core.CheckUpdate(m.xrayVersion())
		return updateCheckMsg{release: release, err: err}
	}
}

// xrayVersion returns the pinned xray-core release tag to act on, falling back
// to the default when settings are missing or the version is unset.
func (m *UpdateModal) xrayVersion() string {
	if m.settings != nil && m.settings.Update.XrayVersion != "" {
		return m.settings.Update.XrayVersion
	}
	return config.DefaultSettings().Update.XrayVersion
}

func (m *UpdateModal) applyUpdate() tea.Cmd {
	return func() tea.Msg {
		url, err := core.FindAssetURL(m.release)
		if err != nil {
			return updateApplyMsg{err: err}
		}

		if err := m.svc.ApplyXrayUpdate(m.xray, m.release, url, m.settings, false, false); err != nil {
			return updateApplyMsg{err: err}
		}

		return updateApplyMsg{}
	}
}

func (m *UpdateModal) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case updateCheckMsg:
		m.checking = false
		if msg.err != nil {
			m.err = msg.err.Error()
		} else {
			m.release = msg.release
		}
		return m, nil

	case updateApplyMsg:
		m.applying = false
		if msg.err != nil {
			m.err = msg.err.Error()
		} else {
			m.result = fmt.Sprintf("Updated to %s", m.release.TagName)
			m.done = true
		}
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			if m.done || m.err != "" {
				m.Done = true
				return m, nil
			}
			if m.release != nil && !m.applying {
				m.applying = true
				return m, m.applyUpdate()
			}
		case "esc":
			m.Done = true
			return m, nil
		}
	}

	return m, nil
}

func (m *UpdateModal) View() string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.Current().Accent).
		MarginBottom(1)

	modalWidth := 55
	if m.width > 0 && m.width-4 < modalWidth {
		modalWidth = m.width - 4
	}
	if modalWidth < 30 {
		modalWidth = 30
	}

	modal := lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(theme.Current().Accent).
		Padding(1, 2).
		Width(modalWidth)

	var b strings.Builder
	b.WriteString(titleStyle.Render("Update Xray Core"))
	b.WriteString("\n\n")

	label := lipgloss.NewStyle().Foreground(theme.Current().Muted)

	b.WriteString(fmt.Sprintf("  %s   %s\n", label.Render("Current:"), m.current))
	b.WriteString(fmt.Sprintf("  %s  %s/%s\n", label.Render("Platform:"), runtime.GOOS, runtime.GOARCH))
	b.WriteString(fmt.Sprintf("  %s     %s\n", label.Render("Asset:"), core.AssetName()))

	if m.checking {
		b.WriteString("\nChecking for updates...")
	} else if m.err != "" {
		errStyle := lipgloss.NewStyle().Foreground(theme.Current().Error)
		b.WriteString(fmt.Sprintf("\n%s\n", errStyle.Render(m.err)))
	} else if m.release != nil {
		b.WriteString(fmt.Sprintf("%s    %s\n", label.Render("Pinned:"), m.release.TagName))

		if m.applying {
			b.WriteString("\nDownloading and installing...")
		} else if m.done {
			ok := lipgloss.NewStyle().Foreground(theme.Current().Success).Bold(true)
			b.WriteString(fmt.Sprintf("\n%s\n", ok.Render(m.result)))
		} else {
			warn := lipgloss.NewStyle().Foreground(theme.Current().Selected)
			b.WriteString(fmt.Sprintf("\n%s\n", warn.Render("Xray will be restarted during update")))
		}
	}

	b.WriteString("\n")
	hint := lipgloss.NewStyle().Foreground(theme.Current().Muted)
	if m.done || m.err != "" {
		b.WriteString(hint.Render("[Enter/Esc] Close"))
	} else if m.release != nil && !m.applying {
		b.WriteString(hint.Render("[Enter] Update  [Esc] Cancel"))
	} else {
		b.WriteString(hint.Render("[Esc] Cancel"))
	}

	return modal.Render(b.String())
}
