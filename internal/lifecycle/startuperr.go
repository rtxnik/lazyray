// internal/lifecycle/startuperr.go
package lifecycle

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/rtxnik/lazyray/internal/config"
	"github.com/rtxnik/lazyray/internal/fsutil"
)

// StartupError is the persisted record of the most recent supervisor startup
// failure, written by `lzr __run` so `lzr status`/`lzr doctor` can surface why
// a background start silently failed.
type StartupError struct {
	Time    time.Time `json:"time"`
	Stage   string    `json:"stage"` // one of: lock | routing | start | state | supervise
	Message string    `json:"message"`
}

// WriteStartupError atomically persists a startup failure record (0600).
func WriteStartupError(stage string, cause error) error {
	if err := config.EnsureDirs(); err != nil {
		return err
	}
	rec := StartupError{
		Time:    time.Now().UTC(),
		Stage:   stage,
		Message: cause.Error(),
	}
	data, err := json.MarshalIndent(&rec, "", "  ")
	if err != nil {
		return err
	}
	return fsutil.WriteFile(config.LastErrorPath(), data, 0o600)
}

// ReadStartupError reads the persisted record. Returns (nil, nil) when absent.
func ReadStartupError() (*StartupError, error) {
	data, err := os.ReadFile(config.LastErrorPath())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var rec StartupError
	if err := json.Unmarshal(data, &rec); err != nil {
		return nil, err
	}
	return &rec, nil
}

// ClearStartupError removes the persisted record; absent is not an error.
func ClearStartupError() error {
	if err := os.Remove(config.LastErrorPath()); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// StagedError wraps a supervisor failure with the stage at which it occurred so
// callers can tag the persisted StartupError without string-matching.
type StagedError struct {
	Stage string
	Err   error
}

func (e *StagedError) Error() string { return fmt.Sprintf("%s: %v", e.Stage, e.Err) }
func (e *StagedError) Unwrap() error { return e.Err }
