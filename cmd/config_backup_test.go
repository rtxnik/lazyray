package cmd

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/rtxnik/lazyray/internal/config"
)

// withNoEncrypt flips the backup --no-encrypt flag for one test.
func withNoEncrypt(t *testing.T) {
	t.Helper()
	backupNoEncrypt = true
	t.Cleanup(func() { backupNoEncrypt = false })
}

// saveMarkerProfile stores a profile whose UUID is a recognizable plaintext
// marker, through the real save path.
func saveMarkerProfile(t *testing.T) {
	t.Helper()
	servers := &config.ServersConfig{Profiles: []config.Profile{{
		Name:    "marker",
		Default: true,
		Server: config.ServerConfig{
			Protocol: "vless",
			Address:  "marker.example.org",
			Port:     443,
			UUID:     "SECRET-UUID-MARKER",
		},
	}}}
	if err := config.SaveServers(servers); err != nil {
		t.Fatalf("SaveServers: %v", err)
	}
}

// plainArchive builds an in-memory tar.gz with a single named member.
func plainArchive(t *testing.T, member, body string) []byte {
	t.Helper()
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)
	if err := tw.WriteHeader(&tar.Header{Name: member, Mode: 0600, Size: int64(len(body))}); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write([]byte(body)); err != nil {
		t.Fatal(err)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// SEC3-01 regression: plaintext archives must carry no group/other bits.
func TestBackup_NoEncrypt_Mode0600(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX file permissions are not honored on Windows")
	}
	cleanup := setupTestHome(t)
	defer cleanup()
	clearPassphraseEnv(t)
	saveMarkerProfile(t)
	withNoEncrypt(t)

	outPath := filepath.Join(t.TempDir(), "backup.tar.gz")
	if err := configBackupCmd.RunE(configBackupCmd, []string{outPath}); err != nil {
		t.Fatalf("backup: %v", err)
	}
	info, err := os.Stat(outPath)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm()&0o077 != 0 {
		t.Errorf("archive mode = %v, want no group/other bits", info.Mode().Perm())
	}
}

// SEC3-01: the default (encrypted) backup must not expose plaintext secrets.
func TestBackup_DefaultEncrypted_NoPlaintextSecrets(t *testing.T) {
	cleanup := setupTestHome(t)
	defer cleanup()
	saveMarkerProfile(t)
	t.Setenv(passphraseEnvVar, "backup-pass")

	outPath := filepath.Join(t.TempDir(), "backup.bin")
	if err := configBackupCmd.RunE(configBackupCmd, []string{outPath}); err != nil {
		t.Fatalf("backup: %v", err)
	}
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(strings.TrimSpace(string(data)), "LZRENC2:") {
		t.Errorf("default backup should be an LZRENC2 blob, got prefix %q", string(data[:min(len(data), 12)]))
	}
	if strings.Contains(string(data), "SECRET-UUID-MARKER") {
		t.Error("plaintext secret leaked into the default backup")
	}
}

// SEC4-01 regression: restore must replace a planted symlink, not write
// through it.
func TestRestore_DoesNotFollowSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink semantics differ on Windows")
	}
	cleanup := setupTestHome(t)
	defer cleanup()
	clearPassphraseEnv(t)
	if err := config.EnsureDirs(); err != nil {
		t.Fatal(err)
	}

	home, _ := os.UserHomeDir()
	victim := filepath.Join(home, "victim")
	if err := os.WriteFile(victim, []byte("ORIGINAL"), 0o600); err != nil {
		t.Fatal(err)
	}
	os.Remove(config.ServersPath())
	if err := os.Symlink(victim, config.ServersPath()); err != nil {
		t.Fatal(err)
	}

	archivePath := filepath.Join(t.TempDir(), "planted.tar.gz")
	if err := os.WriteFile(archivePath, plainArchive(t, "servers.yaml", "PWNED_BY_RESTORE"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := configRestoreCmd.RunE(configRestoreCmd, []string{archivePath}); err != nil {
		t.Fatalf("restore: %v", err)
	}

	got, err := os.ReadFile(victim)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "ORIGINAL" {
		t.Errorf("victim content = %q, want ORIGINAL (restore wrote through the symlink)", got)
	}
	fi, err := os.Lstat(config.ServersPath())
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		t.Error("servers.yaml is still a symlink; restore should have replaced it with a regular file")
	}
}

// Encrypted backup → restore round-trip through the real command path.
func TestBackupRestore_EncryptedRoundTrip(t *testing.T) {
	cleanup := setupTestHome(t)
	defer cleanup()
	saveMarkerProfile(t)
	t.Setenv(passphraseEnvVar, "roundtrip-pass")

	outPath := filepath.Join(t.TempDir(), "backup.enc")
	if err := configBackupCmd.RunE(configBackupCmd, []string{outPath}); err != nil {
		t.Fatalf("backup: %v", err)
	}

	original, err := os.ReadFile(config.ServersPath())
	if err != nil {
		t.Fatal(err)
	}
	os.Remove(config.ServersPath())

	if err := configRestoreCmd.RunE(configRestoreCmd, []string{outPath}); err != nil {
		t.Fatalf("restore: %v", err)
	}
	restored, err := os.ReadFile(config.ServersPath())
	if err != nil {
		t.Fatal(err)
	}
	if string(restored) != string(original) {
		t.Error("restored servers.yaml differs from the original")
	}
}

// Encryption is the default and fails closed without a passphrase source.
func TestBackup_FailClosedWithoutPassphraseSource(t *testing.T) {
	cleanup := setupTestHome(t)
	defer cleanup()
	clearPassphraseEnv(t)
	fakeTerminal(t, false)
	saveMarkerProfile(t)

	err := configBackupCmd.RunE(configBackupCmd, []string{filepath.Join(t.TempDir(), "b")})
	if err == nil {
		t.Fatal("backup without a passphrase source must fail")
	}
	for _, want := range []string{"--no-encrypt", passphraseEnvVar, "--passphrase-file"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q should mention %s", err, want)
		}
	}
}

// --passphrase-file works for both directions.
func TestBackupRestore_PassphraseFile(t *testing.T) {
	cleanup := setupTestHome(t)
	defer cleanup()
	clearPassphraseEnv(t)
	saveMarkerProfile(t)

	passFile := filepath.Join(t.TempDir(), "pass")
	if err := os.WriteFile(passFile, []byte("pf-secret\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	backupPassphraseFile = passFile
	restorePassphraseFile = passFile
	t.Cleanup(func() { backupPassphraseFile = ""; restorePassphraseFile = "" })

	outPath := filepath.Join(t.TempDir(), "backup.enc")
	if err := configBackupCmd.RunE(configBackupCmd, []string{outPath}); err != nil {
		t.Fatalf("backup: %v", err)
	}
	os.Remove(config.ServersPath())
	if err := configRestoreCmd.RunE(configRestoreCmd, []string{outPath}); err != nil {
		t.Fatalf("restore: %v", err)
	}
	if _, err := os.Stat(config.ServersPath()); err != nil {
		t.Error("servers.yaml should be restored")
	}
}

// Default filenames reflect the content: .tar.gz.enc vs .tar.gz.
func TestBackup_DefaultFilenameExtension(t *testing.T) {
	cleanup := setupTestHome(t)
	defer cleanup()
	saveMarkerProfile(t)
	t.Setenv(passphraseEnvVar, "ext-pass")

	if err := configBackupCmd.RunE(configBackupCmd, []string{}); err != nil {
		t.Fatalf("backup: %v", err)
	}
	enc, _ := filepath.Glob(filepath.Join(config.BackupDir(), "lazyray-backup-*.tar.gz.enc"))
	if len(enc) != 1 {
		t.Errorf("want exactly one .tar.gz.enc default backup, got %v", enc)
	}

	withNoEncrypt(t)
	if err := configBackupCmd.RunE(configBackupCmd, []string{}); err != nil {
		t.Fatalf("backup --no-encrypt: %v", err)
	}
	plain, _ := filepath.Glob(filepath.Join(config.BackupDir(), "lazyray-backup-*.tar.gz"))
	if len(plain) != 1 {
		t.Errorf("want exactly one plain .tar.gz default backup, got %v", plain)
	}
}

// --no-encrypt with an explicit passphrase source is a contradiction, not a
// silent plaintext downgrade.
func TestBackup_NoEncryptExcludesPassphraseFile(t *testing.T) {
	if err := configBackupCmd.Flags().Set("no-encrypt", "true"); err != nil {
		t.Fatal(err)
	}
	if err := configBackupCmd.Flags().Set("passphrase-file", "/tmp/whatever"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = configBackupCmd.Flags().Set("no-encrypt", "false")
		_ = configBackupCmd.Flags().Set("passphrase-file", "")
		backupNoEncrypt = false
		backupPassphraseFile = ""
		configBackupCmd.Flags().Lookup("no-encrypt").Changed = false
		configBackupCmd.Flags().Lookup("passphrase-file").Changed = false
	})
	if err := configBackupCmd.ValidateFlagGroups(); err == nil {
		t.Fatal("no-encrypt + passphrase-file must be mutually exclusive")
	}
}

// Wrong passphrase fails without touching the current config.
func TestRestore_WrongPassphraseTouchesNothing(t *testing.T) {
	cleanup := setupTestHome(t)
	defer cleanup()
	saveMarkerProfile(t)
	t.Setenv(passphraseEnvVar, "right-pass")

	outPath := filepath.Join(t.TempDir(), "backup.enc")
	if err := configBackupCmd.RunE(configBackupCmd, []string{outPath}); err != nil {
		t.Fatalf("backup: %v", err)
	}
	before, _ := os.ReadFile(config.ServersPath())

	t.Setenv(passphraseEnvVar, "wrong-pass")
	if err := configRestoreCmd.RunE(configRestoreCmd, []string{outPath}); err == nil {
		t.Fatal("restore with wrong passphrase must fail")
	}
	after, _ := os.ReadFile(config.ServersPath())
	if string(before) != string(after) {
		t.Error("failed restore must not modify existing config")
	}
}
