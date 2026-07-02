// internal/lifecycle/errors.go
package lifecycle

import "errors"

// ErrLocked is returned when another process already holds the supervisor lock.
var ErrLocked = errors.New("supervisor lock already held")
