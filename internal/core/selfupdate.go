package core

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/rtxnik/lazyray/internal/release"
)

const selfUpdateAPIURL = "https://api.github.com/repos/rtxnik/lazyray/releases/latest"

// selfArchiveExt returns the goreleaser archive extension for the current OS.
func selfArchiveExt() string {
	if runtime.GOOS == "windows" {
		return ".zip"
	}
	return ".tar.gz"
}

// SelfAssetName returns the goreleaser archive name for the current platform.
// version is a release tag (e.g. "v0.9.0" or "0.9.0"); the leading "v" is
// trimmed to match goreleaser's name_template lazyray_{{.Version}}_{{.Os}}_{{.Arch}}.
func SelfAssetName(version string) string {
	ver := strings.TrimPrefix(version, "v")
	return fmt.Sprintf("lazyray_%s_%s_%s%s", ver, runtime.GOOS, runtime.GOARCH, selfArchiveExt())
}

// CheckSelfUpdate checks for new lazyray releases.
func CheckSelfUpdate() (*ReleaseInfo, error) {
	client := directClient(15 * time.Second)
	resp, err := safeGet(context.Background(), client, selfUpdateAPIURL, 1<<20)
	if err != nil {
		return nil, fmt.Errorf("checking for lazyray updates: %w", err)
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

// SelfAssetURLs holds the three release-asset URLs needed for a verified
// self-update: the platform archive and the signed checksum manifest pair.
type SelfAssetURLs struct {
	AssetName string // archive basename, e.g. lazyray_0.9.0_linux_amd64.tar.gz
	Archive   string
	Checksums string
	Signature string
}

// FindSelfAssetURL resolves, from a single /releases/latest payload, the URLs of
// the current platform's archive plus checksums.txt and checksums.txt.minisig.
// The archive name is derived from rel.TagName via SelfAssetName. Any missing
// asset yields release.ErrAssetNotFound (fail closed: no partial update).
func FindSelfAssetURL(rel *ReleaseInfo) (SelfAssetURLs, error) {
	want := SelfAssetName(rel.TagName)
	urls := SelfAssetURLs{AssetName: want}
	for _, asset := range rel.Assets {
		switch asset.Name {
		case want:
			urls.Archive = asset.BrowserDownloadURL
		case "checksums.txt":
			urls.Checksums = asset.BrowserDownloadURL
		case "checksums.txt.minisig":
			urls.Signature = asset.BrowserDownloadURL
		}
	}
	if urls.Archive == "" {
		return SelfAssetURLs{}, fmt.Errorf("archive %s not found in release %s: %w", want, rel.TagName, release.ErrAssetNotFound)
	}
	if urls.Checksums == "" {
		return SelfAssetURLs{}, fmt.Errorf("checksums.txt not found in release %s: %w", rel.TagName, release.ErrAssetNotFound)
	}
	if urls.Signature == "" {
		return SelfAssetURLs{}, fmt.Errorf("checksums.txt.minisig not found in release %s: %w", rel.TagName, release.ErrAssetNotFound)
	}
	return urls, nil
}

// downloadToTemp safeGets url into a fresh temp file under dir and returns its
// path. The caller is responsible for removing the returned file.
func downloadToTemp(dir, pattern, url string) (string, error) {
	tmp, err := os.CreateTemp(dir, pattern)
	if err != nil {
		return "", fmt.Errorf("creating temp file: %w", err)
	}
	tmpPath := tmp.Name()

	client := directClient(120 * time.Second)
	resp, err := safeGet(context.Background(), client, url, 256<<20) // 256 MB cap for archive assets
	if err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return "", fmt.Errorf("downloading %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		tmp.Close()
		os.Remove(tmpPath)
		return "", fmt.Errorf("download of %s returned status %d", url, resp.StatusCode)
	}

	if _, err := io.Copy(tmp, resp.Body); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return "", fmt.Errorf("writing %s: %w", url, err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return "", fmt.Errorf("closing %s: %w", tmpPath, err)
	}
	return tmpPath, nil
}

// ApplySelfUpdate downloads the platform archive plus the signed checksum
// manifest, verifies the release with the embedded minisign key BEFORE touching
// anything, then extracts the lzr binary from the archive and atomically
// replaces execPath. It fails closed: on any verification or extraction error
// the live executable is left untouched and all temp files are removed.
func ApplySelfUpdate(urls SelfAssetURLs, execPath string) error {
	workDir, err := os.MkdirTemp("", "lazyray-update-")
	if err != nil {
		return fmt.Errorf("creating work dir: %w", err)
	}
	defer os.RemoveAll(workDir)

	tmpArchive, err := downloadToTemp(workDir, "archive-*", urls.Archive)
	if err != nil {
		return err
	}
	// VerifyRelease keys the checksum-manifest lookup on filepath.Base(archivePath),
	// so the downloaded archive must sit under its real asset name on disk.
	archivePath := filepath.Join(workDir, urls.AssetName)
	if err := os.Rename(tmpArchive, archivePath); err != nil {
		return fmt.Errorf("staging archive: %w", err)
	}
	checksumsPath, err := downloadToTemp(workDir, "checksums-*", urls.Checksums)
	if err != nil {
		return err
	}
	sigPath, err := downloadToTemp(workDir, "checksums-sig-*", urls.Signature)
	if err != nil {
		return err
	}

	checksumsTxt, err := os.ReadFile(checksumsPath)
	if err != nil {
		return fmt.Errorf("reading checksums: %w", err)
	}
	checksumsSig, err := os.ReadFile(sigPath)
	if err != nil {
		return fmt.Errorf("reading signature: %w", err)
	}

	// Verify signature + SHA-256 before extracting or swapping. Fail closed.
	if err := release.VerifyRelease(archivePath, checksumsTxt, checksumsSig); err != nil {
		return err
	}

	// Extract the lzr binary from the verified archive (never write the archive
	// raw). Stage it in the SAME directory as execPath so the final swap is an
	// atomic same-filesystem rename — no cross-device truncate-in-place.
	binName := "lzr"
	if runtime.GOOS == "windows" {
		binName = "lzr.exe"
	}
	stage, err := os.CreateTemp(filepath.Dir(execPath), ".lzr-new-*")
	if err != nil {
		return fmt.Errorf("staging update: %w", err)
	}
	extracted := stage.Name()
	stage.Close()
	defer os.Remove(extracted) // no-op once renamed into place

	if strings.HasSuffix(urls.AssetName, ".zip") {
		err = extractFromZip(archivePath, binName, extracted)
	} else {
		err = extractFromTarGz(archivePath, binName, extracted)
	}
	if err != nil {
		return fmt.Errorf("extracting %s: %w", binName, err)
	}

	if err := os.Chmod(extracted, 0o755); err != nil {
		return fmt.Errorf("setting permissions: %w", err)
	}

	// Fsync the extracted file before the rename that consumes it, so its
	// content is durable ahead of the swap.
	if f, err := os.Open(extracted); err == nil {
		_ = f.Sync()
		_ = f.Close()
	}

	// Atomic same-fs swap with backup/rollback.
	if err := swapBinary(extracted, execPath); err != nil {
		return err
	}

	// On macOS, strip the quarantine/extended attributes so Gatekeeper does not
	// block the freshly written binary. Best-effort: failure is non-fatal.
	if runtime.GOOS == "darwin" {
		_ = exec.Command("xattr", "-cr", execPath).Run()
	}

	return nil
}

// verifyBackupSHA re-hashes the file at path and compares it to want, failing
// closed if the backup was tampered with (or otherwise changed) in the window
// between it being written and being read back for a restore.
func verifyBackupSHA(path, want string) error {
	got, err := sha256OfFileHex(path)
	if err != nil {
		return err
	}
	if got != want {
		return fmt.Errorf("backup %s failed re-verification", filepath.Base(path))
	}
	return nil
}

// restoreVerifiedBackup restores backup -> execPath, but only after re-verifying
// the backup against wantSHA. If wantSHA is empty (no baseline was captured) the
// verification is skipped. On a re-verify mismatch it returns an error WITHOUT
// touching execPath, so a tampered backup can never overwrite the intact original.
func restoreVerifiedBackup(execPath, backup, wantSHA string) error {
	if wantSHA != "" {
		if err := verifyBackupSHA(backup, wantSHA); err != nil {
			return err
		}
	}
	if err := copyFile(backup, execPath); err != nil {
		return fmt.Errorf("restoring backup: %w", err)
	}
	_ = os.Chmod(execPath, 0o755)
	_ = os.Remove(backup)
	return nil
}

// swapBinary atomically replaces execPath with the already-prepared newPath,
// keeping a backup so a failed swap can be rolled back. newPath must live on the
// same filesystem as execPath (the caller stages it under filepath.Dir(execPath))
// so the rename is atomic. The original is copied — not moved — to a sibling
// .bak, so execPath stays valid right up to the instant the rename swaps in the
// new binary. On success the backup is removed; on failure the backup is
// re-verified against its recorded SHA-256 before being restored, and the
// error returned.
func swapBinary(newPath, execPath string) (err error) {
	backup := execPath + ".bak"
	_ = os.Remove(backup) // clear any stale backup from a prior interrupted run
	var backupSHA string
	if _, statErr := os.Stat(execPath); statErr == nil {
		if err = copyFile(execPath, backup); err != nil {
			return fmt.Errorf("backing up current binary: %w", err)
		}
		backupSHA, _ = sha256OfFileHex(backup)
	}
	if err = os.Rename(newPath, execPath); err != nil {
		// Roll back from the backup we just made, then drop it — but only
		// after confirming the backup itself was not tampered with in the
		// window since it was written.
		if _, statErr := os.Stat(backup); statErr == nil {
			if rErr := restoreVerifiedBackup(execPath, backup, backupSHA); rErr != nil {
				return fmt.Errorf("swap failed and %v; original left intact: %w", rErr, err)
			}
		}
		return fmt.Errorf("swapping binary: %w", err)
	}
	// Fsync the containing directory so the rename's new directory entry is
	// durable — fsyncing the dir before the rename would not persist the new
	// name, so this must happen right after the successful rename.
	if d, err := os.Open(filepath.Dir(execPath)); err == nil {
		_ = d.Sync()
		_ = d.Close()
	}
	_ = os.Remove(backup) // success: drop the backup
	return nil
}
