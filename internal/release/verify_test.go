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
	if got := DefaultPublicKey(); got != releaseSigningPubKeys[0] {
		t.Errorf("DefaultPublicKey() = %q, want %q", got, releaseSigningPubKeys[0])
	}
}

// signChecksums builds a goreleaser-style checksums.txt covering archiveName's
// content plus an unrelated entry, signs it with priv, and returns both.
func signChecksums(t *testing.T, priv minisign.PrivateKey, archiveContent []byte, archiveName string) (checksums, sig []byte) {
	t.Helper()
	checksums = []byte(
		checksumsLine([]byte("other"), "lazyray_0.9.0_darwin_arm64.tar.gz") +
			checksumsLine(archiveContent, archiveName),
	)
	sig = minisign.Sign(priv, checksums)
	return checksums, sig
}

// withTamperedKeyID flips every bit of a .minisig's unauthenticated KeyID hint,
// leaving the Ed25519 signature bytes intact.
func withTamperedKeyID(t *testing.T, sig []byte) []byte {
	t.Helper()
	var s minisign.Signature
	if err := s.UnmarshalText(sig); err != nil {
		t.Fatalf("UnmarshalText() error = %v", err)
	}
	s.KeyID ^= 0xFFFFFFFFFFFFFFFF
	out, err := s.MarshalText()
	if err != nil {
		t.Fatalf("MarshalText() error = %v", err)
	}
	return out
}

func TestVerifyRelease_TrustList_ListedKeyAccepted(t *testing.T) {
	_, _, keyAText := newEphemeralKey(t)
	_, keyBPriv, keyBText := newEphemeralKey(t)

	content := []byte("pretend tar.gz bytes")
	archivePath := writeArchive(t, content)
	checksums, sig := signChecksums(t, keyBPriv, content, filepath.Base(archivePath))

	// Signed by B; B is the SECOND entry in the trust-list -> accepted.
	if err := verifyReleaseWithKeys([]string{keyAText, keyBText}, archivePath, checksums, sig); err != nil {
		t.Errorf("verifyReleaseWithKeys() listed second key: error = %v, want nil", err)
	}
}

func TestVerifyRelease_TrustList_UnlistedKeyRejected(t *testing.T) {
	_, _, keyAText := newEphemeralKey(t)
	_, keyBPriv, _ := newEphemeralKey(t)

	content := []byte("pretend tar.gz bytes")
	archivePath := writeArchive(t, content)
	checksums, sig := signChecksums(t, keyBPriv, content, filepath.Base(archivePath))

	// Signed by B; the list has only A -> rejected, fail closed.
	if err := verifyReleaseWithKeys([]string{keyAText}, archivePath, checksums, sig); !errors.Is(err, ErrSignatureInvalid) {
		t.Errorf("verifyReleaseWithKeys() unlisted key: error = %v, want %v", err, ErrSignatureInvalid)
	}
}

func TestVerifyRelease_TrustList_WrongKeyIDHintStillVerifies(t *testing.T) {
	_, keyAPriv, keyAText := newEphemeralKey(t)
	_, _, keyBText := newEphemeralKey(t)

	content := []byte("pretend tar.gz bytes")
	archivePath := writeArchive(t, content)
	checksums, sig := signChecksums(t, keyAPriv, content, filepath.Base(archivePath))

	// Corrupt the sig's KeyID hint so it matches NO listed key; A actually signed
	// the Ed25519 content, so the try-all fallback must still accept it.
	tampered := withTamperedKeyID(t, sig)
	if err := verifyReleaseWithKeys([]string{keyAText, keyBText}, archivePath, checksums, tampered); err != nil {
		t.Errorf("verifyReleaseWithKeys() wrong KeyID hint: error = %v, want nil", err)
	}
}

func TestVerifyRelease_TrustList_TamperedManifestRejected(t *testing.T) {
	_, keyAPriv, keyAText := newEphemeralKey(t)

	content := []byte("pretend tar.gz bytes")
	archivePath := writeArchive(t, content)
	archiveName := filepath.Base(archivePath)
	checksums, sig := signChecksums(t, keyAPriv, content, archiveName)

	// Rename the archive entry after signing: the manifest no longer matches its sig.
	tampered := bytes.Replace(checksums, []byte(archiveName), []byte("renamed.tar.gz"), 1)
	if err := verifyReleaseWithKeys([]string{keyAText}, archivePath, tampered, sig); !errors.Is(err, ErrSignatureInvalid) {
		t.Errorf("verifyReleaseWithKeys() tampered manifest: error = %v, want %v", err, ErrSignatureInvalid)
	}
}

func TestSetPublicKeysForTest_RoundTrip(t *testing.T) {
	orig := DefaultPublicKeys()
	restore := SetPublicKeysForTest([]string{"k1", "k2"})
	if got := DefaultPublicKeys(); len(got) != 2 || got[0] != "k1" || got[1] != "k2" {
		t.Fatalf("DefaultPublicKeys() = %v, want [k1 k2]", got)
	}
	restore()
	if got := DefaultPublicKeys(); len(got) != len(orig) || got[0] != orig[0] {
		t.Fatalf("after restore DefaultPublicKeys() = %v, want %v", got, orig)
	}
}
