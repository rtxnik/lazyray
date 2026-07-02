package core

import (
	"archive/zip"
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/rtxnik/lazyray/internal/config"
	"github.com/rtxnik/lazyray/internal/platform"
)

// xrayReleaseAPIBase is the GitHub releases API prefix for xray-core. The pinned
// tag (settings.Update.XrayVersion, e.g. "v26.3.27") is appended by
// xrayReleaseAPIURL to fetch /releases/tags/<tag> instead of /releases/latest,
// so updates adopt a known version rather than silently tracking upstream HEAD.
const xrayReleaseAPIBase = "https://api.github.com/repos/XTLS/Xray-core/releases/tags/"

// xrayReleaseAPIURL builds the release-by-tag API URL for the given xray-core
// version tag (the leading "v" is part of the tag and is preserved).
func xrayReleaseAPIURL(version string) string {
	return xrayReleaseAPIBase + version
}

// ErrXrayChecksumMismatch is returned when the downloaded xray-core archive does
// not match the SHA2-256 value published in its .dgst checksum file, or when the
// .dgst file has no parseable SHA2-256 line. Fail closed: the archive is never
// extracted or executed.
var ErrXrayChecksumMismatch = errors.New("xray-core archive checksum mismatch")

// dgstAssetName returns the .dgst checksum filename for the current platform's
// xray-core asset (e.g. "Xray-linux-64.zip.dgst").
func dgstAssetName() string {
	return AssetName() + ".dgst"
}

// parseDgstSHA256 extracts the lowercase hex SHA2-256 value from an XTLS .dgst
// file. Lines look like "SHA2-256= <hex>" (note the space after '='); the key is
// matched case-sensitively against "SHA2-256" so a lowercase "sha256" key is NOT
// accepted. Returns false if no SHA2-256 line is present.
func parseDgstSHA256(dgst []byte) (string, bool) {
	scanner := bufio.NewScanner(bytes.NewReader(dgst))
	for scanner.Scan() {
		line := scanner.Text()
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		if strings.TrimSpace(key) != "SHA2-256" {
			continue
		}
		return strings.ToLower(strings.TrimSpace(value)), true
	}
	return "", false
}

// verifyXrayChecksum computes the SHA-256 of the file at zipPath and compares it
// to the SHA2-256 value parsed from the supplied .dgst content. Any failure —
// missing/garbled SHA2-256 line, read error, or hash mismatch — returns
// ErrXrayChecksumMismatch so the caller fails closed.
func verifyXrayChecksum(zipPath string, dgst []byte) error {
	want, ok := parseDgstSHA256(dgst)
	if !ok || want == "" {
		return fmt.Errorf("%w: no SHA2-256 entry in checksum file", ErrXrayChecksumMismatch)
	}

	f, err := os.Open(zipPath)
	if err != nil {
		return fmt.Errorf("opening archive for checksum: %w", err)
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return fmt.Errorf("hashing archive: %w", err)
	}
	got := hex.EncodeToString(h.Sum(nil))

	if got != want {
		return fmt.Errorf("%w: have %s, want %s", ErrXrayChecksumMismatch, got, want)
	}
	return nil
}

// ReleaseInfo contains GitHub release information.
type ReleaseInfo struct {
	TagName string  `json:"tag_name"`
	Assets  []Asset `json:"assets"`
}

// Asset represents a release asset.
type Asset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

// CheckUpdate fetches the xray-core release pinned to the given version tag
// (e.g. "v26.3.27") via /releases/tags/<version>.
func CheckUpdate(version string) (*ReleaseInfo, error) {
	client := directClient(15 * time.Second)
	resp, err := safeGet(context.Background(), client, xrayReleaseAPIURL(version), 1<<20)
	if err != nil {
		return nil, fmt.Errorf("checking for updates: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var release ReleaseInfo
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("parsing release info: %w", err)
	}

	return &release, nil
}

// AssetName returns the expected asset filename for the current platform.
func AssetName() string {
	switch {
	case runtime.GOOS == "darwin" && runtime.GOARCH == "arm64":
		return "Xray-macos-arm64-v8a.zip"
	case runtime.GOOS == "darwin" && runtime.GOARCH == "amd64":
		return "Xray-macos-64.zip"
	case runtime.GOOS == "linux" && runtime.GOARCH == "amd64":
		return "Xray-linux-64.zip"
	case runtime.GOOS == "linux" && runtime.GOARCH == "arm64":
		return "Xray-linux-arm64-v8a.zip"
	case runtime.GOOS == "windows" && runtime.GOARCH == "amd64":
		return "Xray-windows-64.zip"
	case runtime.GOOS == "windows" && runtime.GOARCH == "arm64":
		return "Xray-windows-arm64-v8a.zip"
	default:
		return ""
	}
}

// FindAssetURL finds the download URL for the current platform.
func FindAssetURL(release *ReleaseInfo) (string, error) {
	target := AssetName()
	if target == "" {
		return "", fmt.Errorf("unsupported platform: %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	for _, asset := range release.Assets {
		if asset.Name == target {
			return asset.BrowserDownloadURL, nil
		}
	}

	return "", fmt.Errorf("asset %s not found in release %s", target, release.TagName)
}

// findDgstURL resolves the download URL of the current platform's xray-core
// .dgst checksum asset (e.g. "Xray-linux-64.zip.dgst") within the given release.
func findDgstURL(release *ReleaseInfo) (string, error) {
	target := dgstAssetName()
	if target == ".dgst" {
		return "", fmt.Errorf("unsupported platform: %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	for _, asset := range release.Assets {
		if asset.Name == target {
			return asset.BrowserDownloadURL, nil
		}
	}

	return "", fmt.Errorf("checksum asset %s not found in release %s", target, release.TagName)
}

// ApplyUpdate downloads and installs a new xray binary. Before extracting or
// running anything, it verifies the downloaded archive against the SHA2-256 in
// the XTLS-published .dgst checksum file from the same release (fail closed:
// ErrXrayChecksumMismatch). After restarting, it runs a minimal health check
// (process + port). If the check fails, it rolls back to the backup binary and
// notifies the user.
func ApplyUpdate(xrayProc *XrayProcess, release *ReleaseInfo, downloadURL string, backup bool) error {
	if err := config.EnsureDirs(); err != nil {
		return err
	}

	xrayBin := config.XrayBinaryPath()

	// Backup current binary
	var backupPath string
	if backup {
		if _, err := os.Stat(xrayBin); err == nil {
			backupName := fmt.Sprintf("xray.%s.bak", time.Now().Format("20060102-150405"))
			backupPath = filepath.Join(config.BackupDir(), backupName)
			if err := copyFile(xrayBin, backupPath); err != nil {
				return fmt.Errorf("creating backup: %w", err)
			}
		}
	}

	// Download to temp file
	tmpDir, err := os.MkdirTemp("", "lazyray-update-*")
	if err != nil {
		return fmt.Errorf("creating temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	zipPath := filepath.Join(tmpDir, "xray.zip")
	if err := downloadFile(downloadURL, zipPath); err != nil {
		return fmt.Errorf("downloading: %w", err)
	}

	// Verify the downloaded archive against the published .dgst SHA2-256 BEFORE
	// extracting, copying, or chmod-ing anything. Fail closed on any mismatch.
	dgstURL, err := findDgstURL(release)
	if err != nil {
		return err
	}
	dgstPath := filepath.Join(tmpDir, "xray.zip.dgst")
	if err := downloadFile(dgstURL, dgstPath); err != nil {
		return fmt.Errorf("downloading checksum: %w", err)
	}
	dgst, err := os.ReadFile(dgstPath)
	if err != nil {
		return fmt.Errorf("reading checksum: %w", err)
	}
	if err := verifyXrayChecksum(zipPath, dgst); err != nil {
		return err
	}

	// Extract xray binary and data files from zip
	xrayName := "xray"
	if runtime.GOOS == "windows" {
		xrayName = "xray.exe"
	}

	filesToExtract := []string{xrayName, "geoip.dat", "geosite.dat"}
	for _, name := range filesToExtract {
		extractedPath := filepath.Join(tmpDir, name)
		if err := extractFromZip(zipPath, name, extractedPath); err != nil {
			return fmt.Errorf("extracting %s: %w", name, err)
		}
	}

	// Stop xray if running
	wasRunning := xrayProc.IsRunning()
	if wasRunning {
		if err := xrayProc.Stop(); err != nil {
			return fmt.Errorf("stopping xray before update: %w", err)
		}
	}

	// Replace binary and data files
	destDir := filepath.Dir(xrayBin)
	for _, name := range filesToExtract {
		src := filepath.Join(tmpDir, name)
		dst := filepath.Join(destDir, name)
		if err := copyFile(src, dst); err != nil {
			return fmt.Errorf("installing %s: %w", name, err)
		}
	}

	if err := os.Chmod(xrayBin, 0755); err != nil {
		return fmt.Errorf("setting permissions: %w", err)
	}

	// Remove macOS Gatekeeper quarantine attribute
	if runtime.GOOS == "darwin" {
		_ = exec.Command("xattr", "-cr", destDir).Run()
	}

	// Restart if was running and verify health
	if wasRunning {
		if err := xrayProc.Start(); err != nil {
			rollbackErr := rollbackUpdate(xrayProc, backupPath, xrayBin)
			if rollbackErr != nil {
				return fmt.Errorf("restart failed: %w (rollback also failed: %v)", err, rollbackErr)
			}
			return fmt.Errorf("restart failed after update, rolled back to previous version: %w", err)
		}

		// Verify health after update with 15 second timeout
		if err := verifyPostUpdate(xrayProc); err != nil {
			rollbackErr := rollbackUpdate(xrayProc, backupPath, xrayBin)
			if rollbackErr != nil {
				return fmt.Errorf("health check failed: %w (rollback also failed: %v)", err, rollbackErr)
			}
			return fmt.Errorf("health check failed after update, rolled back to previous version: %w", err)
		}
	}

	InvalidateXrayVersionCache()
	return nil
}

// verifyPostUpdate checks that xray is running and accepting connections
// on configured ports within a 15 second timeout.
func verifyPostUpdate(xrayProc *XrayProcess) error {
	settings, _ := config.LoadSettings()
	if settings == nil {
		settings = config.DefaultSettings()
	}

	deadline := time.Now().Add(15 * time.Second)

	// Wait for process to be alive
	for time.Now().Before(deadline) {
		if xrayProc.IsRunning() {
			break
		}
		time.Sleep(time.Second)
	}
	if !xrayProc.IsRunning() {
		return fmt.Errorf("xray process not running after 15s")
	}

	// Wait for SOCKS5 port to accept connections
	addr := net.JoinHostPort(settings.Local.Listen, strconv.Itoa(settings.Local.SocksPort))
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
		if err == nil {
			conn.Close()
			return nil
		}
		time.Sleep(time.Second)
	}

	return fmt.Errorf("SOCKS5 port %s not accepting connections after update", addr)
}

// rollbackUpdate stops xray, restores the backup binary, and restarts.
func rollbackUpdate(xrayProc *XrayProcess, backupPath, xrayBin string) error {
	if backupPath == "" {
		return fmt.Errorf("no backup available for rollback")
	}

	_ = xrayProc.Stop()

	if err := copyFile(backupPath, xrayBin); err != nil {
		return fmt.Errorf("restoring backup: %w", err)
	}
	if err := os.Chmod(xrayBin, 0755); err != nil {
		return fmt.Errorf("setting permissions on restored binary: %w", err)
	}

	if s, err := config.LoadSettings(); err == nil && s.Notifications.Enabled {
		_ = platform.Current().Notify("lazyray", "Xray update failed — rolled back to previous version")
	}

	if err := xrayProc.Start(); err != nil {
		return fmt.Errorf("restarting after rollback: %w", err)
	}
	return nil
}

func downloadFile(url, dest string) error {
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Duration(attempt*2) * time.Second)
		}
		if err := downloadFileOnce(url, dest); err != nil {
			lastErr = err
			continue
		}
		return nil
	}
	return fmt.Errorf("download failed after 3 attempts: %w", lastErr)
}

func downloadFileOnce(url, dest string) error {
	client := directClient(5 * time.Minute)
	resp, err := safeGet(context.Background(), client, url, 256<<20) // 256 MB cap for binary assets
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download returned status %d", resp.StatusCode)
	}

	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, resp.Body)
	return err
}

func extractFromZip(zipPath, targetName, dest string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		name := filepath.Base(f.Name)
		if strings.EqualFold(name, targetName) {
			rc, err := f.Open()
			if err != nil {
				return err
			}
			defer rc.Close()

			out, err := os.Create(dest)
			if err != nil {
				return err
			}
			defer out.Close()

			_, err = io.Copy(out, rc)
			return err
		}
	}

	return fmt.Errorf("%s not found in archive", targetName)
}

// copyFile copies src to dst atomically: it streams into a temp file in dst's
// directory, fsyncs it, then renames it into place — so a crash mid-copy leaves
// any previous dst intact instead of a half-written file. src may live on any
// filesystem; the temp + rename happen within dst's directory.
func copyFile(src, dst string) (err error) {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	mode := os.FileMode(0o644)
	if fi, statErr := in.Stat(); statErr == nil {
		mode = fi.Mode().Perm()
	}

	tmp, err := os.CreateTemp(filepath.Dir(dst), ".tmp-copy-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() {
		if err != nil {
			_ = tmp.Close()
			_ = os.Remove(tmpName)
		}
	}()

	if _, err = io.Copy(tmp, in); err != nil {
		return err
	}
	if err = tmp.Sync(); err != nil {
		return err
	}
	if err = tmp.Chmod(mode); err != nil {
		return err
	}
	if err = tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, dst)
}
