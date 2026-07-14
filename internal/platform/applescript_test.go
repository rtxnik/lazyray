package platform

import "testing"

func TestEscapeAppleScript(t *testing.T) {
	cases := map[string]string{
		`plain`:        `plain`,
		`a"b`:          `a\"b`,
		`a\b`:          `a\\b`,
		"line1\nline2": "line1 line2",
		"cr\rtab":      "cr tab",
		`"; do evil`:   `\"; do evil`,
	}
	for in, want := range cases {
		if got := escapeAppleScript(in); got != want {
			t.Errorf("escapeAppleScript(%q) = %q, want %q", in, got, want)
		}
	}
}
