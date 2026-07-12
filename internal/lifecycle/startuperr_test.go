// internal/lifecycle/startuperr_test.go
package lifecycle

import (
	"errors"
	"os"
	"runtime"
	"testing"

	"github.com/rtxnik/lazyray/internal/config"
)

func TestWriteReadClearStartupError_Roundtrip(t *testing.T) {
	withTempData(t)

	cause := errors.New("boom from xray start")
	if err := WriteStartupError("start", cause); err != nil {
		t.Fatalf("WriteStartupError() = %v", err)
	}

	got, err := ReadStartupError()
	if err != nil {
		t.Fatalf("ReadStartupError() = %v", err)
	}
	if got == nil {
		t.Fatal("ReadStartupError() = nil, want a record")
	}
	if got.Stage != "start" {
		t.Errorf("Stage = %q, want %q", got.Stage, "start")
	}
	if got.Message != "boom from xray start" {
		t.Errorf("Message = %q, want %q", got.Message, "boom from xray start")
	}
	if got.Time.IsZero() {
		t.Error("Time is zero, want a timestamp")
	}

	if runtime.GOOS != "windows" {
		info, statErr := os.Stat(config.LastErrorPath())
		if statErr != nil {
			t.Fatalf("stat last-error = %v", statErr)
		}
		if perm := info.Mode().Perm(); perm != 0o600 {
			t.Errorf("last-error perm = %o, want 600", perm)
		}
	}

	if err := ClearStartupError(); err != nil {
		t.Fatalf("ClearStartupError() = %v", err)
	}
	after, err := ReadStartupError()
	if err != nil {
		t.Fatalf("ReadStartupError() after clear = %v", err)
	}
	if after != nil {
		t.Errorf("ReadStartupError() after clear = %+v, want nil", after)
	}
}

func TestReadStartupError_AbsentIsNilNil(t *testing.T) {
	withTempData(t)
	got, err := ReadStartupError()
	if err != nil {
		t.Fatalf("ReadStartupError() = %v, want nil error", err)
	}
	if got != nil {
		t.Errorf("ReadStartupError() = %+v, want nil when file absent", got)
	}
}

func TestClearStartupError_AbsentIsNil(t *testing.T) {
	withTempData(t)
	if err := ClearStartupError(); err != nil {
		t.Errorf("ClearStartupError() on absent file = %v, want nil", err)
	}
}

func TestStagedError_WrapsAndUnwraps(t *testing.T) {
	cause := errors.New("routing failed")
	se := &StagedError{Stage: "routing", Err: cause}

	if !errors.Is(se, cause) {
		t.Error("errors.Is(se, cause) = false, want true (Unwrap chain)")
	}
	var target *StagedError
	if !errors.As(se, &target) {
		t.Fatal("errors.As(se, &*StagedError) = false, want true")
	}
	if target.Stage != "routing" {
		t.Errorf("Stage = %q, want %q", target.Stage, "routing")
	}
	if se.Error() == "" {
		t.Error("Error() = empty string, want a message")
	}
}

func TestStartupFailureStage(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{"nil", nil, ""},
		{"plain error (post-success teardown)", errors.New("teardown boom"), ""},
		{"staged start failure is recorded", &StagedError{Stage: "start", Err: errors.New("exec fail")}, "start"},
		{"lock contention is expected, not recorded", &StagedError{Stage: "lock", Err: ErrLocked}, ""},
		{"genuine lock-file failure is recorded", &StagedError{Stage: "lock", Err: errors.New("permission denied")}, "lock"},
	}
	for _, tc := range tests {
		if got := StartupFailureStage(tc.err); got != tc.want {
			t.Errorf("%s: StartupFailureStage() = %q, want %q", tc.name, got, tc.want)
		}
	}
}
