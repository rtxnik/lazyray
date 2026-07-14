package doctor

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCheckFilePermsFlagsLooseSettings(t *testing.T) {
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
