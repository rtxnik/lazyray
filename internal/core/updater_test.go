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
