package modals

import (
	"bytes"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	qrterminal "github.com/mdp/qrterminal/v3"

	"github.com/rtxnik/lazyray/internal/tui/theme"
)

// QRModal displays a QR code for a proxy URL (VLESS, VMess, Trojan, Shadowsocks).
type QRModal struct {
	url    string
	name   string
	Done   bool
	width  int
	height int
}

// NewQRModal creates a QR code display modal.
func NewQRModal(name, url string, width, height int) *QRModal {
	return &QRModal{
		url:    url,
		name:   name,
		width:  width,
		height: height,
	}
}

func (m *QRModal) Init() tea.Cmd {
	return nil
}

func (m *QRModal) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.KeyMsg); ok {
		switch msg.String() {
		case "esc", "q", "enter":
			m.Done = true
		}
	}
	return m, nil
}

func (m *QRModal) View() string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.Current().Accent).
		MarginBottom(1)

	var buf bytes.Buffer
	qrterminal.GenerateWithConfig(m.url, qrterminal.Config{
		Level:          qrterminal.L,
		Writer:         &buf,
		HalfBlocks:     true,
		BlackChar:      qrterminal.BLACK_BLACK,
		WhiteBlackChar: qrterminal.WHITE_BLACK,
		WhiteChar:      qrterminal.WHITE_WHITE,
		BlackWhiteChar: qrterminal.BLACK_WHITE,
		QuietZone:      1,
	})

	qrText := strings.TrimSpace(buf.String())

	// Calculate modal width from QR output
	qrLines := strings.Split(qrText, "\n")
	maxQRWidth := 0
	for _, line := range qrLines {
		w := lipgloss.Width(line)
		if w > maxQRWidth {
			maxQRWidth = w
		}
	}

	modalWidth := maxQRWidth + 6
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
	b.WriteString(titleStyle.Render("QR Code: " + m.name))
	b.WriteString("\n\n")
	b.WriteString(qrText)
	b.WriteString("\n\n")
	hint := lipgloss.NewStyle().Foreground(theme.Current().Muted)
	b.WriteString(hint.Render("[Esc] Close"))

	return modal.Render(b.String())
}
