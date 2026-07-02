// internal/lifecycle/lock_unix.go
//go:build !windows

package lifecycle

import (
	"os"
	"syscall"

	"github.com/rtxnik/lazyray/internal/config"
)

// Lock is an exclusive advisory lock held for the supervisor's lifetime.
type Lock struct {
	f *os.File
}

// AcquireLock takes a non-blocking exclusive flock on the lock file.
// Returns ErrLocked if another open file description already holds it.
func AcquireLock() (*Lock, error) {
	if err := config.EnsureDirs(); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(LockPath(), os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		f.Close()
		if err == syscall.EWOULDBLOCK {
			return nil, ErrLocked
		}
		return nil, err
	}
	return &Lock{f: f}, nil
}

// Release unlocks and closes the lock file.
func (l *Lock) Release() error {
	if l == nil || l.f == nil {
		return nil
	}
	_ = syscall.Flock(int(l.f.Fd()), syscall.LOCK_UN)
	err := l.f.Close()
	l.f = nil
	return err
}

// SupervisorAlive reports whether a live supervisor currently holds the lock.
// It probes by attempting to acquire; success means no live holder.
func SupervisorAlive() bool {
	l, err := AcquireLock()
	if err != nil {
		// ErrLocked (held) or any probe error → treat as "alive/uncertain".
		return true
	}
	_ = l.Release()
	return false
}
