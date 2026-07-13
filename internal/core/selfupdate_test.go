package core

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"aead.dev/minisign"
	"github.com/rtxnik/lazyray/internal/release"
)

func TestSelfAssetName(t *testing.T) {
	ext := ".tar.gz"
	if runtime.GOOS == "windows" {
		ext = ".zip"
	}
	tests := []struct {
		name    string
		version string
		want    string
	}{
		{
			name:    "tag with v prefix is trimmed",
			version: "v0.9.0",
			want:    fmt.Sprintf("lazyray_0.9.0_%s_%s%s", runtime.GOOS, runtime.GOARCH, ext),
		},
		{
			name:    "tag without v prefix is unchanged",
			version: "1.2.3",
			want:    fmt.Sprintf("lazyray_1.2.3_%s_%s%s", runtime.GOOS, runtime.GOARCH, ext),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SelfAssetName(tt.version); got != tt.want {
				t.Errorf("SelfAssetName(%q) = %q, want %q", tt.version, got, tt.want)
			}
		})
	}
}

// makeTarGz builds an in-memory tar.gz containing a single entry name->content.
func makeTarGz(t *testing.T, name string, content []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)
	hdr := &tar.Header{Name: name, Mode: 0644, Size: int64(len(content))}
	if err := tw.WriteHeader(hdr); err != nil {
		t.Fatalf("tar header: %v", err)
	}
	if _, err := tw.Write(content); err != nil {
		t.Fatalf("tar write: %v", err)
	}
	if err := tw.Close(); err != nil {
		t.Fatalf("tar close: %v", err)
	}
	if err := gw.Close(); err != nil {
		t.Fatalf("gzip close: %v", err)
	}
	return buf.Bytes()
}

func TestExtractFromTarGz(t *testing.T) {
	dir := t.TempDir()
	want := []byte("#!/bin/sh\necho lzr\n")
	archive := filepath.Join(dir, "a.tar.gz")
	if err := os.WriteFile(archive, makeTarGz(t, "lzr", want), 0644); err != nil {
		t.Fatalf("write archive: %v", err)
	}

	t.Run("extracts the target entry", func(t *testing.T) {
		dest := filepath.Join(dir, "out-lzr")
		if err := extractFromTarGz(archive, "lzr", dest); err != nil {
			t.Fatalf("extractFromTarGz: %v", err)
		}
		got, err := os.ReadFile(dest)
		if err != nil {
			t.Fatalf("read dest: %v", err)
		}
		if !bytes.Equal(got, want) {
			t.Errorf("extracted content = %q, want %q", got, want)
		}
	})

	t.Run("missing entry returns error", func(t *testing.T) {
		dest := filepath.Join(dir, "out-missing")
		err := extractFromTarGz(archive, "nope", dest)
		if err == nil {
			t.Fatal("expected error for missing entry, got nil")
		}
		if _, statErr := os.Stat(dest); !os.IsNotExist(statErr) {
			t.Errorf("dest should not be created on miss, stat err = %v", statErr)
		}
	})
}

func TestFindSelfAssetURL(t *testing.T) {
	archiveName := SelfAssetName("v0.9.0")
	rel := &ReleaseInfo{
		TagName: "v0.9.0",
		Assets: []Asset{
			{Name: archiveName, BrowserDownloadURL: "https://example.com/" + archiveName},
			{Name: "checksums.txt", BrowserDownloadURL: "https://example.com/checksums.txt"},
			{Name: "checksums.txt.minisig", BrowserDownloadURL: "https://example.com/checksums.txt.minisig"},
		},
	}

	t.Run("resolves all three URLs", func(t *testing.T) {
		urls, err := FindSelfAssetURL(rel)
		if err != nil {
			t.Fatalf("FindSelfAssetURL: %v", err)
		}
		if urls.Archive != "https://example.com/"+archiveName {
			t.Errorf("Archive = %q", urls.Archive)
		}
		if urls.Checksums != "https://example.com/checksums.txt" {
			t.Errorf("Checksums = %q", urls.Checksums)
		}
		if urls.Signatures["checksums.txt.minisig"] != "https://example.com/checksums.txt.minisig" {
			t.Errorf("Signatures[checksums.txt.minisig] = %q", urls.Signatures["checksums.txt.minisig"])
		}
		if urls.AssetName != archiveName {
			t.Errorf("AssetName = %q, want %q", urls.AssetName, archiveName)
		}
	})

	t.Run("missing archive returns ErrAssetNotFound", func(t *testing.T) {
		bad := &ReleaseInfo{
			TagName: "v0.9.0",
			Assets: []Asset{
				{Name: "checksums.txt", BrowserDownloadURL: "https://example.com/checksums.txt"},
				{Name: "checksums.txt.minisig", BrowserDownloadURL: "https://example.com/checksums.txt.minisig"},
			},
		}
		_, err := FindSelfAssetURL(bad)
		if !errors.Is(err, release.ErrAssetNotFound) {
			t.Errorf("err = %v, want release.ErrAssetNotFound", err)
		}
	})

	t.Run("missing signature returns ErrAssetNotFound", func(t *testing.T) {
		bad := &ReleaseInfo{
			TagName: "v0.9.0",
			Assets: []Asset{
				{Name: archiveName, BrowserDownloadURL: "https://example.com/" + archiveName},
				{Name: "checksums.txt", BrowserDownloadURL: "https://example.com/checksums.txt"},
			},
		}
		_, err := FindSelfAssetURL(bad)
		if !errors.Is(err, release.ErrAssetNotFound) {
			t.Errorf("err = %v, want release.ErrAssetNotFound", err)
		}
	})
}

func TestFindSelfAssetURL_RequiresAllSigAssets(t *testing.T) {
	rel := &ReleaseInfo{TagName: "v0.9.0", Assets: []Asset{
		{Name: "lazyray_0.9.0_" + runtime.GOOS + "_" + runtime.GOARCH + selfArchiveExt(), BrowserDownloadURL: "u/a"},
		{Name: "checksums.txt", BrowserDownloadURL: "u/c"},
		// checksums.txt.minisig deliberately ABSENT
	}}
	if _, err := FindSelfAssetURL(rel); !errors.Is(err, release.ErrAssetNotFound) {
		t.Fatalf("missing required sig asset: got %v, want ErrAssetNotFound", err)
	}
}

// sha256Hex returns the lowercase hex SHA-256 of b.
func sha256Hex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// selfUpdateFixture serves a real archive + checksums.txt + checksums.txt.minisig
// signed by an ephemeral minisign key, and overrides release's embedded key for
// the duration of the test. mutate lets a case tamper with the served bytes.
type selfUpdateFixture struct {
	server      *httptest.Server
	archiveName string
}

func newSelfUpdateFixture(t *testing.T, newBinary []byte, mutate func(checksums, sig *[]byte, archive *[]byte)) *selfUpdateFixture {
	t.Helper()

	// Ephemeral signing key; install its public key into the verifier seam.
	pub, priv, err := minisign.GenerateKey(nil)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	restore := release.SetRequiredSignerForTest(pub.String(), "checksums.txt.minisig")
	t.Cleanup(restore)

	archiveName := SelfAssetName("v0.9.0")
	var archive []byte
	if strings.HasSuffix(archiveName, ".zip") {
		archive = makeZip(t, "lzr.exe", newBinary)
	} else {
		archive = makeTarGz(t, "lzr", newBinary)
	}

	checksums := []byte(fmt.Sprintf("%s  %s\n", sha256Hex(archive), archiveName))
	sig := minisign.Sign(priv, checksums)

	if mutate != nil {
		mutate(&checksums, &sig, &archive)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/"+archiveName, func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write(archive) })
	mux.HandleFunc("/checksums.txt", func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write(checksums) })
	mux.HandleFunc("/checksums.txt.minisig", func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write(sig) })

	// safeGet enforces https-only, so serve TLS. directClient builds its own
	// transport; point its TLS seam at a pool that trusts this server's
	// self-signed cert for the duration of the test (core tests are sequential
	// — no t.Parallel — so the global swap is safe and restored).
	srv := httptest.NewTLSServer(mux)
	t.Cleanup(srv.Close)

	pool := x509.NewCertPool()
	pool.AddCert(srv.Certificate())
	origTLS := directTLSConfig
	directTLSConfig = &tls.Config{RootCAs: pool}
	t.Cleanup(func() { directTLSConfig = origTLS })

	return &selfUpdateFixture{server: srv, archiveName: archiveName}
}

func (f *selfUpdateFixture) urls() SelfAssetURLs {
	return SelfAssetURLs{
		AssetName: f.archiveName,
		Archive:   f.server.URL + "/" + f.archiveName,
		Checksums: f.server.URL + "/checksums.txt",
		Signatures: map[string]string{
			"checksums.txt.minisig": f.server.URL + "/checksums.txt.minisig",
		},
	}
}

func TestApplySelfUpdate(t *testing.T) {
	// httptest serves 127.0.0.1; allow loopback through the SSRF guard for this test.
	orig := lookupIP
	lookupIP = func(host string) ([]net.IP, error) { return []net.IP{net.ParseIP("93.184.216.34")}, nil }
	t.Cleanup(func() { lookupIP = orig })

	// The dial pin (pinnedDialContext) refuses loopback literals, so swap the
	// directClient dial seam for a plain dialer that can reach the local server.
	origDial := dialContext
	dialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		var d net.Dialer
		return d.DialContext(ctx, network, addr)
	}
	t.Cleanup(func() { dialContext = origDial })

	newBinary := []byte("NEW-LZR-BINARY-CONTENT")

	writeExec := func(t *testing.T) string {
		t.Helper()
		dir := t.TempDir()
		p := filepath.Join(dir, "lzr-live")
		if err := os.WriteFile(p, []byte("OLD-LIVE-BINARY"), 0755); err != nil {
			t.Fatalf("seed exec: %v", err)
		}
		return p
	}

	t.Run("happy path replaces the executable with the extracted binary", func(t *testing.T) {
		execPath := writeExec(t)
		f := newSelfUpdateFixture(t, newBinary, nil)
		if err := ApplySelfUpdate(f.urls(), execPath); err != nil {
			t.Fatalf("ApplySelfUpdate: %v", err)
		}
		got, err := os.ReadFile(execPath)
		if err != nil {
			t.Fatalf("read exec: %v", err)
		}
		if !bytes.Equal(got, newBinary) {
			t.Errorf("exec content = %q, want extracted binary %q", got, newBinary)
		}
	})

	cases := []struct {
		name    string
		mutate  func(checksums, sig, archive *[]byte)
		wantErr error
	}{
		{
			name:    "tampered checksum manifest -> ErrSignatureInvalid",
			mutate:  func(c, s, a *[]byte) { *c = append(*c, '\n') },
			wantErr: release.ErrSignatureInvalid,
		},
		{
			name:    "corrupt signature -> ErrSignatureInvalid",
			mutate:  func(c, s, a *[]byte) { (*s)[len(*s)-1] ^= 0xFF },
			wantErr: release.ErrSignatureInvalid,
		},
		{
			name:    "archive bytes do not match manifest -> ErrChecksumMismatch",
			mutate:  func(c, s, a *[]byte) { *a = append([]byte("CORRUPT"), *a...) },
			wantErr: release.ErrChecksumMismatch,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			execPath := writeExec(t)
			f := newSelfUpdateFixture(t, newBinary, tc.mutate)
			err := ApplySelfUpdate(f.urls(), execPath)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("err = %v, want %v", err, tc.wantErr)
			}
			// Live binary must be untouched on any verification failure.
			got, readErr := os.ReadFile(execPath)
			if readErr != nil {
				t.Fatalf("read exec: %v", readErr)
			}
			if string(got) != "OLD-LIVE-BINARY" {
				t.Errorf("live binary was modified on failure: %q", got)
			}
		})
	}
}

func TestSwapBinary_Success(t *testing.T) {
	dir := t.TempDir()
	execPath := filepath.Join(dir, "lzr")
	if err := os.WriteFile(execPath, []byte("OLD"), 0o755); err != nil {
		t.Fatalf("seed exec: %v", err)
	}
	// Staged replacement must live in the same dir (caller's contract).
	newPath := filepath.Join(dir, ".lzr-new")
	if err := os.WriteFile(newPath, []byte("NEW"), 0o755); err != nil {
		t.Fatalf("seed new: %v", err)
	}

	if err := swapBinary(newPath, execPath); err != nil {
		t.Fatalf("swapBinary: %v", err)
	}
	got, _ := os.ReadFile(execPath)
	if string(got) != "NEW" {
		t.Errorf("exec content = %q, want NEW", got)
	}
	if _, err := os.Stat(execPath + ".bak"); !os.IsNotExist(err) {
		t.Errorf("backup left behind after success: stat err = %v", err)
	}
}

func TestSwapBinary_RollbackOnFailure(t *testing.T) {
	dir := t.TempDir()
	execPath := filepath.Join(dir, "lzr")
	if err := os.WriteFile(execPath, []byte("OLD"), 0o755); err != nil {
		t.Fatalf("seed exec: %v", err)
	}
	// A nonexistent source makes the rename fail deterministically.
	missing := filepath.Join(dir, ".lzr-new-missing")

	if err := swapBinary(missing, execPath); err == nil {
		t.Fatal("expected swapBinary to fail with a missing source")
	}
	got, err := os.ReadFile(execPath)
	if err != nil {
		t.Fatalf("read exec after failed swap: %v", err)
	}
	if string(got) != "OLD" {
		t.Errorf("original binary was lost on failure: %q, want OLD", got)
	}
	if _, err := os.Stat(execPath + ".bak"); !os.IsNotExist(err) {
		t.Errorf("backup left behind after rollback: stat err = %v", err)
	}
}

func TestExtractFromTarGz_SyncsBeforeReturn(t *testing.T) {
	src, err := os.ReadFile("selfupdate_extract.go")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(src), "out.Sync()") {
		t.Fatal("extractFromTarGz does not fsync the extracted file before returning")
	}
}

func TestVerifyBackupSHA_RejectsTampered(t *testing.T) {
	dir := t.TempDir()
	backup := filepath.Join(dir, "lzr.bak")
	_ = os.WriteFile(backup, []byte("live-binary"), 0o755)
	sum, _ := sha256OfFileHex(backup)
	_ = os.WriteFile(backup, []byte("TAMPERED"), 0o755)
	if err := verifyBackupSHA(backup, sum); err == nil {
		t.Fatal("verifyBackupSHA accepted a tampered .bak")
	}
	// A matching backup passes.
	_ = os.WriteFile(backup, []byte("live-binary"), 0o755)
	if err := verifyBackupSHA(backup, sum); err != nil {
		t.Fatalf("verifyBackupSHA rejected an untampered .bak: %v", err)
	}
}

func TestRestoreVerifiedBackup(t *testing.T) {
	dir := t.TempDir()
	exec := filepath.Join(dir, "lzr")
	backup := exec + ".bak"
	if err := os.WriteFile(exec, []byte("intact-original"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(backup, []byte("good-backup"), 0o755); err != nil {
		t.Fatal(err)
	}
	sum, _ := sha256OfFileHex(backup)
	// Tampered backup -> error, execPath left intact.
	if err := os.WriteFile(backup, []byte("TAMPERED"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := restoreVerifiedBackup(exec, backup, sum); err == nil {
		t.Fatal("tampered backup accepted")
	}
	if got, _ := os.ReadFile(exec); string(got) != "intact-original" {
		t.Fatal("execPath overwritten despite tampered backup")
	}
	// Untampered backup -> restored.
	if err := os.WriteFile(backup, []byte("good-backup"), 0o755); err != nil {
		t.Fatal(err)
	}
	sum2, _ := sha256OfFileHex(backup)
	if err := restoreVerifiedBackup(exec, backup, sum2); err != nil {
		t.Fatalf("untampered restore failed: %v", err)
	}
	if got, _ := os.ReadFile(exec); string(got) != "good-backup" {
		t.Fatal("execPath not restored from untampered backup")
	}
}

func makeZip(t *testing.T, name string, content []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.Create(name)
	if err != nil {
		t.Fatalf("zip create: %v", err)
	}
	if _, err := w.Write(content); err != nil {
		t.Fatalf("zip write: %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("zip close: %v", err)
	}
	return buf.Bytes()
}
