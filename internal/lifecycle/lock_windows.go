// internal/lifecycle/lock_windows.go
//go:build windows

package lifecycle

import (
	"os"
	"strconv"
	"strings"

	"github.com/rtxnik/lazyray/internal/config"
)

// Lock on Windows is a best-effort PID lockfile (no kernel advisory lock).
// Per the design (Unix-first), Windows single-instance is best-effort.
type Lock struct {
	path string
}

// AcquireLock creates the lock file exclusively. If it exists and records a
// live PID, returns ErrLocked; if the recorded PID is dead, it steals the lock.
func AcquireLock() (*Lock, error) {
	if err := config.EnsureDirs(); err != nil {
		return nil, err
	}
	path := LockPath()
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		if os.IsExist(err) {
			if data, rerr := os.ReadFile(path); rerr == nil {
				if pid, perr := strconv.Atoi(strings.TrimSpace(string(data))); perr == nil && isProcessAlive(pid) {
					return nil, ErrLocked
				}
			}
			// Stale lock: remove and retry once.
			_ = os.Remove(path)
			f, err = os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
			if err != nil {
				return nil, ErrLocked
			}
		} else {
			return nil, err
		}
	}
	_, _ = f.WriteString(strconv.Itoa(os.Getpid()))
	_ = f.Close()
	return &Lock{path: path}, nil
}

// Release removes the lock file.
func (l *Lock) Release() error {
	if l == nil || l.path == "" {
		return nil
	}
	err := os.Remove(l.path)
	l.path = ""
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// SupervisorAlive probes the PID lockfile.
func SupervisorAlive() bool {
	l, err := AcquireLock()
	if err != nil {
		return true
	}
	_ = l.Release()
	return false
}
