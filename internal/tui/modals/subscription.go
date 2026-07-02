package modals

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/rtxnik/lazyray/internal/config"
	"github.com/rtxnik/lazyray/internal/tui/theme"
)

// SubscriptionAction indicates what action the user wants to perform.
type SubscriptionAction int

const (
	SubActionNone SubscriptionAction = iota
	SubActionAdd
	SubActionUpdate
	SubActionDelete
)

// SubscriptionModal manages subscription URLs.
type SubscriptionModal struct {
	subs      []config.SubscriptionEntry
	selected  int
	adding    bool
	urlInput  textinput.Model
	nameInput textinput.Model
	focusURL  bool

	Done        bool
	Action      SubscriptionAction
	SubURL      string
	SubName     string
	DeleteIndex int

	err    string
	width  int
	height int
}

// NewSubscriptionModal creates a subscription management modal.
func NewSubscriptionModal(subs []config.SubscriptionEntry, width, height int) *SubscriptionModal {
	urlInput := textinput.New()
	urlInput.Placeholder = "https://example.com/sub"
	urlInput.Width = 50
	urlInput.CharLimit = 512

	nameInput := textinput.New()
	nameInput.Placeholder = "Subscription name"
	nameInput.Width = 50
	nameInput.CharLimit = 64

	copied := make([]config.SubscriptionEntry, len(subs))
	copy(copied, subs)

	return &SubscriptionModal{
		subs:        copied,
		urlInput:    urlInput,
		nameInput:   nameInput,
		focusURL:    true,
		width:       width,
		height:      height,
		DeleteIndex: -1,
	}
}

func (m *SubscriptionModal) Init() tea.Cmd {
	return nil
}

func (m *SubscriptionModal) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.adding {
			return m.updateAddMode(msg)
		}
		return m.updateListMode(msg)
	}

	// Forward to text inputs if adding
	if m.adding {
		var cmd tea.Cmd
		if m.focusURL {
			m.urlInput, cmd = m.urlInput.Update(msg)
		} else {
			m.nameInput, cmd = m.nameInput.Update(msg)
		}
		return m, cmd
	}

	return m, nil
}

func (m *SubscriptionModal) updateListMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.selected > 0 {
			m.selected--
		}
	case "down", "j":
		if m.selected < len(m.subs)-1 {
			m.selected++
		}
	case "a":
		m.adding = true
		m.focusURL = true
		m.urlInput.SetValue("")
		m.nameInput.SetValue("")
		m.urlInput.Focus()
		m.err = ""
		return m, textinput.Blink
	case "d":
		if len(m.subs) > 0 && m.selected < len(m.subs) {
			m.Action = SubActionDelete
			m.DeleteIndex = m.selected
			m.Done = true
		}
	case "u", "enter":
		if len(m.subs) > 0 && m.selected < len(m.subs) {
			m.Action = SubActionUpdate
			m.SubURL = m.subs[m.selected].URL
			m.SubName = m.subs[m.selected].Name
			m.Done = true
		}
	case "esc":
		m.Done = true
	}
	return m, nil
}

func (m *SubscriptionModal) updateAddMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		if m.focusURL {
			m.focusURL = false
			m.urlInput.Blur()
			m.nameInput.Focus()
			return m, textinput.Blink
		}
		url := strings.TrimSpace(m.urlInput.Value())
		if url == "" {
			m.err = "URL is required"
			return m, nil
		}
		name := strings.TrimSpace(m.nameInput.Value())
		if name == "" {
			name = "subscription"
		}
		m.Action = SubActionAdd
		m.SubURL = url
		m.SubName = name
		m.Done = true
		return m, nil
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
		m.adding = false
		m.err = ""
		return m, nil
	}

	var cmd tea.Cmd
	if m.focusURL {
		m.urlInput, cmd = m.urlInput.Update(msg)
	} else {
		m.nameInput, cmd = m.nameInput.Update(msg)
	}
	return m, cmd
}

func (m *SubscriptionModal) View() string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.Current().Accent).
		MarginBottom(1)

	modalWidth := 60
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
	b.WriteString(titleStyle.Render("Subscriptions"))
	b.WriteString("\n\n")

	if m.adding {
		b.WriteString("Subscription URL:\n")
		b.WriteString(m.urlInput.View())
		b.WriteString("\n\nName:\n")
		b.WriteString(m.nameInput.View())

		if m.err != "" {
			errStyle := lipgloss.NewStyle().Foreground(theme.Current().Error)
			b.WriteString("\n\n")
			b.WriteString(errStyle.Render(m.err))
		}

		b.WriteString("\n\n")
		hint := lipgloss.NewStyle().Foreground(theme.Current().Muted)
		b.WriteString(hint.Render("[Enter] Confirm  [Tab] Switch  [Esc] Back"))
	} else {
		if len(m.subs) == 0 {
			dim := lipgloss.NewStyle().Foreground(theme.Current().Muted)
			b.WriteString(dim.Render("No subscriptions yet. Add one with [a]."))
			b.WriteString("\n")
		} else {
			selectedStyle := lipgloss.NewStyle().
				Foreground(theme.Current().Accent).
				Bold(true)
			normalStyle := lipgloss.NewStyle().
				Foreground(theme.Current().Fg)
			dimStyle := lipgloss.NewStyle().
				Foreground(theme.Current().Muted)

			for i, sub := range m.subs {
				prefix := "  "
				style := normalStyle
				if i == m.selected {
					prefix = "> "
					style = selectedStyle
				}
				name := sub.Name
				if name == "" {
					name = "unnamed"
				}
				b.WriteString(style.Render(fmt.Sprintf("%s%s", prefix, name)))
				b.WriteString("\n")
				b.WriteString(dimStyle.Render(fmt.Sprintf("    %s", sub.URL)))
				if i < len(m.subs)-1 {
					b.WriteString("\n")
				}
			}
		}

		b.WriteString("\n\n")
		hint := lipgloss.NewStyle().Foreground(theme.Current().Muted)
		b.WriteString(hint.Render("[a] Add  [u/Enter] Update  [d] Delete  [Esc] Close"))
	}

	return modal.Render(b.String())
}
