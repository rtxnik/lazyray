package execsafe

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestSecureLookPathRejectsWorldWritableDir(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX writability model")
	}
	dir := t.TempDir()
	bin := filepath.Join(dir, "tool")
	if err := os.WriteFile(bin, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	// Mkdir mode is umask-masked; force world-writable explicitly.
	if err := os.Chmod(dir, 0o777); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	if _, err := SecureLookPath("tool"); err == nil {
		t.Fatal("accepted a tool in a world-writable dir")
	}
}

func TestSecureLookPathAcceptsTightDir(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX writability model")
	}
	dir := t.TempDir()
	if err := os.Chmod(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(dir, "tool")
	if err := os.WriteFile(bin, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	got, err := SecureLookPath("tool")
	if err != nil {
		t.Fatalf("rejected a tight dir: %v", err)
	}
	if got == "" {
		t.Fatal("empty path")
	}
}

func TestSecureLookPathAcceptsStickyWorldWritableDir(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX writability model")
	}
	dir := t.TempDir()
	// Sticky + world-writable (the /tmp pattern) is acceptable — only entry
	// owners can rename/delete, so it is not a free-interposition vector.
	if err := os.Chmod(dir, os.ModeSticky|0o777); err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(dir, "tool")
	if err := os.WriteFile(bin, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	if _, err := SecureLookPath("tool"); err != nil {
		t.Fatalf("rejected a sticky world-writable dir: %v", err)
	}
}

func TestSecureLookPathRejectsSymlinkFromWorldWritableDir(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX writability model")
	}
	tight := t.TempDir()
	if err := os.Chmod(tight, 0o755); err != nil {
		t.Fatal(err)
	}
	realBin := filepath.Join(tight, "tool")
	if err := os.WriteFile(realBin, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	evil := t.TempDir()
	if err := os.Chmod(evil, 0o777); err != nil {
		t.Fatal(err)
	}
	// A symlink in the world-writable dir points at the tight-dir binary.
	if err := os.Symlink(realBin, filepath.Join(evil, "tool")); err != nil {
		t.Skipf("symlink unsupported: %v", err)
	}
	t.Setenv("PATH", evil)
	_, err := SecureLookPath("tool")
	if err == nil {
		t.Fatal("accepted a symlink planted in a world-writable dir")
	}
	// Must reject because of the world-writable planting dir, not by accident.
	if !strings.Contains(err.Error(), evil) {
		t.Fatalf("rejected for the wrong reason (not the planting dir %s): %v", evil, err)
	}
}

func TestSecureLookPathAcceptsSymlinkedPathDir(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX writability model")
	}
	// Mimics usr-merge: the PATH entry is a symlink to the real tight dir. A
	// symlinked component's own lrwxrwxrwx mode must not trigger a false reject.
	real := t.TempDir()
	if err := os.Chmod(real, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(real, "tool"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(t.TempDir(), "bin")
	if err := os.Symlink(real, link); err != nil {
		t.Skipf("symlink unsupported: %v", err)
	}
	t.Setenv("PATH", link)
	if _, err := SecureLookPath("tool"); err != nil {
		t.Fatalf("rejected a binary reached via a symlinked PATH dir: %v", err)
	}
}
