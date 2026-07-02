// internal/lifecycle/state_test.go
package lifecycle

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

// withTempData points config.DataDir at a temp dir for the duration of a test
// by setting HOME (Unix) and LOCALAPPDATA/APPDATA (Windows) so config.DataDir()
// and config.ConfigDir() resolve under the temp tree on all platforms.
func withTempData(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(dir, ".config"))
	t.Setenv("LOCALAPPDATA", filepath.Join(dir, "AppData", "Local"))
	t.Setenv("APPDATA", filepath.Join(dir, "AppData", "Roaming"))
	return dir
}

func TestWriteReadState_RoundTrip(t *testing.T) {
	withTempData(t)
	in := &State{
		Owner:         OwnerDaemon,
		SupervisorPID: 111,
		XrayPID:       222,
		StartedAt:     time.Now().UTC().Truncate(time.Second),
		SocksPort:     10808,
		HTTPPort:      10809,
		Routing:       Routing{SystemProxy: true},
	}
	if err := WriteState(in); err != nil {
		t.Fatalf("WriteState() error = %v", err)
	}
	out, err := ReadState()
	if err != nil {
		t.Fatalf("ReadState() error = %v", err)
	}
	if out == nil {
		t.Fatal("ReadState() returned nil, want state")
	}
	if out.Owner != OwnerDaemon || out.SupervisorPID != 111 || out.XrayPID != 222 {
		t.Errorf("roundtrip mismatch: %+v", out)
	}
	if !out.Routing.SystemProxy {
		t.Error("Routing.SystemProxy = false, want true")
	}
}

func TestReadState_AbsentReturnsNil(t *testing.T) {
	withTempData(t)
	out, err := ReadState()
	if err != nil {
		t.Fatalf("ReadState() error = %v", err)
	}
	if out != nil {
		t.Errorf("ReadState() = %+v, want nil when absent", out)
	}
}

func TestWriteState_Perms0600(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX file permissions are not honored on Windows")
	}
	withTempData(t)
	if err := WriteState(&State{Owner: OwnerTUI}); err != nil {
		t.Fatalf("WriteState() error = %v", err)
	}
	info, err := os.Stat(StatePath())
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if info.Mode().Perm() != 0600 {
		t.Errorf("perm = %v, want -rw-------", info.Mode().Perm())
	}
}

// WriteState must replace state.json atomically (new file identity), never
// truncate it in place — so a crash mid-write cannot corrupt the runtime record.
func TestWriteState_AtomicReplace(t *testing.T) {
	withTempData(t)
	s := &State{Owner: OwnerDaemon, SupervisorPID: 1, XrayPID: 2}
	if err := WriteState(s); err != nil {
		t.Fatalf("first WriteState: %v", err)
	}
	before, err := os.Stat(StatePath())
	if err != nil {
		t.Fatalf("stat before: %v", err)
	}
	s.XrayPID = 3
	if err := WriteState(s); err != nil {
		t.Fatalf("second WriteState: %v", err)
	}
	after, err := os.Stat(StatePath())
	if err != nil {
		t.Fatalf("stat after: %v", err)
	}
	// The identity proxy (see fsutil.TestWriteFile_ReplacesRatherThanTruncates) is
	// only meaningful on Unix; the rename is atomic on Windows regardless.
	if runtime.GOOS != "windows" && os.SameFile(before, after) {
		t.Error("state.json kept the same identity — it was truncated in place, not atomically replaced")
	}
	if runtime.GOOS != "windows" && after.Mode().Perm() != 0o600 {
		t.Errorf("state.json perm = %o, want 0600", after.Mode().Perm())
	}
}

func TestRemoveState_Idempotent(t *testing.T) {
	withTempData(t)
	if err := RemoveState(); err != nil {
		t.Errorf("RemoveState() on absent = %v, want nil", err)
	}
	_ = WriteState(&State{Owner: OwnerDaemon})
	if err := RemoveState(); err != nil {
		t.Errorf("RemoveState() = %v, want nil", err)
	}
	if _, err := os.Stat(StatePath()); !os.IsNotExist(err) {
		t.Error("state file still present after RemoveState")
	}
}
