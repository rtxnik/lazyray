package procutil

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestWritePIDFile_Perm0600(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX perms not applicable on Windows")
	}
	p := filepath.Join(t.TempDir(), "pid")
	if err := WritePIDFile(p, []byte("4242\n8080")); err != nil {
		t.Fatalf("WritePIDFile = %v", err)
	}
	fi, err := os.Stat(p)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if fi.Mode().Perm() != 0o600 {
		t.Errorf("perm = %o, want 600", fi.Mode().Perm())
	}
	b, _ := os.ReadFile(p)
	if string(b) != "4242\n8080" {
		t.Errorf("content = %q, want the two-line PID\\nport body", string(b))
	}
}

func TestWritePIDFile_SurfacesError(t *testing.T) {
	// A path whose parent does not exist forces fsutil's atomic write to fail.
	p := filepath.Join(t.TempDir(), "no-such-dir", "pid")
	if err := WritePIDFile(p, []byte("1")); err == nil {
		t.Error("WritePIDFile to a missing dir returned nil, want error")
	}
}
