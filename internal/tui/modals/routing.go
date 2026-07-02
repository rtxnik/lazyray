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

// RoutingModal handles per-profile routing rules editing.
// Three sections: Bypass (→ direct), Block (→ blackhole), and DNS rules.
// Each entry is one rule: geoip:cn, geosite:category-ads-all, domain:example.com, 10.0.0.0/8, etc.
type RoutingModal struct {
	bypassInput textinput.Model
	blockInput  textinput.Model
	dnsInput    textinput.Model
	focusIndex  int // 0=bypass, 1=block, 2=dns
	fieldCount  int
	Done        bool
	Routing     *config.ProfileRouting
	err         string
	width       int
	height      int
	profileName string
}

// NewRoutingModal creates a routing editor pre-filled with existing rules.
func NewRoutingModal(profile *config.Profile, width, height int) *RoutingModal {
	m := &RoutingModal{
		width:       width,
		height:      height,
		profileName: profile.Name,
		fieldCount:  3,
	}

	m.bypassInput = textinput.New()
	m.bypassInput.Prompt = ""
	m.bypassInput.CharLimit = 1024
	m.bypassInput.Width = 50
	m.bypassInput.Placeholder = "geoip:private, geosite:private, domain:example.com"
	m.bypassInput.SetValue(strings.Join(profile.Routing.Bypass, ", "))

	m.blockInput = textinput.New()
	m.blockInput.Prompt = ""
	m.blockInput.CharLimit = 1024
	m.blockInput.Width = 50
	m.blockInput.Placeholder = "geosite:category-ads-all, domain:ads.example.com"
	m.blockInput.SetValue(strings.Join(profile.Routing.Block, ", "))

	m.dnsInput = textinput.New()
	m.dnsInput.Prompt = ""
	m.dnsInput.CharLimit = 2048
	m.dnsInput.Width = 50
	m.dnsInput.Placeholder = "domain:example.com>https://dns.google/dns-query"
	m.dnsInput.SetValue(formatDNSRules(profile.Routing.DNSRules))

	m.bypassInput.Focus()
	return m
}

func (m *RoutingModal) Init() tea.Cmd {
	return textinput.Blink
}

func (m *RoutingModal) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "tab", "down":
			m.moveFocus(1)
			return m, textinput.Blink
		case "shift+tab", "up":
			m.moveFocus(-1)
			return m, textinput.Blink
		case "ctrl+s":
			return m, m.submit()
		case "enter":
			if m.focusIndex == m.fieldCount-1 {
				return m, m.submit()
			}
			m.moveFocus(1)
			return m, textinput.Blink
		case "esc":
			m.Done = true
			return m, nil
		}
	}

	var cmd tea.Cmd
	switch m.focusIndex {
	case 0:
		m.bypassInput, cmd = m.bypassInput.Update(msg)
	case 1:
		m.blockInput, cmd = m.blockInput.Update(msg)
	case 2:
		m.dnsInput, cmd = m.dnsInput.Update(msg)
	}
	return m, cmd
}

func (m *RoutingModal) moveFocus(delta int) {
	m.blurCurrent()
	m.focusIndex = (m.focusIndex + delta + m.fieldCount) % m.fieldCount
	m.focusCurrent()
}

func (m *RoutingModal) blurCurrent() {
	switch m.focusIndex {
	case 0:
		m.bypassInput.Blur()
	case 1:
		m.blockInput.Blur()
	case 2:
		m.dnsInput.Blur()
	}
}

func (m *RoutingModal) focusCurrent() {
	switch m.focusIndex {
	case 0:
		m.bypassInput.Focus()
	case 1:
		m.blockInput.Focus()
	case 2:
		m.dnsInput.Focus()
	}
}

func (m *RoutingModal) submit() tea.Cmd {
	return func() tea.Msg {
		bypass := parseRuleEntries(m.bypassInput.Value())
		block := parseRuleEntries(m.blockInput.Value())

		// Validate entries
		for _, entry := range bypass {
			if err := validateRuleEntry(entry); err != nil {
				m.err = fmt.Sprintf("Bypass rule error: %s", err)
				return nil
			}
		}
		for _, entry := range block {
			if err := validateRuleEntry(entry); err != nil {
				m.err = fmt.Sprintf("Block rule error: %s", err)
				return nil
			}
		}

		// Parse DNS rules
		dnsRules, err := parseDNSRules(m.dnsInput.Value())
		if err != nil {
			m.err = fmt.Sprintf("DNS rule error: %s", err)
			return nil
		}

		m.Routing = &config.ProfileRouting{
			Bypass:   bypass,
			Block:    block,
			DNSRules: dnsRules,
		}
		m.Done = true
		return nil
	}
}

// parseRuleEntries splits comma-separated entries and trims whitespace.
func parseRuleEntries(input string) []string {
	input = strings.TrimSpace(input)
	if input == "" {
		return nil
	}
	parts := strings.Split(input, ",")
	var entries []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			entries = append(entries, p)
		}
	}
	return entries
}

// validateRuleEntry checks that a routing rule entry looks valid.
func validateRuleEntry(entry string) error {
	if entry == "" {
		return fmt.Errorf("empty entry")
	}
	// Known valid prefixes
	validPrefixes := []string{
		"geoip:", "geosite:", "domain:", "full:", "regexp:", "keyword:",
	}
	for _, prefix := range validPrefixes {
		if strings.HasPrefix(entry, prefix) {
			value := strings.TrimPrefix(entry, prefix)
			if value == "" {
				return fmt.Errorf("%q has empty value", entry)
			}
			return nil
		}
	}
	// Bare IP/CIDR — basic format check
	if strings.Contains(entry, "/") || strings.Contains(entry, ":") {
		return nil // CIDR or IPv6
	}
	if len(entry) > 0 && entry[0] >= '0' && entry[0] <= '9' && strings.Contains(entry, ".") {
		return nil // IPv4
	}
	// Could be a bare domain
	if strings.Contains(entry, ".") {
		return nil
	}
	return fmt.Errorf("%q is not a valid rule (use geoip:, geosite:, domain:, IP/CIDR, or domain name)", entry)
}

// parseDNSRules parses comma-separated DNS routing rules.
// Format: "domains>server" where domains is semicolon-separated.
// Example: "domain:example.com;domain:google.com>https://dns.google/dns-query"
func parseDNSRules(input string) ([]config.DNSRule, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return nil, nil
	}

	parts := strings.Split(input, ",")
	var rules []config.DNSRule

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// Split on ">" to separate domains from server
		idx := strings.Index(part, ">")
		if idx < 0 {
			return nil, fmt.Errorf("%q: use format 'domains>server' (e.g. domain:example.com>https://dns.google/dns-query)", part)
		}

		domainsStr := strings.TrimSpace(part[:idx])
		server := strings.TrimSpace(part[idx+1:])

		if server == "" {
			return nil, fmt.Errorf("empty DNS server in rule %q", part)
		}
		if domainsStr == "" {
			return nil, fmt.Errorf("empty domains in rule %q", part)
		}

		// Domains are semicolon-separated within a rule
		var domains []string
		for _, d := range strings.Split(domainsStr, ";") {
			d = strings.TrimSpace(d)
			if d != "" {
				domains = append(domains, d)
			}
		}

		rules = append(rules, config.DNSRule{
			Server:  server,
			Domains: domains,
		})
	}

	return rules, nil
}

// formatDNSRules converts DNS rules back to the editable string format.
func formatDNSRules(rules []config.DNSRule) string {
	if len(rules) == 0 {
		return ""
	}

	var parts []string
	for _, rule := range rules {
		domains := strings.Join(rule.Domains, ";")
		parts = append(parts, fmt.Sprintf("%s>%s", domains, rule.Server))
	}
	return strings.Join(parts, ", ")
}

func (m *RoutingModal) View() string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.Current().Accent).
		MarginBottom(1)

	labelStyle := lipgloss.NewStyle().
		Foreground(theme.Current().Selected).
		Bold(true)
	focusedLabelStyle := labelStyle.
		Foreground(theme.Current().Orange)

	dimStyle := lipgloss.NewStyle().
		Foreground(theme.Current().Muted)

	modalWidth := 72
	if m.width > 0 && m.width-4 < modalWidth {
		modalWidth = m.width - 4
	}
	if modalWidth < 40 {
		modalWidth = 40
	}

	inputWidth := modalWidth - 8
	if inputWidth < 20 {
		inputWidth = 20
	}
	m.bypassInput.Width = inputWidth
	m.blockInput.Width = inputWidth
	m.dnsInput.Width = inputWidth

	modal := lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(theme.Current().Accent).
		Padding(1, 2).
		Width(modalWidth)

	var b strings.Builder
	b.WriteString(titleStyle.Render(fmt.Sprintf("Routing Rules — %s", m.profileName)))
	b.WriteString("\n\n")

	// Bypass section
	bypassLS := labelStyle
	if m.focusIndex == 0 {
		bypassLS = focusedLabelStyle
	}
	b.WriteString(bypassLS.Render("Bypass (→ direct):"))
	b.WriteString("\n")
	b.WriteString(m.bypassInput.View())
	b.WriteString("\n")
	b.WriteString(dimStyle.Render("  Comma-separated. e.g. geoip:private, geosite:private"))
	b.WriteString("\n\n")

	// Block section
	blockLS := labelStyle
	if m.focusIndex == 1 {
		blockLS = focusedLabelStyle
	}
	b.WriteString(blockLS.Render("Block (→ blackhole):"))
	b.WriteString("\n")
	b.WriteString(m.blockInput.View())
	b.WriteString("\n")
	b.WriteString(dimStyle.Render("  Comma-separated. e.g. geosite:category-ads-all"))
	b.WriteString("\n\n")

	// DNS rules section
	dnsLS := labelStyle
	if m.focusIndex == 2 {
		dnsLS = focusedLabelStyle
	}
	b.WriteString(dnsLS.Render("DNS Rules (conditional DNS):"))
	b.WriteString("\n")
	b.WriteString(m.dnsInput.View())
	b.WriteString("\n")
	b.WriteString(dimStyle.Render("  Format: domains>server. e.g. domain:example.com>https://dns.google/dns-query"))

	if m.err != "" {
		errStyle := lipgloss.NewStyle().Foreground(theme.Current().Error)
		b.WriteString("\n\n")
		b.WriteString(errStyle.Render(m.err))
	}

	b.WriteString("\n\n")
	hint := dimStyle
	b.WriteString(hint.Render("[Tab/↑↓] Switch field  [Ctrl+S/Enter] Save  [Esc] Cancel"))

	return modal.Render(b.String())
}
