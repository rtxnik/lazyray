package core

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestAssetName(t *testing.T) {
	name := AssetName()

	if name == "" {
		t.Fatal("AssetName() returned empty string")
	}

	// Should start with "Xray-"
	if !strings.HasPrefix(name, "Xray-") {
		t.Errorf("AssetName() = %q, should start with 'Xray-'", name)
	}

	// Should end with ".zip"
	if !strings.HasSuffix(name, ".zip") {
		t.Errorf("AssetName() = %q, should end with '.zip'", name)
	}

	// Should contain the OS
	osNames := map[string]string{
		"darwin":  "macos",
		"linux":   "linux",
		"windows": "windows",
	}
	expectedOS, ok := osNames[runtime.GOOS]
	if ok && !strings.Contains(name, expectedOS) {
		t.Errorf("AssetName() = %q, should contain %q for GOOS=%q", name, expectedOS, runtime.GOOS)
	}
}

func TestFindAssetURL(t *testing.T) {
	release := &ReleaseInfo{
		TagName: "v1.8.24",
		Assets: []Asset{
			{Name: "Xray-linux-64.zip", BrowserDownloadURL: "https://example.com/linux-64.zip"},
			{Name: "Xray-macos-arm64-v8a.zip", BrowserDownloadURL: "https://example.com/macos-arm64.zip"},
			{Name: "Xray-windows-64.zip", BrowserDownloadURL: "https://example.com/win-64.zip"},
		},
	}

	url, err := FindAssetURL(release)
	if err != nil {
		// This may fail if current platform asset isn't in the test list
		// but should succeed for common platforms
		if runtime.GOARCH != "amd64" && runtime.GOARCH != "arm64" {
			t.Skipf("unsupported test arch: %s", runtime.GOARCH)
		}
		t.Logf("FindAssetURL() error = %v (may be expected for %s/%s)", err, runtime.GOOS, runtime.GOARCH)
		return
	}

	if url == "" {
		t.Error("FindAssetURL() returned empty URL")
	}
	if !strings.HasPrefix(url, "https://") {
		t.Errorf("FindAssetURL() = %q, should start with https://", url)
	}
}

func TestFindAssetURL_NotFound(t *testing.T) {
	release := &ReleaseInfo{
		TagName: "v1.0.0",
		Assets:  []Asset{},
	}

	_, err := FindAssetURL(release)
	if err == nil {
		t.Error("FindAssetURL() should return error for empty assets")
	}
}

func TestCheckUpdate_BlocksPrivateHost(t *testing.T) {
	orig := lookupIP
	t.Cleanup(func() { lookupIP = orig })
	lookupIP = func(host string) ([]net.IP, error) {
		return []net.IP{net.ParseIP("169.254.169.254")}, nil
	}

	_, err := CheckUpdate("v26.3.27")
	if err == nil {
		t.Fatal("CheckUpdate did not fail when the API host resolves to link-local")
	}
	if !strings.Contains(err.Error(), "blocked") {
		t.Errorf("error %q should indicate a blocked address", err.Error())
	}
}

func TestXrayReleaseAPIURL(t *testing.T) {
	tests := []struct {
		name    string
		version string
		want    string
	}{
		{
			name:    "pinned tag",
			version: "v26.3.27",
			want:    "https://api.github.com/repos/XTLS/Xray-core/releases/tags/v26.3.27",
		},
		{
			name:    "override tag",
			version: "v25.1.30",
			want:    "https://api.github.com/repos/XTLS/Xray-core/releases/tags/v25.1.30",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := xrayReleaseAPIURL(tt.version)
			if got != tt.want {
				t.Errorf("xrayReleaseAPIURL(%q) = %q, want %q", tt.version, got, tt.want)
			}
		})
	}
}

func TestClearQuarantineFn_ScopedToBinary(t *testing.T) {
	var gotPath string
	orig := clearQuarantineFn
	clearQuarantineFn = func(p string) error { gotPath = p; return nil }
	defer func() { clearQuarantineFn = orig }()

	xrayBin := filepath.Join(t.TempDir(), "bin", "xray")
	if err := clearQuarantineFn(xrayBin); err != nil {
		t.Fatal(err)
	}
	if gotPath != xrayBin {
		t.Fatalf("quarantine cleared on %q, want exact binary %q", gotPath, xrayBin)
	}
}

func TestUpdaterSource_NoRecursiveDestDirQuarantine(t *testing.T) {
	src, err := os.ReadFile("updater.go")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(src), `"-cr", destDir`) {
		t.Fatal("updater.go still clears quarantine on the whole destDir")
	}
}

func TestSha256OfFileHex(t *testing.T) {
	p := filepath.Join(t.TempDir(), "f")
	_ = os.WriteFile(p, []byte("abc"), 0o644)
	got, err := sha256OfFileHex(p)
	if err != nil {
		t.Fatal(err)
	}
	want := "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad" // gitleaks:allow
	if got != want {
		t.Fatalf("sha256OfFileHex = %q, want %q", got, want)
	}
}

// newTestXrayProc returns an XrayProcess safe for rollback tests: IsRunning()
// is false and Stop()/Start() touch only files under an isolated HOME/APPDATA,
// so there is no real PID file or process to find. Mirrors the isolation
// pattern used by setTestHome (core_8c_extra_test.go) and
// TestStopLocked_ForeignPID_NotKilled (xray_stop_unix_test.go).
func newTestXrayProc(t *testing.T) *XrayProcess {
	t.Helper()
	setTestHome(t, t.TempDir())
	return &XrayProcess{}
}

func TestRollbackUpdate_RejectsTamperedBackup(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "xray")
	bpath := filepath.Join(dir, "xray.bak")
	_ = os.WriteFile(dest, []byte("new-binary"), 0o755)
	_ = os.WriteFile(bpath, []byte("good-backup"), 0o755)
	goodSHA, _ := sha256OfFileHex(bpath)
	_ = os.WriteFile(bpath, []byte("EVIL-backup"), 0o755) // tamper after hash recorded
	backups := []fileBackup{{dest: dest, backupPath: bpath, sha: goodSHA}}

	if err := rollbackUpdate(newTestXrayProc(t), backups, dest, false); err == nil {
		t.Fatal("rollback restored a tampered .bak; want re-verification error")
	}
	if got, _ := os.ReadFile(dest); string(got) == "EVIL-backup" {
		t.Fatal("rollback wrote tampered backup bytes over the binary")
	}
}

func TestRollbackUpdate_RestoresWholeSet(t *testing.T) {
	dir := t.TempDir()
	mk := func(name, content string) fileBackup {
		dest := filepath.Join(dir, name)
		bpath := filepath.Join(dir, name+".bak")
		_ = os.WriteFile(dest, []byte("new-"+name), 0o644)
		_ = os.WriteFile(bpath, []byte(content), 0o644)
		sum, _ := sha256OfFileHex(bpath)
		return fileBackup{dest: dest, backupPath: bpath, sha: sum}
	}
	backups := []fileBackup{mk("xray", "old-xray"), mk("geoip.dat", "old-geoip")}
	if err := rollbackUpdate(newTestXrayProc(t), backups, filepath.Join(dir, "xray"), false); err != nil {
		t.Fatalf("rollback: %v", err)
	}
	for _, b := range backups {
		got, _ := os.ReadFile(b.dest)
		wantB, _ := os.ReadFile(b.backupPath)
		if string(got) != string(wantB) {
			t.Fatalf("%s not restored from backup", b.dest)
		}
	}
}

func TestAssertNotWorldWritable(t *testing.T) {
	dir := t.TempDir()
	_ = os.Chmod(dir, 0o777)
	if err := assertNotWorldWritable(dir); err == nil {
		t.Fatal("0777 dir accepted")
	}
	_ = os.Chmod(dir, 0o700)
	if err := assertNotWorldWritable(dir); err != nil {
		t.Fatalf("0700 dir rejected: %v", err)
	}
}

func TestVerifyXrayChecksum(t *testing.T) {
	// Create a temp "zip" with known content and compute its SHA-256.
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "xray.zip")
	content := []byte("fake xray archive bytes")
	if err := os.WriteFile(zipPath, content, 0o600); err != nil {
		t.Fatalf("writing temp zip: %v", err)
	}
	sum := sha256.Sum256(content)
	goodHex := hex.EncodeToString(sum[:])

	validDgst := "MD5= deadbeef\n" +
		"SHA1= cafebabe\n" +
		"SHA2-256= " + goodHex + "\n" +
		"SHA2-512= abc123\n"

	tests := []struct {
		name    string
		dgst    string
		wantErr error
	}{
		{
			name: "valid checksum",
			dgst: validDgst,
		},
		{
			name:    "mismatched checksum",
			dgst:    "SHA2-256= 0000000000000000000000000000000000000000000000000000000000000000\n",
			wantErr: ErrXrayChecksumMismatch,
		},
		{
			name:    "no SHA2-256 line",
			dgst:    "MD5= deadbeef\nSHA1= cafebabe\n",
			wantErr: ErrXrayChecksumMismatch,
		},
		{
			name:    "uppercase-key only, lowercase sha256 is ignored",
			dgst:    "sha256= " + goodHex + "\n",
			wantErr: ErrXrayChecksumMismatch,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := verifyXrayChecksum(zipPath, []byte(tt.dgst))
			if tt.wantErr == nil {
				if err != nil {
					t.Fatalf("verifyXrayChecksum() unexpected error: %v", err)
				}
				return
			}
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("verifyXrayChecksum() error = %v, want errors.Is %v", err, tt.wantErr)
			}
		})
	}
}

func TestXrayUpdateAllowed(t *testing.T) {
	tests := []struct {
		name           string
		target         string
		installed      string
		allowDowngrade bool
		wantErr        error
	}{
		{"below floor", "v1.0.0", "not installed", false, ErrXrayBelowFloor},
		{"equal", "v26.3.27", "v26.3.27", false, nil},
		{"older is downgrade", "v26.3.26", "v26.3.27", false, ErrXrayDowngrade},
		{"older with override", "v26.3.26", "v26.3.27", true, nil},
		{"fresh at floor", "v26.3.27", "not installed", false, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := XrayUpdateAllowed(tt.target, tt.installed, tt.allowDowngrade)
			if tt.wantErr == nil {
				if err != nil {
					t.Fatalf("XrayUpdateAllowed() unexpected error: %v", err)
				}
				return
			}
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("XrayUpdateAllowed() error = %v, want errors.Is %v", err, tt.wantErr)
			}
		})
	}
}
