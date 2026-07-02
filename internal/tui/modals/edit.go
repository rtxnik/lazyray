package modals

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/rtxnik/lazyray/internal/config"
	"github.com/rtxnik/lazyray/internal/tui/theme"
)

const (
	fieldName = iota
	fieldAddress
	fieldPort
	fieldUUID
	fieldEncryption
	fieldFlow
	fieldNetwork
	fieldPath
	fieldMode
	fieldHost
	fieldSecurityType
	fieldSNI
	fieldFingerprint
	fieldPublicKey
	fieldShortID
	fieldSpiderX
	fieldCount
)

var fieldLabels = [fieldCount]string{
	"Name", "Address", "Port", "UUID",
	"Encryption", "Flow", "Network", "Path",
	"Mode", "Host", "Security", "SNI",
	"Fingerprint", "Public Key", "Short ID", "SpiderX",
}

// EditModal handles profile editing.
type EditModal struct {
	inputs     [fieldCount]textinput.Model
	focusIndex int
	scrollTop  int
	Done       bool
	Profile    *config.Profile
	err        string
	width      int
	height     int
}

// NewEditModal creates an edit modal pre-filled with profile data.
func NewEditModal(profile *config.Profile, width, height int) *EditModal {
	m := &EditModal{
		width:  width,
		height: height,
	}

	for i := 0; i < fieldCount; i++ {
		ti := textinput.New()
		ti.Prompt = ""
		ti.CharLimit = 256
		ti.Width = 40
		m.inputs[i] = ti
	}

	// Pre-fill from profile
	m.inputs[fieldName].SetValue(profile.Name)
	m.inputs[fieldAddress].SetValue(profile.Server.Address)
	m.inputs[fieldPort].SetValue(strconv.Itoa(profile.Server.Port))
	m.inputs[fieldUUID].SetValue(profile.Server.UUID)
	m.inputs[fieldEncryption].SetValue(profile.Server.Encryption)
	m.inputs[fieldFlow].SetValue(profile.Server.Flow)
	m.inputs[fieldNetwork].SetValue(profile.Server.Transport.Network)
	m.inputs[fieldPath].SetValue(profile.Server.Transport.Path)
	m.inputs[fieldMode].SetValue(profile.Server.Transport.Mode)
	m.inputs[fieldHost].SetValue(profile.Server.Transport.Host)
	m.inputs[fieldSecurityType].SetValue(profile.Server.Security.Type)
	m.inputs[fieldSNI].SetValue(profile.Server.Security.SNI)
	m.inputs[fieldFingerprint].SetValue(profile.Server.Security.Fingerprint)
	m.inputs[fieldPublicKey].SetValue(profile.Server.Security.PublicKey)
	m.inputs[fieldShortID].SetValue(profile.Server.Security.ShortID)
	m.inputs[fieldSpiderX].SetValue(profile.Server.Security.SpiderX)

	// Placeholders for commonly empty fields
	m.inputs[fieldEncryption].Placeholder = "none"
	m.inputs[fieldFlow].Placeholder = "empty for XHTTP"
	m.inputs[fieldHost].Placeholder = "optional"
	m.inputs[fieldSpiderX].Placeholder = "/"

	m.inputs[m.focusIndex].Focus()
	return m
}

func (m *EditModal) Init() tea.Cmd {
	return textinput.Blink
}

func (m *EditModal) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			if m.focusIndex == fieldCount-1 {
				return m, m.submit()
			}
			m.moveFocus(1)
			return m, textinput.Blink

		case "tab", "down":
			m.moveFocus(1)
			return m, textinput.Blink

		case "shift+tab", "up":
			m.moveFocus(-1)
			return m, textinput.Blink

		case "ctrl+s":
			return m, m.submit()

		case "esc":
			m.Done = true
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.inputs[m.focusIndex], cmd = m.inputs[m.focusIndex].Update(msg)
	return m, cmd
}

func (m *EditModal) moveFocus(delta int) {
	m.inputs[m.focusIndex].Blur()
	m.focusIndex = (m.focusIndex + delta + fieldCount) % fieldCount
	m.inputs[m.focusIndex].Focus()
	m.ensureVisible()
}

func (m *EditModal) ensureVisible() {
	visibleRows := m.visibleRows()
	if visibleRows <= 0 {
		return
	}
	if m.focusIndex < m.scrollTop {
		m.scrollTop = m.focusIndex
	}
	if m.focusIndex >= m.scrollTop+visibleRows {
		m.scrollTop = m.focusIndex - visibleRows + 1
	}
}

func (m *EditModal) visibleRows() int {
	// Modal inner height minus: title(2) + error(2 max) + hint(2) + padding
	rows := m.modalHeight() - 6
	if rows < 4 {
		rows = 4
	}
	return rows
}

func (m *EditModal) modalHeight() int {
	h := m.height - 6
	if h > fieldCount*2+10 {
		h = fieldCount*2 + 10
	}
	if h < 16 {
		h = 16
	}
	return h
}

func (m *EditModal) submit() tea.Cmd {
	return func() tea.Msg {
		name := strings.TrimSpace(m.inputs[fieldName].Value())
		if name == "" {
			m.err = "Name is required"
			return nil
		}
		addr := strings.TrimSpace(m.inputs[fieldAddress].Value())
		if addr == "" {
			m.err = "Address is required"
			return nil
		}
		portStr := strings.TrimSpace(m.inputs[fieldPort].Value())
		port, err := strconv.Atoi(portStr)
		if err != nil || port < 1 || port > 65535 {
			m.err = "Port must be 1-65535"
			return nil
		}
		uuid := strings.TrimSpace(m.inputs[fieldUUID].Value())
		if uuid == "" {
			m.err = "UUID is required"
			return nil
		}

		m.Profile = &config.Profile{
			Name: name,
			Server: config.ServerConfig{
				Address:    addr,
				Port:       port,
				UUID:       uuid,
				Encryption: strings.TrimSpace(m.inputs[fieldEncryption].Value()),
				Flow:       strings.TrimSpace(m.inputs[fieldFlow].Value()),
				Transport: config.TransportConfig{
					Network: strings.TrimSpace(m.inputs[fieldNetwork].Value()),
					Path:    strings.TrimSpace(m.inputs[fieldPath].Value()),
					Mode:    strings.TrimSpace(m.inputs[fieldMode].Value()),
					Host:    strings.TrimSpace(m.inputs[fieldHost].Value()),
				},
				Security: config.SecurityConfig{
					Type:        strings.TrimSpace(m.inputs[fieldSecurityType].Value()),
					SNI:         strings.TrimSpace(m.inputs[fieldSNI].Value()),
					Fingerprint: strings.TrimSpace(m.inputs[fieldFingerprint].Value()),
					PublicKey:   strings.TrimSpace(m.inputs[fieldPublicKey].Value()),
					ShortID:     strings.TrimSpace(m.inputs[fieldShortID].Value()),
					SpiderX:     strings.TrimSpace(m.inputs[fieldSpiderX].Value()),
				},
			},
		}
		m.Done = true
		return nil
	}
}

func (m *EditModal) View() string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.Current().Accent).
		MarginBottom(1)

	labelStyle := lipgloss.NewStyle().
		Foreground(theme.Current().Selected).
		Width(13)
	focusedLabelStyle := labelStyle.
		Foreground(theme.Current().Orange).
		Bold(true)

	modalWidth := 64
	if m.width > 0 && m.width-4 < modalWidth {
		modalWidth = m.width - 4
	}
	if modalWidth < 36 {
		modalWidth = 36
	}

	inputWidth := modalWidth - 20
	if inputWidth < 16 {
		inputWidth = 16
	}
	for i := range m.inputs {
		m.inputs[i].Width = inputWidth
	}

	modal := lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(theme.Current().Accent).
		Padding(1, 2).
		Width(modalWidth)

	var b strings.Builder
	b.WriteString(titleStyle.Render("Edit Profile"))
	b.WriteString("\n\n")

	visibleRows := m.visibleRows()
	end := m.scrollTop + visibleRows
	if end > fieldCount {
		end = fieldCount
	}

	// Scroll indicator top
	if m.scrollTop > 0 {
		dim := lipgloss.NewStyle().Foreground(theme.Current().Muted)
		b.WriteString(dim.Render(fmt.Sprintf("  ▲ %d more above", m.scrollTop)))
		b.WriteString("\n")
	}

	for i := m.scrollTop; i < end; i++ {
		ls := labelStyle
		if i == m.focusIndex {
			ls = focusedLabelStyle
		}
		b.WriteString(ls.Render(fieldLabels[i]+":") + " " + m.inputs[i].View())
		if i < end-1 {
			b.WriteString("\n")
		}
	}

	// Scroll indicator bottom
	if end < fieldCount {
		dim := lipgloss.NewStyle().Foreground(theme.Current().Muted)
		b.WriteString("\n")
		b.WriteString(dim.Render(fmt.Sprintf("  ▼ %d more below", fieldCount-end)))
	}

	if m.err != "" {
		errStyle := lipgloss.NewStyle().Foreground(theme.Current().Error)
		b.WriteString("\n\n")
		b.WriteString(errStyle.Render(m.err))
	}

	b.WriteString("\n\n")
	hint := lipgloss.NewStyle().Foreground(theme.Current().Muted)
	b.WriteString(hint.Render("[Tab/↑↓] Navigate  [Ctrl+S/Enter] Save  [Esc] Cancel"))

	return modal.Render(b.String())
}
