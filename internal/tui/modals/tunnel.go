package modals

import (
	"fmt"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/rtxnik/lazyray/internal/config"
	"github.com/rtxnik/lazyray/internal/core"
	"github.com/rtxnik/lazyray/internal/tui/theme"
)

// TunnelModal manages SSH tunnel connections.
type TunnelModal struct {
	servers  *config.ServersConfig
	tunnels  *core.TunnelManager
	statuses []core.TunnelStatus
	selected int
	Done     bool
	err      string
	width    int
	height   int
}

// NewTunnelModal creates a new tunnel modal.
func NewTunnelModal(servers *config.ServersConfig, tunnels *core.TunnelManager, width, height int) *TunnelModal {
	m := &TunnelModal{
		servers: servers,
		tunnels: tunnels,
		width:   width,
		height:  height,
	}
	m.refreshStatuses()
	return m
}

func (m *TunnelModal) refreshStatuses() {
	m.statuses = m.tunnels.Status(m.servers.Profiles)
}

func (m *TunnelModal) Init() tea.Cmd { return nil }

func (m *TunnelModal) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.KeyMsg); ok {
		m.err = ""
		switch msg.String() {
		case "esc":
			m.Done = true
			return m, nil
		case "up", "k":
			if m.selected > 0 {
				m.selected--
			}
		case "down", "j":
			if m.selected < len(m.statuses)-1 {
				m.selected++
			}
		case "enter":
			m.toggleIndex(m.selected)
		default:
			// Number keys 1-9 for quick toggle
			if n, err := strconv.Atoi(msg.String()); err == nil && n >= 1 && n <= len(m.statuses) {
				m.toggleIndex(n - 1)
			}
		}
	}
	return m, nil
}

func (m *TunnelModal) toggleIndex(idx int) {
	if idx < 0 || idx >= len(m.statuses) {
		return
	}

	s := m.statuses[idx]
	if s.Connected {
		if err := m.tunnels.Disconnect(s.Name); err != nil {
			m.err = err.Error()
		}
	} else {
		for i := range m.servers.Profiles {
			if m.servers.Profiles[i].Name == s.Name {
				if err := m.tunnels.Connect(&m.servers.Profiles[i]); err != nil {
					m.err = err.Error()
				}
				break
			}
		}
	}
	m.refreshStatuses()
}

func (m *TunnelModal) findProfile(name string) *config.Profile {
	for i := range m.servers.Profiles {
		if m.servers.Profiles[i].Name == name {
			return &m.servers.Profiles[i]
		}
	}
	return nil
}

func (m *TunnelModal) View() string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.Current().Accent).
		MarginBottom(1)

	modalWidth := 58
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

	ok := lipgloss.NewStyle().Foreground(theme.Current().Success).Bold(true)
	dim := lipgloss.NewStyle().Foreground(theme.Current().Muted)
	sel := lipgloss.NewStyle().Foreground(theme.Current().Accent).Bold(true)
	keyStyle := lipgloss.NewStyle().Foreground(theme.Current().Accent).Bold(true)

	var b strings.Builder
	b.WriteString(titleStyle.Render("SSH Tunnels"))
	b.WriteString("\n\n")

	if len(m.statuses) == 0 {
		b.WriteString(dim.Render("No SSH-capable profiles configured."))
		b.WriteString("\n")
	} else {
		for i, s := range m.statuses {
			marker := "  "
			nameStyle := dim
			if i == m.selected {
				marker = "> "
				nameStyle = sel
			}

			// Status indicator
			var state string
			if s.Connected {
				state = ok.Render("Connected")
			} else {
				state = dim.Render("Disconnected")
			}

			// Number key shortcut
			numKey := keyStyle.Render(fmt.Sprintf("[%d]", i+1))

			// Action hint
			action := "Connect"
			if s.Connected {
				action = "Disconnect"
			}

			b.WriteString(fmt.Sprintf("%s%s  %s  %s %s\n",
				marker,
				nameStyle.Render(s.Name),
				state,
				numKey,
				action))

			// Show SSH command
			profile := m.findProfile(s.Name)
			if profile != nil && profile.SSH.Host != "" {
				sshCmd := fmt.Sprintf("ssh -p %d %s@%s",
					profile.SSH.Port, profile.SSH.User, profile.SSH.Host)
				b.WriteString(fmt.Sprintf("    %s\n", dim.Render(sshCmd)))
			}

			// Show panel URL when connected
			if s.Connected && s.PanelURL != "" {
				b.WriteString(fmt.Sprintf("    %s\n", ok.Render(s.PanelURL)))
			}
			b.WriteString("\n")
		}
	}

	if m.err != "" {
		errStyle := lipgloss.NewStyle().Foreground(theme.Current().Error)
		b.WriteString(errStyle.Render(m.err))
		b.WriteString("\n\n")
	}

	b.WriteString(dim.Render("[Up/Down] Navigate  [Enter/1-9] Toggle  [Esc] Close"))

	return modal.Render(b.String())
}
