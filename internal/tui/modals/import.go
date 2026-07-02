package modals

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/rtxnik/lazyray/internal/config"
	"github.com/rtxnik/lazyray/internal/core"
	"github.com/rtxnik/lazyray/internal/tui/theme"
)

// ImportModal handles VLESS URL import.
type ImportModal struct {
	urlInput  textinput.Model
	nameInput textinput.Model
	focusURL  bool
	Done      bool
	Profile   *config.Profile
	err       string
	width     int
	height    int
}

// NewImportModal creates a new import modal.
func NewImportModal(width, height int) *ImportModal {
	urlInput := textinput.New()
	urlInput.Placeholder = "vless://, vmess://, or trojan://..."
	urlInput.Focus()
	urlInput.Width = 50
	urlInput.CharLimit = 1024

	nameInput := textinput.New()
	nameInput.Placeholder = "Profile name (optional)"
	nameInput.Width = 50
	nameInput.CharLimit = 64

	return &ImportModal{
		urlInput:  urlInput,
		nameInput: nameInput,
		focusURL:  true,
		width:     width,
		height:    height,
	}
}

func (m *ImportModal) Init() tea.Cmd {
	return textinput.Blink
}

func (m *ImportModal) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			if m.focusURL {
				// Move to name input
				m.focusURL = false
				m.urlInput.Blur()
				m.nameInput.Focus()
				return m, textinput.Blink
			}
			// Submit
			return m, m.submit()

		case "tab":
			m.focusURL = !m.focusURL
			if m.focusURL {
				m.nameInput.Blur()
				m.urlInput.Focus()
			} else {
				m.urlInput.Blur()
				m.nameInput.Focus()
			}
			return m, textinput.Blink

		case "esc":
			m.Done = true
			return m, nil
		}
	}

	var cmd tea.Cmd
	if m.focusURL {
		m.urlInput, cmd = m.urlInput.Update(msg)
	} else {
		m.nameInput, cmd = m.nameInput.Update(msg)
	}
	return m, cmd
}

func (m *ImportModal) submit() tea.Cmd {
	return func() tea.Msg {
		url := strings.TrimSpace(m.urlInput.Value())
		if url == "" {
			m.err = "URL is required"
			return nil
		}

		profile, err := core.ParseProxyURL(url)
		if err != nil {
			m.err = err.Error()
			return nil
		}

		name := strings.TrimSpace(m.nameInput.Value())
		if name != "" {
			profile.Name = name
		}

		m.Profile = profile
		m.Done = true
		return nil
	}
}

func (m *ImportModal) View() string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.Current().Accent).
		MarginBottom(1)

	// Responsive modal width: fit within terminal, min 30, max 60
	modalWidth := 60
	if m.width > 0 && m.width-4 < modalWidth {
		modalWidth = m.width - 4
	}
	if modalWidth < 30 {
		modalWidth = 30
	}

	// Adjust input widths to fit inside modal (accounting for padding + border)
	inputWidth := modalWidth - 6
	if inputWidth < 16 {
		inputWidth = 16
	}
	m.urlInput.Width = inputWidth
	m.nameInput.Width = inputWidth

	modal := lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(theme.Current().Accent).
		Padding(1, 2).
		Width(modalWidth)

	var b strings.Builder
	b.WriteString(titleStyle.Render("Import Configuration"))
	b.WriteString("\n\n")
	b.WriteString("Paste proxy URL:\n")
	b.WriteString(m.urlInput.View())
	b.WriteString("\n\n")
	b.WriteString("Profile name:\n")
	b.WriteString(m.nameInput.View())

	if m.err != "" {
		errStyle := lipgloss.NewStyle().Foreground(theme.Current().Error)
		b.WriteString("\n\n")
		b.WriteString(errStyle.Render(m.err))
	}

	b.WriteString("\n\n")
	hint := lipgloss.NewStyle().Foreground(theme.Current().Muted)
	b.WriteString(hint.Render("[Enter] Import  [Tab] Switch  [Esc] Cancel"))

	return modal.Render(b.String())
}
