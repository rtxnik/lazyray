// Package clihint renders command failures with an optional, actionable next
// step ("hint"). It is the single place lazyray formats error output, so every
// command speaks with one voice. The package is pure: Render writes only to the
// io.Writer it is given, and the package holds no global state.
package clihint

import (
	"errors"
	"fmt"
	"io"
)

// Error is a user-facing failure carrying an optional actionable next step.
// It is an ordinary error; the Hint is surfaced only by Render. Msg is the
// already-formatted message; Hint is an imperative next step (may be empty).
type Error struct {
	Msg     string
	Hint    string
	wrapped error
}

// Error returns the message, satisfying the error interface.
func (e *Error) Error() string { return e.Msg }

// Unwrap exposes any error wrapped via a %w verb in Errorf, so errors.Is and
// errors.As keep working through a *clihint.Error. The wrapped field holds the
// fmt.Errorf value directly, preserving its own Unwrap implementation (which
// may return a []error for multi-%w formats, valid since Go 1.20).
func (e *Error) Unwrap() error { return e.wrapped }

// Errorf builds an *Error whose Msg is fmt-formatted from format and args and
// whose Hint is fixed. A %w verb in format is honored: the wrapped error stays
// reachable via errors.Is / errors.As (including multi-%w, Go 1.20+).
func Errorf(hint, format string, a ...any) *Error {
	formatted := fmt.Errorf(format, a...)
	return &Error{Msg: formatted.Error(), Hint: hint, wrapped: formatted}
}

// Render writes the canonical failure presentation for err to w:
//
//	Error: <msg>
//	  → try: <hint>
//
// The hint line appears only when err carries a *Error with a non-empty Hint
// (found anywhere in the chain via errors.As). For a plain error the message is
// printed with no hint line. Render is a no-op when err is nil.
func Render(w io.Writer, err error) {
	if err == nil {
		return
	}
	var he *Error
	if errors.As(err, &he) {
		fmt.Fprintf(w, "Error: %s\n", he.Msg)
		if he.Hint != "" {
			fmt.Fprintf(w, "  → try: %s\n", he.Hint)
		}
		return
	}
	fmt.Fprintf(w, "Error: %s\n", err.Error())
}
