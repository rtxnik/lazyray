package fsutil_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/rtxnik/lazyray/internal/fsutil"
)

func TestWriteFile_ContentAndPerm(t *testing.T) {
	path := filepath.Join(t.TempDir(), "f.yaml")
	if err := fsutil.WriteFile(path, []byte("hello"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(got) != "hello" {
		t.Errorf("content = %q, want %q", got, "hello")
	}
	if runtime.GOOS != "windows" {
		fi, err := os.Stat(path)
		if err != nil {
			t.Fatalf("stat: %v", err)
		}
		if fi.Mode().Perm() != 0o600 {
			t.Errorf("perm = %o, want 0600", fi.Mode().Perm())
		}
	}
}

func TestWriteFile_OverwriteRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "f.yaml")
	if err := fsutil.WriteFile(path, []byte("first"), 0o600); err != nil {
		t.Fatalf("first write: %v", err)
	}
	if err := fsutil.WriteFile(path, []byte("second-longer"), 0o600); err != nil {
		t.Fatalf("second write: %v", err)
	}
	got, _ := os.ReadFile(path)
	if string(got) != "second-longer" {
		t.Errorf("content = %q, want %q", got, "second-longer")
	}
}

// A truncate-in-place writer keeps the SAME file identity (inode) across writes;
// an atomic temp+rename writer replaces it with a NEW file. Asserting the
// identity changes proves the destination is never truncated in place, so a
// crash mid-write can never leave a partial/empty file.
func TestWriteFile_ReplacesRatherThanTruncates(t *testing.T) {
	path := filepath.Join(t.TempDir(), "f.yaml")
	if err := fsutil.WriteFile(path, []byte("OLD"), 0o600); err != nil {
		t.Fatalf("seed: %v", err)
	}
	before, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat before: %v", err)
	}
	if err := fsutil.WriteFile(path, []byte("NEW"), 0o600); err != nil {
		t.Fatalf("rewrite: %v", err)
	}
	after, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat after: %v", err)
	}
	// os.SameFile compares the file identity (inode on Unix). An atomic
	// temp+rename gives the destination a NEW identity; a truncate-in-place
	// writer keeps the same one. Windows path-based os.Stat does not populate a
	// reliable identity, so this proxy is only meaningful on Unix — the rename is
	// still atomic on Windows (MoveFileEx), and content correctness is asserted
	// by the other tests regardless of platform.
	if runtime.GOOS != "windows" && os.SameFile(before, after) {
		t.Error("destination kept the same identity — it was truncated in place, not atomically replaced")
	}
}

// On failure (a temp file cannot be created because the parent dir is missing),
// any pre-existing destination must be left untouched.
func TestWriteFile_FailureLeavesNoPartial(t *testing.T) {
	missingDir := filepath.Join(t.TempDir(), "does-not-exist")
	path := filepath.Join(missingDir, "f.yaml")
	if err := fsutil.WriteFile(path, []byte("data"), 0o600); err == nil {
		t.Fatal("expected error writing into a missing directory, got nil")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("a file was created on the failure path: stat err = %v", err)
	}
}
