package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestPanelStyle_Dimensions(t *testing.T) {
	tests := []struct {
		name   string
		active bool
		width  int
		height int
	}{
		{"active small", true, 20, 5},
		{"inactive small", false, 20, 5},
		{"active medium", true, 40, 10},
		{"inactive medium", false, 40, 10},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			style := panelStyle(tc.active, tc.width, tc.height)
			rendered := style.Render("test")
			w := lipgloss.Width(rendered)
			h := lipgloss.Height(rendered)

			// Panel style sets width and height, border adds 2 to each
			expectedW := tc.width + 2
			expectedH := tc.height + 2
			if w != expectedW {
				t.Errorf("panelStyle width = %d, want %d", w, expectedW)
			}
			if h != expectedH {
				t.Errorf("panelStyle height = %d, want %d", h, expectedH)
			}
		})
	}
}

func TestRenderPanelWithTitle_ContainsTitle(t *testing.T) {
	rendered := panelStyle(true, 30, 5).Render("content")
	result := renderPanelWithTitle(rendered, "Status", true)

	// The title should appear in the first line
	lines := strings.Split(result, "\n")
	if len(lines) == 0 {
		t.Fatal("rendered panel has no lines")
	}

	// Check that "Status" text appears (strip ANSI for checking)
	firstLine := stripANSI(lines[0])
	if !strings.Contains(firstLine, "Status") {
		t.Errorf("first line %q should contain title 'Status'", firstLine)
	}
}

func TestRenderPanelWithTitle_PreservesWidth(t *testing.T) {
	rendered := panelStyle(false, 40, 5).Render("content")
	originalWidth := lipgloss.Width(strings.Split(rendered, "\n")[0])

	result := renderPanelWithTitle(rendered, "Profiles", false)
	lines := strings.Split(result, "\n")

	newWidth := lipgloss.Width(lines[0])
	if newWidth != originalWidth {
		t.Errorf("title line width %d != original %d", newWidth, originalWidth)
	}
}

func TestRenderPanelWithTitle_NoANSIArtifacts(t *testing.T) {
	rendered := panelStyle(true, 30, 5).Render("content")
	result := renderPanelWithTitle(rendered, "Test", true)

	stripped := stripANSI(result)
	if strings.Contains(stripped, "[0m") {
		t.Error("found [0m artifact in rendered output")
	}
	if strings.Contains(stripped, "\x1b") {
		t.Error("found stray escape character in stripped output")
	}
}

func TestRenderPanelWithTitle_EmptyTitle(t *testing.T) {
	rendered := panelStyle(true, 30, 5).Render("content")
	result := renderPanelWithTitle(rendered, "", true)

	if result != rendered {
		t.Error("empty title should return original rendered string")
	}
}

func TestRenderPanelWithTitle_TitleTooWide(t *testing.T) {
	rendered := panelStyle(true, 6, 3).Render("x")
	result := renderPanelWithTitle(rendered, "VeryLongTitle", true)

	if result != rendered {
		t.Error("title wider than panel should return original rendered string")
	}
}

func TestRenderPanelWithTitle_BorderChars(t *testing.T) {
	rendered := panelStyle(true, 30, 5).Render("content")
	result := renderPanelWithTitle(rendered, "Title", true)

	stripped := stripANSI(result)
	lines := strings.Split(stripped, "\n")
	firstLine := lines[0]

	border := lipgloss.RoundedBorder()
	if !strings.HasPrefix(firstLine, border.TopLeft) {
		t.Errorf("first line should start with %q, got %q", border.TopLeft, firstLine[:3])
	}
	if !strings.HasSuffix(firstLine, border.TopRight) {
		t.Errorf("first line should end with %q, got %q", border.TopRight, firstLine[len(firstLine)-3:])
	}
}

// stripANSI removes ANSI escape sequences for testing.
func stripANSI(s string) string {
	var result strings.Builder
	inEscape := false
	for _, r := range s {
		if r == '\x1b' {
			inEscape = true
			continue
		}
		if inEscape {
			if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') {
				inEscape = false
			}
			continue
		}
		result.WriteRune(r)
	}
	return result.String()
}
