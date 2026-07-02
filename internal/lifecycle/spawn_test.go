// internal/lifecycle/spawn_test.go
package lifecycle

import (
	"os"
	"runtime"
	"testing"

	"github.com/rtxnik/lazyray/internal/config"
)

// withTempHome points config's path helpers at a throwaway dir.
func withTempHome(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if runtime.GOOS == "windows" {
		t.Setenv("LOCALAPPDATA", dir)
		t.Setenv("APPDATA", dir)
	} else {
		t.Setenv("HOME", dir)
	}
	return dir
}

func TestOpenSupervisorLog_CreatesAppendableFile(t *testing.T) {
	withTempHome(t)
	if err := config.EnsureDirs(); err != nil {
		t.Fatalf("EnsureDirs() = %v", err)
	}

	f := openSupervisorLog()
	if f == nil {
		t.Fatal("openSupervisorLog() = nil, want a file handle")
	}
	defer func() { _ = f.Close() }()

	if _, err := f.WriteString("hello supervisor\n"); err != nil {
		t.Fatalf("write to log = %v", err)
	}
	if err := f.Sync(); err != nil {
		t.Fatalf("sync = %v", err)
	}

	data, err := os.ReadFile(config.SupervisorLogPath())
	if err != nil {
		t.Fatalf("read back log = %v", err)
	}
	if string(data) != "hello supervisor\n" {
		t.Errorf("log contents = %q, want %q", string(data), "hello supervisor\n")
	}

	if runtime.GOOS != "windows" {
		info, err := os.Stat(config.SupervisorLogPath())
		if err != nil {
			t.Fatalf("stat log = %v", err)
		}
		if perm := info.Mode().Perm(); perm != 0o600 {
			t.Errorf("log perm = %o, want 600", perm)
		}
	}
}

func TestOpenSupervisorLog_FallsBackToNilWhenDirMissing(t *testing.T) {
	withTempHome(t)
	// Do NOT create dirs: LogDir() does not exist, so the open must fail and
	// the helper must degrade to nil rather than erroring or panicking.
	if f := openSupervisorLog(); f != nil {
		_ = f.Close()
		t.Error("openSupervisorLog() = non-nil, want nil fallback when dir absent")
	}
}
