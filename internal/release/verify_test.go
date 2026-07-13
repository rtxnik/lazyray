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
// so tests never depend on the embedded requiredSigners var.
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

// mkSigner generates a fresh minisign keypair and a signer entry pointing at a
// sig-asset name derived from the key's ID, so parallel signers in the same
// test never collide on asset name (the marshaled text's leading bytes are a
// constant "untrusted comment: " header shared by every key, so slicing into
// that prefix would always produce the same asset name).
func mkSigner(t *testing.T) (minisign.PrivateKey, signer, string) {
	t.Helper()
	pub, priv, err := minisign.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}
	pt, err := pub.MarshalText()
	if err != nil {
		t.Fatalf("MarshalText() error = %v", err)
	}
	asset := fmt.Sprintf("checksums.txt.%x.minisig", pub.ID())
	return priv, signer{pubKey: string(pt), sigAsset: asset}, asset
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
	priv, s, asset := mkSigner(t)
	restore := SetRequiredSignersForTest([]signer{s})
	defer restore()

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
			sigs := map[string][]byte{asset: tt.sig}
			err := VerifyRelease(archivePath, tt.checksums, sigs)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("VerifyRelease() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestVerifyRelease_ChecksumMismatch(t *testing.T) {
	priv, s, asset := mkSigner(t)
	restore := SetRequiredSignersForTest([]signer{s})
	defer restore()

	archivePath := writeArchive(t, []byte("real bytes on disk"))
	archiveName := filepath.Base(archivePath)

	// Manifest lists a DIFFERENT sha256 for our archive name, but is correctly signed.
	checksums := []byte(checksumsLine([]byte("different bytes"), archiveName))
	sig := minisign.Sign(priv, checksums)

	err := VerifyRelease(archivePath, checksums, map[string][]byte{asset: sig})
	if !errors.Is(err, ErrChecksumMismatch) {
		t.Errorf("VerifyRelease() error = %v, want %v", err, ErrChecksumMismatch)
	}
}

func TestVerifyRelease_AssetNotFound(t *testing.T) {
	priv, s, asset := mkSigner(t)
	restore := SetRequiredSignersForTest([]signer{s})
	defer restore()

	archivePath := writeArchive(t, []byte("bytes"))

	// Correctly signed manifest, but it has no line for our archive's basename.
	checksums := []byte(checksumsLine([]byte("x"), "some_other_artifact.tar.gz"))
	sig := minisign.Sign(priv, checksums)

	err := VerifyRelease(archivePath, checksums, map[string][]byte{asset: sig})
	if !errors.Is(err, ErrAssetNotFound) {
		t.Errorf("VerifyRelease() error = %v, want %v", err, ErrAssetNotFound)
	}
}

func TestVerifyRelease_TwoOfTwo(t *testing.T) {
	aPriv, aSigner, aAsset := mkSigner(t)
	bPriv, bSigner, bAsset := mkSigner(t)
	restore := SetRequiredSignersForTest([]signer{aSigner, bSigner})
	defer restore()

	content := []byte("archive bytes")
	archivePath := writeArchive(t, content)
	checksums := []byte(checksumsLine(content, filepath.Base(archivePath)))
	both := map[string][]byte{aAsset: minisign.Sign(aPriv, checksums), bAsset: minisign.Sign(bPriv, checksums)}
	if err := VerifyRelease(archivePath, checksums, both); err != nil {
		t.Fatalf("both signatures present: %v, want nil", err)
	}
	only := map[string][]byte{aAsset: both[aAsset]}
	if err := VerifyRelease(archivePath, checksums, only); err == nil {
		t.Fatal("accepted a release missing a required signature")
	}
}

func TestVerifyRelease_ProductionInvariant_NoAcceptAny(t *testing.T) {
	if len(requiredSigners) == 0 {
		t.Fatal("shipped requiredSigners is empty")
	}
	otherPriv, _, _ := mkSigner(t)
	content := []byte("archive bytes")
	archivePath := writeArchive(t, content)
	checksums := []byte(checksumsLine(content, filepath.Base(archivePath)))
	sigs := map[string][]byte{requiredSigners[0].sigAsset: minisign.Sign(otherPriv, checksums)}
	if err := VerifyRelease(archivePath, checksums, sigs); err == nil {
		t.Fatal("shipped verifier accepted a signature from an unlisted key (accept-any regression)")
	}
}
