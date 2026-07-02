// internal/lifecycle/lock_unix_test.go
//go:build !windows

package lifecycle

import (
	"errors"
	"testing"
)

func TestAcquireLock_AcquireAndRelease(t *testing.T) {
	withTempData(t)
	l, err := AcquireLock()
	if err != nil {
		t.Fatalf("AcquireLock() error = %v", err)
	}
	if l == nil {
		t.Fatal("AcquireLock() returned nil lock")
	}
	if err := l.Release(); err != nil {
		t.Errorf("Release() error = %v", err)
	}
	// Reacquire after release must succeed.
	l2, err := AcquireLock()
	if err != nil {
		t.Fatalf("re-AcquireLock() error = %v", err)
	}
	_ = l2.Release()
}

func TestAcquireLock_ContentionReturnsErrLocked(t *testing.T) {
	withTempData(t)
	l, err := AcquireLock()
	if err != nil {
		t.Fatalf("AcquireLock() error = %v", err)
	}
	defer func() { _ = l.Release() }()

	// A second open+flock from the same process uses a distinct open file
	// description, so flock(LOCK_EX|LOCK_NB) must fail with ErrLocked.
	_, err = AcquireLock()
	if !errors.Is(err, ErrLocked) {
		t.Errorf("second AcquireLock() error = %v, want ErrLocked", err)
	}
}

func TestSupervisorAlive(t *testing.T) {
	withTempData(t)
	if SupervisorAlive() {
		t.Error("SupervisorAlive() = true with no holder, want false")
	}
	l, _ := AcquireLock()
	defer func() { _ = l.Release() }()
	if !SupervisorAlive() {
		t.Error("SupervisorAlive() = false while lock held, want true")
	}
}
