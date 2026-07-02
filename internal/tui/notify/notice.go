// Package notify is the TUI's in-memory notification substrate: a typed,
// severity-tiered Notice and a bounded ring-buffer Log. It is pure data — it
// imports nothing from the tui/panels/modals packages — so the app, the future
// command palette (E2d), and the doctor modal (E2f) can share one event store.
package notify

import "time"

// Severity ranks a Notice. In the UI, color (never a glyph) carries severity.
type Severity int

const (
	Info Severity = iota
	Success
	Warning
	Error
)

// Tag is the short uppercase label shown before a notice message.
func (s Severity) Tag() string {
	switch s {
	case Error:
		return "ERROR"
	case Warning:
		return "WARN"
	case Success:
		return "OK"
	case Info:
		return "INFO"
	}
	return ""
}

// Notice is one user-facing event.
type Notice struct {
	ID       uint64    // monotonic; assigned by Log.Add; stable identity for dwell/coalesce
	Time     time.Time // set by the caller before Add (the app uses time.Now)
	Severity Severity
	Message  string
	Hint     string // optional actionable next step (from clihint.Error.Hint)
	Source   string // optional origin: "lifecycle"|"profile"|"subscription"|"update"|""
	Count    int    // coalescing count for consecutive identical notices (>=1)
}
