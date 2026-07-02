package panels

import (
	"os"
	"regexp"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
	"github.com/rtxnik/lazyray/internal/config"
	"github.com/rtxnik/lazyray/internal/tui/theme"
)

var (
	ansiEscRegex   = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)
	timestampRegex = regexp.MustCompile(`^\d{4}/\d{2}/\d{2}\s+(\d{2}:\d{2}:\d{2})`)
	domainRegex    = regexp.MustCompile(`[a-zA-Z0-9][-a-zA-Z0-9]*\.(com|org|net|io|dev|me|co|ru|nl|de|uk|fr|eu|app|xyz)`)
	tagRegex       = regexp.MustCompile(`\[[^\]]+>>[^\]]+\]`)
)

// LogsPanel displays xray log output.
type LogsPanel struct {
	Viewport  viewport.Model
	ShowError bool
	Lines     []string
	MaxLines  int
	Width     int
	Height    int
	ready     bool

	// Filter
	Filtering   bool
	FilterInput textinput.Model
	FilterText  string

	// Search
	Searching   bool
	SearchInput textinput.Model
	SearchText  string
}

// NewLogsPanel creates a new logs panel.
func NewLogsPanel(maxLines int) LogsPanel {
	fi := textinput.New()
	fi.Placeholder = "filter..."
	fi.CharLimit = 128

	si := textinput.New()
	si.Placeholder = "search..."
	si.CharLimit = 128

	if maxLines <= 0 {
		maxLines = 100
	}

	return LogsPanel{
		MaxLines:    maxLines,
		FilterInput: fi,
		SearchInput: si,
	}
}

// Init initializes the viewport with dimensions.
func (l *LogsPanel) Init(width, height int) {
	l.Width = width
	l.Height = height
	vpHeight := height
	if l.Filtering || l.Searching {
		vpHeight = height - 1
	}
	if vpHeight < 1 {
		vpHeight = 1
	}
	l.Viewport = viewport.New(width, vpHeight)
	l.Viewport.Style = lipgloss.NewStyle()
	l.ready = true
}

// Resize updates viewport dimensions.
func (l *LogsPanel) Resize(width, height int) {
	l.Width = width
	l.Height = height
	if l.ready {
		l.Viewport.Width = width
		vpHeight := height
		if l.Filtering || l.Searching {
			vpHeight = height - 1
		}
		if vpHeight < 1 {
			vpHeight = 1
		}
		l.Viewport.Height = vpHeight
	}
}

// ToggleLogType switches between access and error logs.
func (l *LogsPanel) ToggleLogType() {
	l.ShowError = !l.ShowError
	l.Refresh()
}

// ToggleFilter toggles the filter input.
func (l *LogsPanel) ToggleFilter() {
	if l.Filtering {
		l.Filtering = false
		l.FilterInput.Blur()
		l.FilterText = ""
		l.adjustViewportHeight()
		l.updateViewportContent()
	} else {
		l.Searching = false
		l.SearchInput.Blur()
		l.Filtering = true
		l.FilterInput.SetValue("")
		l.FilterInput.Focus()
		l.adjustViewportHeight()
	}
}

// ToggleSearch toggles the search input.
func (l *LogsPanel) ToggleSearch() {
	if l.Searching {
		l.Searching = false
		l.SearchInput.Blur()
		l.SearchText = ""
		l.adjustViewportHeight()
		l.updateViewportContent()
	} else {
		l.Filtering = false
		l.FilterInput.Blur()
		l.Searching = true
		l.SearchInput.SetValue("")
		l.SearchInput.Focus()
		l.adjustViewportHeight()
	}
}

// ApplyFilter applies the current filter text.
func (l *LogsPanel) ApplyFilter() {
	l.FilterText = l.FilterInput.Value()
	l.updateViewportContent()
}

// ApplySearch applies the current search text.
func (l *LogsPanel) ApplySearch() {
	l.SearchText = l.SearchInput.Value()
	l.updateViewportContent()
}

// CancelFilterSearch cancels any active filter/search.
func (l *LogsPanel) CancelFilterSearch() {
	l.Filtering = false
	l.Searching = false
	l.FilterInput.Blur()
	l.SearchInput.Blur()
	l.FilterText = ""
	l.SearchText = ""
	l.adjustViewportHeight()
	l.updateViewportContent()
}

func (l *LogsPanel) adjustViewportHeight() {
	if !l.ready {
		return
	}
	vpHeight := l.Height
	if l.Filtering || l.Searching {
		vpHeight = l.Height - 1
	}
	if vpHeight < 1 {
		vpHeight = 1
	}
	l.Viewport.Height = vpHeight
}

// Refresh reloads log content from disk (reads only the tail).
func (l *LogsPanel) Refresh() {
	logPath := config.AccessLogPath()
	if l.ShowError {
		logPath = config.ErrorLogPath()
	}

	lines, err := tailFile(logPath, l.MaxLines)
	if err != nil {
		l.Lines = []string{"Logs appear here once the proxy runs."}
		if l.ready {
			l.Viewport.SetContent("Logs appear here once the proxy runs.")
		}
		return
	}

	for i, line := range lines {
		lines[i] = ansiEscRegex.ReplaceAllString(line, "")
	}
	l.Lines = lines

	l.updateViewportContent()
}

// tailFile reads the last n lines from a file without loading it entirely.
func tailFile(path string, n int) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		return nil, err
	}
	size := stat.Size()
	if size == 0 {
		return nil, nil
	}

	// Read from the end in chunks to find enough newlines
	const chunkSize = 8192
	buf := make([]byte, 0, chunkSize)
	offset := size
	newlines := 0

	for offset > 0 && newlines <= n {
		readSize := int64(chunkSize)
		if readSize > offset {
			readSize = offset
		}
		offset -= readSize

		chunk := make([]byte, readSize)
		if _, err := f.ReadAt(chunk, offset); err != nil {
			return nil, err
		}
		buf = append(chunk, buf...)

		for _, b := range chunk {
			if b == '\n' {
				newlines++
			}
		}
	}

	allLines := strings.Split(strings.TrimSpace(string(buf)), "\n")
	if len(allLines) > n {
		allLines = allLines[len(allLines)-n:]
	}
	return allLines, nil
}

func (l *LogsPanel) updateViewportContent() {
	if !l.ready {
		return
	}

	lines := l.Lines

	// Apply filter
	if l.FilterText != "" {
		var filtered []string
		lower := strings.ToLower(l.FilterText)
		for _, line := range lines {
			if strings.Contains(strings.ToLower(line), lower) {
				filtered = append(filtered, line)
			}
		}
		lines = filtered
	}

	// Colorize log lines
	for i, line := range lines {
		lines[i] = colorizeLogLine(line)
	}

	// Apply search highlighting
	if l.SearchText != "" {
		highlightStyle := lipgloss.NewStyle().
			Background(theme.Current().Selected).
			Foreground(theme.Current().Bg)

		lower := strings.ToLower(l.SearchText)
		for i, line := range lines {
			lineLower := strings.ToLower(line)
			if idx := strings.Index(lineLower, lower); idx >= 0 {
				matchLen := len(l.SearchText)
				before := line[:idx]
				match := line[idx : idx+matchLen]
				after := line[idx+matchLen:]
				lines[i] = before + highlightStyle.Render(match) + after
			}
		}
	}

	content := strings.Join(lines, "\n")
	l.Viewport.SetContent(content)
	l.Viewport.GotoBottom()
}

// AppendLine adds a new log line.
func (l *LogsPanel) AppendLine(line string) {
	l.Lines = append(l.Lines, line)
	if len(l.Lines) > l.MaxLines {
		l.Lines = l.Lines[len(l.Lines)-l.MaxLines:]
	}
	l.updateViewportContent()
}

// View renders the logs panel content.
func (l *LogsPanel) View() string {
	if !l.ready {
		return "Loading logs..."
	}

	var parts []string
	parts = append(parts, l.Viewport.View())

	if l.Filtering {
		prompt := lipgloss.NewStyle().Foreground(theme.Current().Accent).Bold(true).Render("filter: ")
		parts = append(parts, prompt+l.FilterInput.View())
	} else if l.Searching {
		prompt := lipgloss.NewStyle().Foreground(theme.Current().Selected).Bold(true).Render("search: ")
		parts = append(parts, prompt+l.SearchInput.View())
	}

	return strings.Join(parts, "\n")
}

// LogTitle returns the current log type title.
func (l *LogsPanel) LogTitle() string {
	if l.ShowError {
		return "Error Log"
	}
	return "Access Log"
}

// colorizeLogLine applies syntax highlighting to a single log line.
func colorizeLogLine(line string) string {
	logTimestamp := lipgloss.NewStyle().Foreground(theme.Current().Muted)
	logAccepted := lipgloss.NewStyle().Foreground(theme.Current().Success)
	logError := lipgloss.NewStyle().Foreground(theme.Current().Error)
	logDomain := lipgloss.NewStyle().Foreground(theme.Current().Accent)
	logTag := lipgloss.NewStyle().Foreground(theme.Current().Warning)

	lower := strings.ToLower(line)

	// Error/failed/rejected lines get full red treatment
	if strings.Contains(lower, "error") || strings.Contains(lower, "failed") || strings.Contains(lower, "rejected") {
		return logError.Render(line)
	}

	// Accepted lines get green treatment
	if strings.Contains(lower, "accepted") {
		return logAccepted.Render(line)
	}

	result := line

	// Colorize timestamps (HH:MM:SS)
	if m := timestampRegex.FindStringIndex(result); m != nil {
		ts := timestampRegex.FindStringSubmatch(result)
		if len(ts) > 1 {
			result = strings.Replace(result, ts[1], logTimestamp.Render(ts[1]), 1)
		}
	}

	// Colorize tags [socks-in >> proxy]
	result = tagRegex.ReplaceAllStringFunc(result, func(s string) string {
		return logTag.Render(s)
	})

	// Colorize domains
	result = domainRegex.ReplaceAllStringFunc(result, func(s string) string {
		return logDomain.Render(s)
	})

	return result
}
