package release

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"aead.dev/minisign"
)

// newEphemeralKey returns a fresh minisign keypair and the public key as text,
// so tests never depend on the embedded releaseSigningPubKey var.
func newEphemeralKey(t *testing.T) (minisign.PublicKey, minisign.PrivateKey, string) {
	t.Helper()
	pub, priv, err := minisign.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}
	pubText, err := pub.MarshalText()
	if err != nil {
		t.Fatalf("MarshalText() error = %v", err)
	}
	return pub, priv, string(pubText)
}

// writeArchive writes content to a temp file and returns its path.
func writeArchive(t *testing.T, content []byte) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "lazyray_0.9.0_linux_amd64.tar.gz")
	if err := os.WriteFile(p, content, 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	return p
}

// checksumsLine formats a goreleaser-style sha256sum line: "<hex>  <name>".
func checksumsLine(content []byte, name string) string {
	sum := sha256.Sum256(content)
	return fmt.Sprintf("%s  %s\n", hex.EncodeToString(sum[:]), name)
}

func TestVerify(t *testing.T) {
	_, priv, pubText := newEphemeralKey(t)
	msg := []byte("a35ce...  lazyray_0.9.0_linux_amd64.tar.gz\n")
	sig := minisign.Sign(priv, msg)

	tests := []struct {
		name    string
		pubText string
		msg     []byte
		sig     []byte
		wantErr error
	}{
		{name: "valid", pubText: pubText, msg: msg, sig: sig, wantErr: nil},
		{name: "tampered message", pubText: pubText, msg: append([]byte("x"), msg...), sig: sig, wantErr: ErrSignatureInvalid},
		{name: "garbage signature", pubText: pubText, msg: msg, sig: []byte("not a minisig"), wantErr: ErrSignatureInvalid},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Verify(tt.pubText, tt.msg, tt.sig)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("Verify() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestVerify_BadPublicKey(t *testing.T) {
	if err := Verify("not-a-public-key", []byte("m"), []byte("s")); err == nil {
		t.Fatal("Verify() with malformed public key should return an error")
	}
}

func TestVerifyRelease(t *testing.T) {
	_, priv, pubText := newEphemeralKey(t)

	archiveContent := []byte("pretend tar.gz bytes")
	archivePath := writeArchive(t, archiveContent)
	archiveName := filepath.Base(archivePath)

	// A well-formed goreleaser checksums.txt with our archive plus an unrelated entry.
	checksums := []byte(
		checksumsLine([]byte("other"), "lazyray_0.9.0_darwin_arm64.tar.gz") +
			checksumsLine(archiveContent, archiveName),
	)
	sig := minisign.Sign(priv, checksums)

	tests := []struct {
		name      string
		checksums []byte
		sig       []byte
		wantErr   error
	}{
		{name: "happy path", checksums: checksums, sig: sig, wantErr: nil},
		{
			name:      "tampered checksum entry",
			checksums: bytes.Replace(checksums, []byte(archiveName), []byte("renamed.tar.gz"), 1),
			sig:       sig,
			wantErr:   ErrSignatureInvalid, // manifest no longer matches its signature
		},
		{name: "bad signature", checksums: checksums, sig: []byte("garbage"), wantErr: ErrSignatureInvalid},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := verifyReleaseWithKey(pubText, archivePath, tt.checksums, tt.sig)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("verifyReleaseWithKey() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestVerifyRelease_ChecksumMismatch(t *testing.T) {
	_, priv, pubText := newEphemeralKey(t)

	archivePath := writeArchive(t, []byte("real bytes on disk"))
	archiveName := filepath.Base(archivePath)

	// Manifest lists a DIFFERENT sha256 for our archive name, but is correctly signed.
	checksums := []byte(checksumsLine([]byte("different bytes"), archiveName))
	sig := minisign.Sign(priv, checksums)

	err := verifyReleaseWithKey(pubText, archivePath, checksums, sig)
	if !errors.Is(err, ErrChecksumMismatch) {
		t.Errorf("verifyReleaseWithKey() error = %v, want %v", err, ErrChecksumMismatch)
	}
}

func TestVerifyRelease_AssetNotFound(t *testing.T) {
	_, priv, pubText := newEphemeralKey(t)

	archivePath := writeArchive(t, []byte("bytes"))

	// Correctly signed manifest, but it has no line for our archive's basename.
	checksums := []byte(checksumsLine([]byte("x"), "some_other_artifact.tar.gz"))
	sig := minisign.Sign(priv, checksums)

	err := verifyReleaseWithKey(pubText, archivePath, checksums, sig)
	if !errors.Is(err, ErrAssetNotFound) {
		t.Errorf("verifyReleaseWithKey() error = %v, want %v", err, ErrAssetNotFound)
	}
}

func TestDefaultPublicKey(t *testing.T) {
	if got := DefaultPublicKey(); got != releaseSigningPubKey {
		t.Errorf("DefaultPublicKey() = %q, want %q", got, releaseSigningPubKey)
	}
}
