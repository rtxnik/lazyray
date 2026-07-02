package modals

import (
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/rtxnik/lazyray/internal/tui/commands"
	"github.com/rtxnik/lazyray/internal/tui/theme"
	"github.com/sahilm/fuzzy"
)

// paletteWindow is the number of command rows shown at once (the scroll window).
const paletteWindow = 8

// PaletteModal is a fuzzy launcher over a slice of launchable commands. On Enter
// it records the highlighted command in Selected and sets Done; on Esc it sets
// Done with Selected left nil. It mutates no application state — the App reads
// Selected and re-injects that command's key through handleKeyPress.
type PaletteModal struct {
	Done     bool
	Selected *commands.Command

	all      []commands.Command
	filtered []commands.Command
	matched  map[int][]int // filtered index -> matched rune positions in Title
	cursor   int
	offset   int
	input    textinput.Model
	width    int
	height   int
}

// cmdSource adapts commands to fuzzy.Source over their Title.
type cmdSource []commands.Command

func (s cmdSource) String(i int) string { return s[i].Title }
func (s cmdSource) Len() int            { return len(s) }

// NewPaletteModal builds the palette over the given launchable commands.
func NewPaletteModal(cmds []commands.Command, width, height int) *PaletteModal {
	ti := textinput.New()
	ti.Prompt = ": "
	ti.Placeholder = "type to filter…"
	ti.CharLimit = 64
	ti.Width = 40
	ti.Focus()
	m := &PaletteModal{all: cmds, input: ti, width: width, height: height}
	m.refilter()
	return m
}

func (m *PaletteModal) Init() tea.Cmd { return textinput.Blink }

// refilter recomputes filtered/matched from the query and clamps the cursor.
func (m *PaletteModal) refilter() {
	m.matched = map[int][]int{}
	q := strings.TrimSpace(m.input.Value())
	if q == "" {
		m.filtered = m.all
	} else {
		res := fuzzy.FindFrom(q, cmdSource(m.all))
		filtered := make([]commands.Command, 0, len(res))
		for i, match := range res {
			filtered = append(filtered, m.all[match.Index])
			m.matched[i] = match.MatchedIndexes
		}
		m.filtered = filtered
	}
	if m.cursor >= len(m.filtered) {
		m.cursor = len(m.filtered) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
	m.clampWindow()
}

// clampWindow keeps the cursor inside [offset, offset+paletteWindow).
func (m *PaletteModal) clampWindow() {
	if m.cursor < m.offset {
		m.offset = m.cursor
	}
	if m.cursor >= m.offset+paletteWindow {
		m.offset = m.cursor - paletteWindow + 1
	}
	if m.offset < 0 {
		m.offset = 0
	}
}

func (m *PaletteModal) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	km, ok := msg.(tea.KeyMsg)
	if !ok {
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}
	switch km.String() {
	case "enter":
		if len(m.filtered) > 0 {
			sel := m.filtered[m.cursor]
			m.Selected = &sel
		}
		m.Done = true
		return m, nil
	case "esc":
		m.Done = true
		return m, nil
	case "up", "ctrl+p":
		if m.cursor > 0 {
			m.cursor--
			m.clampWindow()
		}
		return m, nil
	case "down", "ctrl+n":
		if m.cursor < len(m.filtered)-1 {
			m.cursor++
			m.clampWindow()
		}
		return m, nil
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	m.refilter()
	return m, cmd
}

// highlightTitle renders a title with the matched characters in the hl style.
// fuzzy.Match.MatchedIndexes holds BYTE offsets (the matcher advances by rune
// byte-size), so we walk the string by byte offset and decode each rune rather
// than indexing by rune position — correct for multi-byte titles, identical to
// rune indexing for the ASCII titles in use today.
func highlightTitle(title string, matched []int, base, hl lipgloss.Style) string {
	if len(matched) == 0 {
		return base.Render(title)
	}
	set := make(map[int]bool, len(matched))
	for _, off := range matched {
		set[off] = true
	}
	var b strings.Builder
	for off := 0; off < len(title); {
		r, size := utf8.DecodeRuneInString(title[off:])
		if set[off] {
			b.WriteString(hl.Render(string(r)))
		} else {
			b.WriteString(base.Render(string(r)))
		}
		off += size
	}
	return b.String()
}

func (m *PaletteModal) renderRow(i, width int, st theme.Styles) string {
	c := m.filtered[i]
	marker := "  "
	var title string
	if i == m.cursor {
		marker = st.Accent.Render("› ")
		title = st.Selected.Render(c.Title) // selection dominates per-char highlight
	} else {
		title = highlightTitle(c.Title, m.matched[i], st.Desc, st.Accent)
	}
	left := marker + title
	keyPart := st.Key.Render(commands.KeyDisplay(c.Binding))
	gap := width - lipgloss.Width(left) - lipgloss.Width(keyPart)
	if gap < 1 {
		gap = 1
	}
	return left + strings.Repeat(" ", gap) + keyPart
}

func (m *PaletteModal) View() string {
	st := theme.CurrentStyles()

	mw := 56
	if m.width > 0 && m.width-4 < mw {
		mw = m.width - 4
	}
	if mw < 30 {
		mw = 30
	}
	rowWidth := mw - 4 // inside Padding(1,2)

	var b strings.Builder
	b.WriteString(st.Title.Render("Command Palette"))
	b.WriteString("\n\n")
	b.WriteString(m.input.View())
	b.WriteString("\n\n")

	if len(m.filtered) == 0 {
		b.WriteString(st.Muted.Render("No matching command."))
	} else {
		end := m.offset + paletteWindow
		if end > len(m.filtered) {
			end = len(m.filtered)
		}
		for i := m.offset; i < end; i++ {
			b.WriteString(m.renderRow(i, rowWidth, st))
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")
	b.WriteString(st.Muted.Render("[↑↓] select   [Enter] run   [Esc] close"))

	return lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(theme.Current().Accent).
		Padding(1, 2).
		Width(mw).
		Render(b.String())
}
