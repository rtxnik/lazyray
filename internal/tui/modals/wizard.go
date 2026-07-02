package modals

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/rtxnik/lazyray/internal/config"
	"github.com/rtxnik/lazyray/internal/core"
	"github.com/rtxnik/lazyray/internal/tui/theme"
)

// WizardStep represents the current step of the onboarding wizard.
type WizardStep int

const (
	WizardStepMethod  WizardStep = iota // welcome value statement + real method menu
	WizardStepURL                       // paste a single proxy URL
	WizardStepName                      // confirm / edit the profile name
	WizardStepSubURL                    // paste a subscription URL
	WizardStepSubName                   // name the subscription
	WizardStepDone                      // "what next" — one passive nudge
)

const welcomeStatement = "lazyray is a calm, keyboard-first manager for your proxy connections. " +
	"Import a profile or add a subscription to get started."

// WizardModal is the first-launch onboarding wizard.
type WizardModal struct {
	step         WizardStep
	method       wizardMethod
	urlInput     textinput.Model
	nameInput    textinput.Model
	subURLInput  textinput.Model
	subNameInput textinput.Model
	err          string

	// Result — consumed by the app once Done is true.
	Done    bool
	Skipped bool
	Profile *config.Profile // set on the URL path
	SubURL  string          // set on the subscription path
	SubName string

	// StartKey is the display key for "start the proxy" (e.g. "s"), injected by
	// the app via commands.KeyDisplay so the Done nudge stays rebind-safe.
	StartKey string

	width  int
	height int
}

// NewWizardModal creates a new onboarding wizard.
func NewWizardModal(width, height int) *WizardModal {
	urlInput := textinput.New()
	urlInput.Placeholder = "vless://, vmess://, trojan://, or ss://..."
	urlInput.Width = 50
	urlInput.CharLimit = 1024

	nameInput := textinput.New()
	nameInput.Placeholder = "Profile name (optional, uses URL remark)"
	nameInput.Width = 50
	nameInput.CharLimit = 64

	subURLInput := textinput.New()
	subURLInput.Placeholder = "https://example.com/subscription"
	subURLInput.Width = 50
	subURLInput.CharLimit = 1024

	subNameInput := textinput.New()
	subNameInput.Placeholder = "Subscription name"
	subNameInput.Width = 50
	subNameInput.CharLimit = 64

	return &WizardModal{
		step:         WizardStepMethod,
		urlInput:     urlInput,
		nameInput:    nameInput,
		subURLInput:  subURLInput,
		subNameInput: subNameInput,
		StartKey:     "s",
		width:        width,
		height:       height,
	}
}

func (m *WizardModal) Init() tea.Cmd { return nil }

func (m *WizardModal) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// The Done card has already captured the result; any key closes it,
		// committing (never skipping).
		if m.step == WizardStepDone {
			switch msg.String() {
			case "enter", "esc", "q":
				m.Done = true
			}
			return m, nil
		}

		switch msg.String() {
		case "q":
			m.Done = true
			m.Skipped = true
			return m, nil
		case "esc":
			return m.back()
		}

		switch m.step {
		case WizardStepMethod:
			return m.updateMethod(msg)
		case WizardStepURL:
			return m.updateURL(msg)
		case WizardStepName:
			return m.updateName(msg)
		case WizardStepSubURL:
			return m.updateSubURL(msg)
		case WizardStepSubName:
			return m.updateSubName(msg)
		}
	}

	// Forward to the active text input.
	var cmd tea.Cmd
	switch m.step {
	case WizardStepURL:
		m.urlInput, cmd = m.urlInput.Update(msg)
	case WizardStepName:
		m.nameInput, cmd = m.nameInput.Update(msg)
	case WizardStepSubURL:
		m.subURLInput, cmd = m.subURLInput.Update(msg)
	case WizardStepSubName:
		m.subNameInput, cmd = m.subNameInput.Update(msg)
	}
	return m, cmd
}

// back implements esc: step back one, or skip from the entry (Method) step.
func (m *WizardModal) back() (tea.Model, tea.Cmd) {
	switch m.step {
	case WizardStepMethod:
		m.Done = true
		m.Skipped = true
	case WizardStepURL:
		m.step = WizardStepMethod
		m.urlInput.Blur()
		m.err = ""
	case WizardStepName:
		m.step = WizardStepURL
		m.nameInput.Blur()
		m.urlInput.Focus()
		m.err = ""
	case WizardStepSubURL:
		m.step = WizardStepMethod
		m.subURLInput.Blur()
		m.err = ""
	case WizardStepSubName:
		m.step = WizardStepSubURL
		m.subNameInput.Blur()
		m.subURLInput.Focus()
		m.err = ""
	}
	return m, nil
}

func (m *WizardModal) updateMethod(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.method > methodURL {
			m.method--
		}
	case "down", "j":
		if m.method < methodSubscription {
			m.method++
		}
	case "1":
		m.method = methodURL
		return m.pickMethod()
	case "2":
		m.method = methodSubscription
		return m.pickMethod()
	case "enter":
		return m.pickMethod()
	}
	return m, nil
}

func (m *WizardModal) pickMethod() (tea.Model, tea.Cmd) {
	switch m.method {
	case methodSubscription:
		m.step = WizardStepSubURL
		m.subURLInput.Focus()
	default:
		m.step = WizardStepURL
		m.urlInput.Focus()
	}
	return m, textinput.Blink
}

func (m *WizardModal) updateURL(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "enter" {
		url := strings.TrimSpace(m.urlInput.Value())
		if url == "" {
			m.err = "URL is required"
			return m, nil
		}
		profile, err := core.ParseProxyURL(url)
		if err != nil {
			m.err = err.Error()
			return m, nil
		}
		m.Profile = profile
		m.err = ""
		m.step = WizardStepName
		m.urlInput.Blur()
		m.nameInput.Focus()
		if profile.Name != "" {
			m.nameInput.SetValue(profile.Name)
		}
		return m, textinput.Blink
	}
	return m, nil
}

func (m *WizardModal) updateName(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "enter" {
		name := strings.TrimSpace(m.nameInput.Value())
		if name != "" {
			m.Profile.Name = name
		}
		if m.Profile.Name == "" {
			m.Profile.Name = fmt.Sprintf("%s:%d", m.Profile.Server.Address, m.Profile.Server.Port)
		}
		m.Profile.Default = true
		m.method = methodURL
		m.step = WizardStepDone // visible "what next" card; Done set on the next key
		return m, nil
	}
	return m, nil
}

func (m *WizardModal) updateSubURL(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "enter" {
		url := strings.TrimSpace(m.subURLInput.Value())
		if url == "" {
			m.err = "Subscription URL is required"
			return m, nil
		}
		if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
			m.err = "Subscription URL must start with http:// or https://"
			return m, nil
		}
		m.SubURL = url
		m.err = ""
		m.step = WizardStepSubName
		m.subURLInput.Blur()
		m.subNameInput.Focus()
		return m, textinput.Blink
	}
	return m, nil
}

func (m *WizardModal) updateSubName(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "enter" {
		name := strings.TrimSpace(m.subNameInput.Value())
		if name == "" {
			name = "subscription"
		}
		m.SubName = name
		m.method = methodSubscription
		m.step = WizardStepDone // visible "what next" card; Done set on the next key
		return m, nil
	}
	return m, nil
}

func (m *WizardModal) View() string {
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

	inputWidth := modalWidth - 6
	if inputWidth < 16 {
		inputWidth = 16
	}
	m.urlInput.Width = inputWidth
	m.nameInput.Width = inputWidth
	m.subURLInput.Width = inputWidth
	m.subNameInput.Width = inputWidth

	modal := lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(theme.Current().Accent).
		Padding(1, 2).
		Width(modalWidth)

	hint := lipgloss.NewStyle().Foreground(theme.Current().Muted)
	muted := lipgloss.NewStyle().Foreground(theme.Current().Muted)
	selected := lipgloss.NewStyle().Foreground(theme.Current().Selected).Bold(true)
	normal := lipgloss.NewStyle().Foreground(theme.Current().Fg)
	wrapped := lipgloss.NewStyle().Foreground(theme.Current().Fg).Width(inputWidth)
	okStyle := lipgloss.NewStyle().Foreground(theme.Current().Success)
	errStyle := lipgloss.NewStyle().Foreground(theme.Current().Error)

	var b strings.Builder
	b.WriteString(titleStyle.Render("Welcome to lazyray"))
	b.WriteString("\n\n")

	switch m.step {
	case WizardStepMethod:
		b.WriteString(wrapped.Render(welcomeStatement))
		b.WriteString("\n\n")
		b.WriteString("How do you want to add your first connection?\n\n")
		options := []struct{ title, desc string }{
			{"Paste a proxy URL", "vless · vmess · trojan · ss · hysteria2"},
			{"Import a subscription", "one URL -> many servers, auto-updates"},
		}
		for i, opt := range options {
			prefix := "  "
			style := normal
			if wizardMethod(i) == m.method {
				prefix = "> "
				style = selected
			}
			b.WriteString(style.Render(prefix + opt.title))
			b.WriteString("\n")
			b.WriteString(muted.Render("    " + opt.desc))
			b.WriteString("\n")
		}
		b.WriteString("\n")
		b.WriteString(hint.Render("[up/down] move  [enter] select  [q] skip"))

	case WizardStepURL:
		b.WriteString("Paste your proxy URL:\n")
		b.WriteString(m.urlInput.View())
		if m.err != "" {
			b.WriteString("\n\n")
			b.WriteString(errStyle.Render(m.err))
		}
		b.WriteString("\n\n")
		b.WriteString(hint.Render("[enter] parse  [esc] back  [q] skip"))

	case WizardStepName:
		proto := m.Profile.Server.GetProtocol()
		addr := fmt.Sprintf("%s:%d", m.Profile.Server.Address, m.Profile.Server.Port)
		b.WriteString(okStyle.Render("URL parsed successfully"))
		b.WriteString("\n")
		b.WriteString(fmt.Sprintf("  Protocol: %s\n", strings.ToUpper(proto)))
		b.WriteString(fmt.Sprintf("  Server:   %s\n", addr))
		b.WriteString("\n")
		b.WriteString("Profile name:\n")
		b.WriteString(m.nameInput.View())
		if m.err != "" {
			b.WriteString("\n\n")
			b.WriteString(errStyle.Render(m.err))
		}
		b.WriteString("\n\n")
		b.WriteString(hint.Render("[enter] save  [esc] back  [q] skip"))

	case WizardStepSubURL:
		b.WriteString("Paste your subscription URL:\n")
		b.WriteString(m.subURLInput.View())
		if m.err != "" {
			b.WriteString("\n\n")
			b.WriteString(errStyle.Render(m.err))
		}
		b.WriteString("\n\n")
		b.WriteString(hint.Render("[enter] next  [esc] back  [q] skip"))

	case WizardStepSubName:
		b.WriteString("Name this subscription:\n")
		b.WriteString(m.subNameInput.View())
		b.WriteString("\n\n")
		b.WriteString(muted.Render("Servers refresh automatically every 24h."))
		b.WriteString("\n\n")
		b.WriteString(hint.Render("[enter] add  [esc] back  [q] skip"))

	case WizardStepDone:
		b.WriteString(okStyle.Render("You're all set"))
		b.WriteString("\n\n")
		switch m.method {
		case methodSubscription:
			b.WriteString(normal.Render(fmt.Sprintf("Subscription %q added.", m.SubName)))
		default:
			b.WriteString(normal.Render(fmt.Sprintf("Imported %q, set as default.", m.Profile.Name)))
		}
		b.WriteString("\n\n")
		b.WriteString("What next:\n")
		b.WriteString(normal.Render("  " + nextNudge(nudgeInput{method: m.method, startKey: m.StartKey}).Text))
		b.WriteString("\n\n")
		b.WriteString(hint.Render("[enter] go to profiles  [q] close"))
	}

	return modal.Render(b.String())
}
