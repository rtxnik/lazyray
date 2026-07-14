package doctor

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// skipNonPOSIX skips a permission-behavior test on Windows, where checkFilePerms
// returns Info and Go does not model 0600/0644 as POSIX bits.
func skipNonPOSIX(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("POSIX file permissions")
	}
}

func TestCheckFilePermsFlagsLooseSettings(t *testing.T) {
	skipNonPOSIX(t)
	dir := t.TempDir()
	loose := filepath.Join(dir, "lazyray.yaml")
	if err := os.WriteFile(loose, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	env := &Env{GOOS: "linux", Stat: os.Stat, SettingsPath: loose}
	r := checkFilePerms(context.Background(), env)
	if r.Severity != SeverityWarn {
		t.Fatalf("severity = %v, want Warn", r.Severity)
	}
	if !strings.Contains(r.Detail, "lazyray.yaml") {
		t.Fatalf("detail did not name the loose file: %q", r.Detail)
	}
}

func TestCheckFilePermsFlagsLooseBackupArchive(t *testing.T) {
	skipNonPOSIX(t)
	backupDir := t.TempDir()
	arch := filepath.Join(backupDir, "lazyray-backup-x.tar.gz.enc")
	if err := os.WriteFile(arch, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	env := &Env{GOOS: "linux", Stat: os.Stat, BackupDir: backupDir}
	r := checkFilePerms(context.Background(), env)
	if r.Severity != SeverityWarn {
		t.Fatalf("severity = %v, want Warn", r.Severity)
	}
	if !strings.Contains(r.Detail, "lazyray-backup-x.tar.gz.enc") {
		t.Fatalf("detail did not name the loose archive: %q", r.Detail)
	}
}

func TestCheckFilePermsTightTreeOK(t *testing.T) {
	skipNonPOSIX(t)
	dir := t.TempDir()
	if err := os.Chmod(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	tight := filepath.Join(dir, "servers.yaml")
	if err := os.WriteFile(tight, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	env := &Env{GOOS: "linux", Stat: os.Stat, ServersPath: tight, ConfigDir: dir}
	r := checkFilePerms(context.Background(), env)
	if r.Severity != SeverityOK {
		t.Fatalf("severity = %v, want OK (detail: %s)", r.Severity, r.Detail)
	}
}
