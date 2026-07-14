package platform

import "strings"

// escapeAppleScript makes s safe to interpolate into an AppleScript double-quoted
// literal: newlines (a literal newline is a compile error) collapse to spaces,
// then backslash and double-quote are escaped (order matters).
func escapeAppleScript(s string) string {
	s = strings.NewReplacer("\r", " ", "\n", " ").Replace(s)
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return s
}
