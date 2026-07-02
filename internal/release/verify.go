// Package release verifies the authenticity and integrity of lazyray release
// artifacts. It is a pure verifier: it never reaches the network and never calls
// os.Exit. Trust is rooted in a minisign public key embedded at release time.
package release

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"aead.dev/minisign"
)

// releaseSigningPubKey is the live minisign public key the project signs its
// releases with. It is a var (not const) so consumer tests (e.g. self-update)
// can point the verifier at an ephemeral key via SetPublicKeyForTest.
var releaseSigningPubKey = "RWT1X2unwbak2iRSpo1E/k3BWHDjQCzAwgPJft7dtXwRS+3IFxNkR0Ag"

// Typed sentinels returned by Verify and VerifyRelease so callers can branch on
// the exact failure with errors.Is.
var (
	// ErrSignatureInvalid means the minisign signature did not verify against the
	// public key (bad/forged signature, or the signed message was altered).
	ErrSignatureInvalid = errors.New("release: signature verification failed")
	// ErrChecksumMismatch means the archive's computed SHA-256 did not match the
	// value recorded in the signed checksum manifest.
	ErrChecksumMismatch = errors.New("release: checksum mismatch")
	// ErrAssetNotFound means the checksum manifest has no entry for the archive.
	ErrAssetNotFound = errors.New("release: asset not found in checksum manifest")
)

// DefaultPublicKey returns the embedded release-signing public key.
func DefaultPublicKey() string {
	return releaseSigningPubKey
}

// SetPublicKeyForTest temporarily replaces the embedded release-signing key with
// pubText and returns a restore function that reinstates the previous value.
// It exists so consumer packages (e.g. self-update) can exercise the production
// VerifyRelease path against an ephemeral keypair. Not for production use.
func SetPublicKeyForTest(pubText string) (restore func()) {
	prev := releaseSigningPubKey
	releaseSigningPubKey = pubText
	return func() { releaseSigningPubKey = prev }
}

// Verify checks that sig is a valid minisign signature over msg under the
// public key encoded in pubKeyText. sig is the full content of a .minisig file.
// It returns nil on success, ErrSignatureInvalid on a bad signature, or a
// descriptive error if pubKeyText cannot be parsed.
func Verify(pubKeyText string, msg, sig []byte) error {
	var pub minisign.PublicKey
	if err := pub.UnmarshalText([]byte(pubKeyText)); err != nil {
		return fmt.Errorf("release: parsing public key: %w", err)
	}
	if !minisign.Verify(pub, msg, sig) {
		return ErrSignatureInvalid
	}
	return nil
}

// VerifyRelease performs the full release-artifact check against the embedded
// public key:
//  1. verify checksumsSig over checksumsTxt,
//  2. parse the SHA-256 for filepath.Base(archivePath) from checksumsTxt,
//  3. compute the SHA-256 of archivePath and compare.
//
// Any failed step fails closed with a typed sentinel.
func VerifyRelease(archivePath string, checksumsTxt, checksumsSig []byte) error {
	return verifyReleaseWithKey(DefaultPublicKey(), archivePath, checksumsTxt, checksumsSig)
}

// verifyReleaseWithKey is the testable core of VerifyRelease. Tests inject an
// ephemeral public key here instead of the embedded production key.
func verifyReleaseWithKey(pubKeyText, archivePath string, checksumsTxt, checksumsSig []byte) error {
	if err := Verify(pubKeyText, checksumsTxt, checksumsSig); err != nil {
		return err
	}

	name := filepath.Base(archivePath)
	want, err := checksumForAsset(checksumsTxt, name)
	if err != nil {
		return err
	}

	got, err := sha256OfFile(archivePath)
	if err != nil {
		return err
	}

	if !strings.EqualFold(got, want) {
		return fmt.Errorf("%w: %s (have %s, want %s)", ErrChecksumMismatch, name, got, want)
	}
	return nil
}

// checksumForAsset returns the lowercase hex SHA-256 recorded for assetName in a
// GNU sha256sum-style manifest ("<hex>  <filename>" per line, two spaces).
func checksumForAsset(checksumsTxt []byte, assetName string) (string, error) {
	scanner := bufio.NewScanner(bytes.NewReader(checksumsTxt))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		// Split on the first run of whitespace: "<hex>  <filename>".
		fields := strings.Fields(line)
		if len(fields) != 2 {
			continue
		}
		if fields[1] == assetName {
			return strings.ToLower(fields[0]), nil
		}
	}
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("release: reading checksum manifest: %w", err)
	}
	return "", fmt.Errorf("%w: %s", ErrAssetNotFound, assetName)
}

// sha256OfFile returns the lowercase hex SHA-256 of the file at path.
func sha256OfFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("release: reading archive: %w", err)
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}
