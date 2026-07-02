package panels

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewLogsPanel(t *testing.T) {
	l := NewLogsPanel(50)
	if l.MaxLines != 50 {
		t.Errorf("MaxLines = %d, want 50", l.MaxLines)
	}
}

func TestNewLogsPanel_DefaultMaxLines(t *testing.T) {
	l := NewLogsPanel(0)
	if l.MaxLines != 100 {
		t.Errorf("MaxLines = %d, want 100 (default)", l.MaxLines)
	}
}

func TestLogsPanel_Init(t *testing.T) {
	l := NewLogsPanel(50)
	l.Init(80, 20)
	if !l.ready {
		t.Error("Init should set ready to true")
	}
	if l.Viewport.Width != 80 {
		t.Errorf("Viewport.Width = %d, want 80", l.Viewport.Width)
	}
}

func TestLogsPanel_Resize(t *testing.T) {
	l := NewLogsPanel(50)
	l.Init(80, 20)
	l.Resize(100, 30)
	if l.Width != 100 {
		t.Errorf("Width = %d, want 100", l.Width)
	}
	if l.Height != 30 {
		t.Errorf("Height = %d, want 30", l.Height)
	}
}

func TestLogsPanel_Resize_WithFilter(t *testing.T) {
	l := NewLogsPanel(50)
	l.Init(80, 20)
	l.Filtering = true
	l.Resize(80, 10)
	// Viewport height should be height - 1 when filtering
	if l.Viewport.Height != 9 {
		t.Errorf("Viewport.Height = %d, want 9 (height-1 for filter)", l.Viewport.Height)
	}
}

func TestLogsPanel_ToggleLogType(t *testing.T) {
	l := NewLogsPanel(50)
	l.Init(80, 20)
	if l.ShowError {
		t.Error("initial ShowError should be false")
	}
	l.ToggleLogType()
	if !l.ShowError {
		t.Error("after toggle, ShowError should be true")
	}
	l.ToggleLogType()
	if l.ShowError {
		t.Error("after second toggle, ShowError should be false")
	}
}

func TestLogsPanel_ToggleFilter(t *testing.T) {
	l := NewLogsPanel(50)
	l.Init(80, 20)
	l.ToggleFilter()
	if !l.Filtering {
		t.Error("after ToggleFilter, Filtering should be true")
	}
	l.ToggleFilter()
	if l.Filtering {
		t.Error("after second ToggleFilter, Filtering should be false")
	}
}

func TestLogsPanel_ToggleSearch(t *testing.T) {
	l := NewLogsPanel(50)
	l.Init(80, 20)
	l.ToggleSearch()
	if !l.Searching {
		t.Error("after ToggleSearch, Searching should be true")
	}
	l.ToggleSearch()
	if l.Searching {
		t.Error("after second ToggleSearch, Searching should be false")
	}
}

func TestLogsPanel_ToggleFilter_CancelsSearch(t *testing.T) {
	l := NewLogsPanel(50)
	l.Init(80, 20)
	l.ToggleSearch()
	l.ToggleFilter()
	if l.Searching {
		t.Error("ToggleFilter should cancel Searching")
	}
	if !l.Filtering {
		t.Error("ToggleFilter should activate Filtering")
	}
}

func TestLogsPanel_ToggleSearch_CancelsFilter(t *testing.T) {
	l := NewLogsPanel(50)
	l.Init(80, 20)
	l.ToggleFilter()
	l.ToggleSearch()
	if l.Filtering {
		t.Error("ToggleSearch should cancel Filtering")
	}
	if !l.Searching {
		t.Error("ToggleSearch should activate Searching")
	}
}

func TestLogsPanel_ApplyFilter(t *testing.T) {
	l := NewLogsPanel(50)
	l.Init(80, 20)
	l.Lines = []string{"hello world", "foo bar", "hello again"}
	l.ToggleFilter()
	l.FilterInput.SetValue("hello")
	l.ApplyFilter()
	if l.FilterText != "hello" {
		t.Errorf("FilterText = %q, want 'hello'", l.FilterText)
	}
}

func TestLogsPanel_ApplySearch(t *testing.T) {
	l := NewLogsPanel(50)
	l.Init(80, 20)
	l.Lines = []string{"hello world"}
	l.ToggleSearch()
	l.SearchInput.SetValue("world")
	l.ApplySearch()
	if l.SearchText != "world" {
		t.Errorf("SearchText = %q, want 'world'", l.SearchText)
	}
}

func TestLogsPanel_CancelFilterSearch(t *testing.T) {
	l := NewLogsPanel(50)
	l.Init(80, 20)
	l.ToggleFilter()
	l.FilterInput.SetValue("test")
	l.ApplyFilter()
	l.CancelFilterSearch()
	if l.Filtering {
		t.Error("CancelFilterSearch should disable filtering")
	}
	if l.FilterText != "" {
		t.Error("CancelFilterSearch should clear filter text")
	}
}

func TestLogsPanel_AppendLine(t *testing.T) {
	l := NewLogsPanel(3)
	l.Init(80, 20)
	l.AppendLine("line 1")
	l.AppendLine("line 2")
	l.AppendLine("line 3")
	l.AppendLine("line 4")
	if len(l.Lines) != 3 {
		t.Errorf("Lines count = %d, want 3 (max)", len(l.Lines))
	}
	if l.Lines[0] != "line 2" {
		t.Errorf("Lines[0] = %q, want 'line 2' (oldest trimmed)", l.Lines[0])
	}
}

func TestLogsPanel_View_NotReady(t *testing.T) {
	l := NewLogsPanel(50)
	view := l.View()
	if !strings.Contains(view, "Loading") {
		t.Error("View when not ready should show 'Loading'")
	}
}

func TestLogsPanel_View_Ready(t *testing.T) {
	l := NewLogsPanel(50)
	l.Init(80, 20)
	l.Lines = []string{"test line"}
	l.updateViewportContent()
	view := l.View()
	if view == "" {
		t.Error("View should not be empty when ready")
	}
}

func TestLogsPanel_View_WithFilter(t *testing.T) {
	l := NewLogsPanel(50)
	l.Init(80, 20)
	l.ToggleFilter()
	view := l.View()
	if !strings.Contains(view, "filter") {
		t.Error("View with filter active should show 'filter' prompt")
	}
}

func TestLogsPanel_View_WithSearch(t *testing.T) {
	l := NewLogsPanel(50)
	l.Init(80, 20)
	l.ToggleSearch()
	view := l.View()
	if !strings.Contains(view, "search") {
		t.Error("View with search active should show 'search' prompt")
	}
}

func TestLogsPanel_LogTitle(t *testing.T) {
	l := NewLogsPanel(50)
	if l.LogTitle() != "Access Log" {
		t.Errorf("LogTitle() = %q, want 'Access Log'", l.LogTitle())
	}
	l.ShowError = true
	if l.LogTitle() != "Error Log" {
		t.Errorf("LogTitle() = %q, want 'Error Log'", l.LogTitle())
	}
}

func TestColorizeLogLine_Error(t *testing.T) {
	result := colorizeLogLine("2024/01/01 12:00:00 error: connection refused")
	if result == "" {
		t.Error("colorizeLogLine should return non-empty string for error lines")
	}
}

func TestColorizeLogLine_Accepted(t *testing.T) {
	result := colorizeLogLine("2024/01/01 12:00:00 accepted google.com:443")
	if result == "" {
		t.Error("colorizeLogLine should return non-empty string for accepted lines")
	}
}

func TestColorizeLogLine_WithTimestamp(t *testing.T) {
	result := colorizeLogLine("2024/01/01 12:34:56 some info line")
	if result == "" {
		t.Error("colorizeLogLine should return non-empty string")
	}
}

func TestColorizeLogLine_WithTag(t *testing.T) {
	result := colorizeLogLine("line with [socks-in >> proxy] tag")
	if result == "" {
		t.Error("colorizeLogLine should return non-empty string")
	}
}

func TestColorizeLogLine_WithDomain(t *testing.T) {
	result := colorizeLogLine("connecting to google.com")
	if result == "" {
		t.Error("colorizeLogLine should return non-empty string")
	}
}

func TestColorizeLogLine_PlainLine(t *testing.T) {
	input := "just some plain text"
	result := colorizeLogLine(input)
	// Plain line without any patterns should be returned as-is
	if result != input {
		t.Errorf("colorizeLogLine(%q) = %q, want unchanged", input, result)
	}
}

func TestTailFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.log")

	var lines []string
	for i := 0; i < 20; i++ {
		lines = append(lines, "line "+string(rune('A'+i)))
	}
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	result, err := tailFile(path, 5)
	if err != nil {
		t.Fatalf("tailFile: %v", err)
	}
	if len(result) != 5 {
		t.Errorf("tailFile returned %d lines, want 5", len(result))
	}
}

func TestTailFile_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.log")
	if err := os.WriteFile(path, []byte(""), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	result, err := tailFile(path, 5)
	if err != nil {
		t.Fatalf("tailFile: %v", err)
	}
	if result != nil {
		t.Errorf("tailFile of empty file should return nil, got %v", result)
	}
}

func TestTailFile_NonExistent(t *testing.T) {
	_, err := tailFile("/nonexistent/file.log", 5)
	if err == nil {
		t.Error("tailFile of non-existent file should return error")
	}
}
