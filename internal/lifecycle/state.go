// internal/lifecycle/state.go
package lifecycle

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/rtxnik/lazyray/internal/config"
	"github.com/rtxnik/lazyray/internal/fsutil"
)

// Owner identifies which entry point started the supervised session.
type Owner string

const (
	OwnerDaemon  Owner = "daemon"
	OwnerTUI     Owner = "tui"
	OwnerService Owner = "service"
)

// ProxySnapshot is the system-proxy state captured before lazyray modified it,
// so teardown can restore it instead of blindly turning everything off.
type ProxySnapshot struct {
	HTTPEnabled  bool   `json:"httpEnabled"`
	HTTPHost     string `json:"httpHost"`
	HTTPPort     int    `json:"httpPort"`
	SOCKSEnabled bool   `json:"socksEnabled"`
	SOCKSHost    string `json:"socksHost"`
	SOCKSPort    int    `json:"socksPort"`
	PACEnabled   bool   `json:"pacEnabled"`
	PACURL       string `json:"pacURL"`
}

// Routing records the OS-level side effects lazyray applied for this session.
type Routing struct {
	SystemProxy bool           `json:"systemProxy"`
	PAC         bool           `json:"pac"`
	Prior       *ProxySnapshot `json:"prior,omitempty"`
}

// State is the durable runtime record of one supervised session.
type State struct {
	Owner         Owner     `json:"owner"`
	SupervisorPID int       `json:"supervisorPID"`
	XrayPID       int       `json:"xrayPID"`
	StartedAt     time.Time `json:"startedAt"`
	SocksPort     int       `json:"socksPort"`
	HTTPPort      int       `json:"httpPort"`
	Routing       Routing   `json:"routing"`
	ActiveProfile string    `json:"activeProfile"` // profile name currently routed by xray
}

// StatePath is the runtime state file.
func StatePath() string { return filepath.Join(config.DataDir(), "state.json") }

// LockPath is the single-instance lock file.
func LockPath() string { return filepath.Join(config.DataDir(), "supervisor.lock") }

// WriteState atomically writes the state file with 0600 perms.
func WriteState(s *State) error {
	if err := config.EnsureDirs(); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return fsutil.WriteFile(StatePath(), data, 0o600)
}

// ReadState reads the state file. Returns (nil, nil) when the file is absent.
func ReadState() (*State, error) {
	data, err := os.ReadFile(StatePath())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var s State
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

// RemoveState deletes the state file; absent is not an error.
func RemoveState() error {
	if err := os.Remove(StatePath()); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
