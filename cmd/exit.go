// cmd/exit.go
package cmd

import "errors"

// Process exit codes (documented contract for scripting).
const (
	ExitOK         = 0
	ExitGeneric    = 1
	ExitNotRunning = 3
	ExitConflict   = 4
	ExitConfig     = 5
)

// ExitError carries a specific process exit code out of a command.
type ExitError struct {
	Code int
	Err  error
}

func (e *ExitError) Error() string { return e.Err.Error() }
func (e *ExitError) Unwrap() error { return e.Err }

// exitCodeFor maps an error returned by command execution to a process code.
func exitCodeFor(err error) int {
	if err == nil {
		return ExitOK
	}
	var ee *ExitError
	if errors.As(err, &ee) {
		return ee.Code
	}
	return ExitGeneric
}
