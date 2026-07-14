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
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/rtxnik/lazyray/internal/config"
	"github.com/rtxnik/lazyray/internal/platform"
	"github.com/rtxnik/lazyray/internal/procutil"
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

// ErrXrayBelowFloor / ErrXrayDowngrade gate xray installs at the primitive level
// so every caller (CLI, TUI, future) enforces the same policy.
var ErrXrayBelowFloor = errors.New("xray-core version below the minimum supported floor")
var ErrXrayDowngrade = errors.New("refusing to install an older xray-core (downgrade)")

// XrayUpdateAllowed enforces: target >= MinXrayVersion (hard floor, no override);
// and target not strictly older than the installed version unless allowDowngrade.
// A fresh/unknown install skips the downgrade comparison but still honors the floor.
func XrayUpdateAllowed(target, installed string, allowDowngrade bool) error {
	if compareVersions(target, MinXrayVersion) < 0 {
		return fmt.Errorf("%w: %s < %s", ErrXrayBelowFloor, target, MinXrayVersion)
	}
	switch installed {
	case "not installed", "unknown", "":
		return nil
	}
	if compareVersions(target, installed) < 0 && !allowDowngrade {
		return fmt.Errorf("%w: %s < installed %s", ErrXrayDowngrade, target, installed)
	}
	return nil
}

// clearQuarantineFn clears macOS Gatekeeper quarantine on the EXACT path given
// (never a directory). Overridable in tests. The platform impl is a no-op off
// darwin, so callers need no build-tag branch.
var clearQuarantineFn = func(path string) error { return platform.Current().ClearQuarantine(path) }

// fileBackup records a backed-up install file and its pre-swap SHA-256 so a
// rollback can restore the whole set and re-verify each member before exec.
type fileBackup struct {
	dest       string // final install path (destDir/<name>)
	backupPath string // saved copy under BackupDir()
	sha        string // sha256 of the file at backup time
}

// sha256OfFileHex returns the lowercase-hex SHA-256 of the file at path.
func sha256OfFileHex(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

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

// withRetry runs fn up to attempts times with linear backoff (backoff, 2*backoff,
// ...) between tries. Returns nil on the first success, else the last error.
func withRetry(attempts int, backoff time.Duration, fn func() error) error {
	var lastErr error
	for attempt := 0; attempt < attempts; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Duration(attempt) * backoff)
		}
		if err := fn(); err != nil {
			lastErr = err
			continue
		}
		return nil
	}
	return fmt.Errorf("failed after %d attempts: %w", attempts, lastErr)
}

// fetchRelease GETs a GitHub release payload (SSRF-guarded, 1 MB cap) and decodes
// it. ctxLabel preserves each caller's observable error wording.
func fetchRelease(url, ctxLabel string) (*ReleaseInfo, error) {
	client := directClient(15 * time.Second)
	resp, err := safeGet(context.Background(), client, url, 1<<20)
	if err != nil {
		return nil, fmt.Errorf("checking for %s: %w", ctxLabel, err)
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

// CheckUpdate fetches the xray-core release pinned to the given version tag
// (e.g. "v26.3.27") via /releases/tags/<version>.
func CheckUpdate(version string) (*ReleaseInfo, error) {
	return fetchRelease(xrayReleaseAPIURL(version), "updates")
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
// running anything, it verifies the downloaded archive against the embedded,
// lazyray-signed pin for the release (fail closed: ErrXrayChecksumMismatch);
// the XTLS-published .dgst checksum path is used only under
// --allow-unverified-xray, which is a corruption check, not a security
// guarantee. After restarting, it runs a minimal health check (process + port).
// If the check fails, it rolls back to the backup binary and notifies the user.
func ApplyUpdate(xrayProc *XrayProcess, release *ReleaseInfo, downloadURL string, backup bool, allowUnverified bool, allowDowngrade bool) error {
	if err := config.EnsureDirs(); err != nil {
		return err
	}
	if err := XrayUpdateAllowed(release.TagName, GetXrayVersion(), allowDowngrade); err != nil {
		return err
	}

	xrayBin := config.XrayBinaryPath()
	destDir := filepath.Dir(xrayBin)
	if err := assertNotWorldWritable(config.BackupDir()); err != nil {
		fmt.Fprintln(os.Stderr, "WARNING:", err)
	}
	if err := assertNotWorldWritable(destDir); err != nil {
		fmt.Fprintln(os.Stderr, "WARNING:", err)
	}

	tmpDir, err := os.MkdirTemp("", "lazyray-update-*")
	if err != nil {
		return fmt.Errorf("creating temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)
	zipPath := filepath.Join(tmpDir, "xray.zip")
	if err := downloadFile(downloadURL, zipPath); err != nil {
		return fmt.Errorf("downloading: %w", err)
	}

	// INTEGRITY: default path verifies against the embedded, lazyray-signed pin.
	// The escape hatch falls back to the co-downloaded .dgst (corruption check
	// only, NOT a security guarantee) and warns loudly.
	if allowUnverified {
		fmt.Fprintln(os.Stderr, "WARNING: --allow-unverified-xray: installing without pin verification (checksum-only, not a security guarantee)")
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
	} else {
		if err := verifyXrayAgainstPins(release.TagName, AssetName(), zipPath); err != nil {
			return err
		}
	}

	xrayName := "xray"
	if runtime.GOOS == "windows" {
		xrayName = "xray.exe"
	}
	filesToExtract := []string{xrayName, "geoip.dat", "geosite.dat"}
	for _, name := range filesToExtract {
		if err := extractFromZip(zipPath, name, filepath.Join(tmpDir, name)); err != nil {
			return fmt.Errorf("extracting %s: %w", name, err)
		}
	}

	// WHOLE-SET backup: copy first, then hash the BACKUP we just wrote (not the
	// live file) so the recorded SHA matches the backup's bytes even if the live
	// file changes concurrently.
	var backups []fileBackup
	if backup {
		for _, name := range filesToExtract {
			dst := filepath.Join(destDir, name)
			if _, statErr := os.Stat(dst); statErr != nil {
				continue
			}
			bpath := filepath.Join(config.BackupDir(), fmt.Sprintf("%s.%s.bak", name, time.Now().Format("20060102-150405")))
			sum, err := copyFileWithHash(dst, bpath)
			if err != nil {
				return fmt.Errorf("creating backup for %s: %w", name, err)
			}
			backups = append(backups, fileBackup{dest: dst, backupPath: bpath, sha: sum})
		}
	}

	// Stage all three next to their destinations WHILE xray still runs. Clean up
	// staged temps on any early exit so we never leave .tmp-set-* litter behind.
	staged := make(map[string]string, len(filesToExtract))
	cleanupStaged := func() {
		for _, tmp := range staged {
			_ = os.Remove(tmp)
		}
	}
	for _, name := range filesToExtract {
		tmp := filepath.Join(destDir, ".tmp-set-"+name)
		if err := copyFile(filepath.Join(tmpDir, name), tmp); err != nil {
			cleanupStaged()
			return fmt.Errorf("staging %s: %w", name, err)
		}
		staged[filepath.Join(destDir, name)] = tmp
	}

	wasRunning := xrayProc.IsRunning()
	if wasRunning {
		if err := xrayProc.Stop(); err != nil {
			cleanupStaged()
			return fmt.Errorf("stopping xray before update: %w", err)
		}
	}
	// AFTER the stop, ANY failure (rename or chmod) must roll the whole set back
	// and restart xray to its prior state — otherwise the engine is left dead in
	// a torn install. Flip data files first, the version-bearing binary last.
	swapErr := func() error {
		for _, dst := range []string{filepath.Join(destDir, "geoip.dat"), filepath.Join(destDir, "geosite.dat"), xrayBin} {
			tmp, ok := staged[dst]
			if !ok {
				continue
			}
			if err := os.Rename(tmp, dst); err != nil {
				return fmt.Errorf("installing %s: %w", filepath.Base(dst), err)
			}
		}
		if err := os.Chmod(xrayBin, 0o755); err != nil {
			return fmt.Errorf("setting permissions: %w", err)
		}
		return nil
	}()
	if swapErr != nil {
		cleanupStaged()
		if len(backups) > 0 {
			if rbErr := rollbackUpdate(xrayProc, backups, xrayBin, wasRunning); rbErr != nil {
				return fmt.Errorf("%w (rollback also failed: %v)", swapErr, rbErr)
			}
			return fmt.Errorf("update failed, rolled back to previous version: %w", swapErr)
		}
		// No backup material to roll back to (backups disabled). Best-effort
		// restart so a running engine is not left dead; the install may be
		// inconsistent, which the error makes explicit.
		if wasRunning {
			_ = xrayProc.Start()
		}
		return fmt.Errorf("update failed and no backup was available to roll back (backups disabled); attempted to restart xray, install may be inconsistent: %w", swapErr)
	}
	_ = clearQuarantineFn(xrayBin) // scoped, post-verify (no-op off darwin)

	// Restart if was running and verify health
	if wasRunning {
		if err := xrayProc.Start(); err != nil {
			rollbackErr := rollbackUpdate(xrayProc, backups, xrayBin, true)
			if rollbackErr != nil {
				return fmt.Errorf("restart failed: %w (rollback also failed: %v)", err, rollbackErr)
			}
			return fmt.Errorf("restart failed after update, rolled back to previous version: %w", err)
		}

		// Verify health after update with 15 second timeout
		if err := verifyPostUpdate(xrayProc); err != nil {
			rollbackErr := rollbackUpdate(xrayProc, backups, xrayBin, true)
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
		if procutil.Reachable(addr, 2*time.Second) == nil {
			return nil
		}
		time.Sleep(time.Second)
	}

	return fmt.Errorf("SOCKS5 port %s not accepting connections after update", addr)
}

// rollbackUpdate restores the WHOLE previously-installed set after a failed
// update. Each backup member is re-verified against its recorded SHA-256 before
// being restored, so a .bak tampered in the backup->rollback window is refused.
// xray is stopped before and restarted after the restore.
func rollbackUpdate(xrayProc *XrayProcess, backups []fileBackup, xrayBin string, restart bool) error {
	if len(backups) == 0 {
		return fmt.Errorf("no backup available for rollback")
	}
	_ = xrayProc.Stop()

	for _, b := range backups {
		got, err := sha256OfFileHex(b.backupPath)
		if err != nil {
			return fmt.Errorf("re-verifying backup %s: %w", filepath.Base(b.dest), err)
		}
		if got != b.sha {
			return fmt.Errorf("backup for %s failed re-verification; refusing to restore", filepath.Base(b.dest))
		}
	}
	for _, b := range backups {
		if err := copyFile(b.backupPath, b.dest); err != nil {
			return fmt.Errorf("restoring %s: %w", filepath.Base(b.dest), err)
		}
	}
	if err := os.Chmod(xrayBin, 0o755); err != nil {
		return fmt.Errorf("setting permissions on restored binary: %w", err)
	}
	if s, err := config.LoadSettings(); err == nil && s.Notifications.Enabled {
		_ = platform.Current().Notify("lazyray", "Xray update failed — rolled back to previous version")
	}
	if restart {
		if err := xrayProc.Start(); err != nil {
			return fmt.Errorf("restarting after rollback: %w", err)
		}
	}
	return nil
}

// assertNotWorldWritable returns an error if dir is group- or world-writable.
// The POSIX permission bits it inspects are not meaningful on Windows (Stat
// reports directories as 0o777 regardless of ACLs), so the check is skipped there.
func assertNotWorldWritable(dir string) error {
	if runtime.GOOS == "windows" {
		return nil
	}
	fi, err := os.Stat(dir)
	if err != nil {
		return nil
	}
	if fi.Mode().Perm()&0o022 != 0 {
		return fmt.Errorf("directory %s is group/world-writable (%o); tighten to 0700", dir, fi.Mode().Perm())
	}
	return nil
}

func downloadFile(url, dest string) error {
	return withRetry(3, 2*time.Second, func() error { return downloadFileOnce(url, dest) })
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

			if err := copyCapped(out, rc, maxXrayMemberBytes); err != nil {
				_ = out.Close()
				_ = os.Remove(dest) // drop the partial (possibly cap-sized) write
				return err
			}
			return nil
		}
	}

	return fmt.Errorf("%s not found in archive", targetName)
}

// maxXrayMemberBytes caps a single extracted archive member. Real members (the
// xray binary, geoip/geosite) are far smaller; the cap defends against a
// decompression bomb without trusting the archive's declared size.
const maxXrayMemberBytes = 512 << 20

// ErrXrayMemberTooLarge reports an archive member that exceeds the extract cap.
var ErrXrayMemberTooLarge = errors.New("archive member exceeds size cap")

// copyCapped copies src to dst, refusing more than cap bytes (decompression-bomb
// guard). It reads one byte past the cap so an exactly-cap member (ok) is
// distinguished from an oversize one (ErrXrayMemberTooLarge).
func copyCapped(dst io.Writer, src io.Reader, cap int64) error {
	n, err := io.CopyN(dst, src, cap+1)
	if errors.Is(err, io.EOF) {
		return nil // fit within the cap
	}
	if err != nil {
		return err
	}
	_ = n
	return ErrXrayMemberTooLarge
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

// copyFileWithHash copies src to dst (atomic temp+rename, like copyFile) and
// returns the lowercase-hex SHA-256 of the exact bytes written, closing the
// re-read TOCTOU window in backup fingerprinting.
func copyFileWithHash(src, dst string) (sum string, err error) {
	in, err := os.Open(src)
	if err != nil {
		return "", err
	}
	defer in.Close()
	mode := os.FileMode(0o644)
	if fi, statErr := in.Stat(); statErr == nil {
		mode = fi.Mode().Perm()
	}
	tmp, err := os.CreateTemp(filepath.Dir(dst), ".tmp-copy-*")
	if err != nil {
		return "", err
	}
	tmpName := tmp.Name()
	defer func() {
		if err != nil {
			_ = tmp.Close()
			_ = os.Remove(tmpName)
		}
	}()
	h := sha256.New()
	if _, err = io.Copy(io.MultiWriter(tmp, h), in); err != nil {
		return "", err
	}
	if err = tmp.Sync(); err != nil {
		return "", err
	}
	if err = tmp.Chmod(mode); err != nil {
		return "", err
	}
	if err = tmp.Close(); err != nil {
		return "", err
	}
	if err = os.Rename(tmpName, dst); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
