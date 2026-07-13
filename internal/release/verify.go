// Package release verifies the authenticity and integrity of lazyray release
// artifacts. It is a pure verifier: it never reaches the network and never calls
// os.Exit. Trust is rooted in a set of minisign public keys embedded at release
// time.
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

// signer is one entry in the required-signer trust list: a minisign public key
// plus the release-asset filename carrying that signer's detached signature
// over checksums.txt.
type signer struct {
	pubKey   string // minisign public-key text
	sigAsset string // release-asset filename carrying this signer's .minisig
}

// requiredSigners: EVERY entry must produce a valid signature over checksums.txt
// for a release to verify (defeats single-key compromise). Retire a key by
// REMOVING its entry (immediate prune — no accept-any residue). Activate 2-of-2
// by appending the second, separately-custodied signer.
var requiredSigners = []signer{
	{pubKey: "RWT1X2unwbak2iRSpo1E/k3BWHDjQCzAwgPJft7dtXwRS+3IFxNkR0Ag", sigAsset: "checksums.txt.minisig"},
}

// Typed sentinels returned by Verify and VerifyRelease so callers can branch on
// the exact failure with errors.Is.
var (
	// ErrSignatureInvalid means the minisign signature did not verify against the
	// public key (bad/forged signature, or the signed message was altered).
	ErrSignatureInvalid = errors.New("release: signature verification failed")
	// ErrChecksumMismatch means the archive's computed SHA-256 did not match the
	// value recorded in the signed checksum manifest.
	ErrChecksumMismatch = errors.New("release: checksum mismatch")
	// ErrAssetNotFound means the checksum manifest has no entry for the archive,
	// or a release is missing a required signature asset.
	ErrAssetNotFound = errors.New("release: asset not found in checksum manifest")
)

// RequiredSigAssets lists the sig-asset names the self-update path must fetch:
// one per required signer, in requiredSigners order.
func RequiredSigAssets() []string {
	out := make([]string, len(requiredSigners))
	for i, s := range requiredSigners {
		out[i] = s.sigAsset
	}
	return out
}

// SetRequiredSignersForTest swaps the trust list for tests; returns a restore
// function that reinstates the previous list. Not for production use.
func SetRequiredSignersForTest(s []signer) (restore func()) {
	prev := requiredSigners
	requiredSigners = append([]signer(nil), s...)
	return func() { requiredSigners = prev }
}

// SetRequiredSignerForTest points VerifyRelease at a single ephemeral signer
// for the duration of a test. It exists for packages outside release (e.g.
// self-update) that need to exercise the production verification path against
// a test keypair but cannot spell the unexported signer type themselves. Not
// for production use.
func SetRequiredSignerForTest(pubKey, sigAsset string) (restore func()) {
	return SetRequiredSignersForTest([]signer{{pubKey: pubKey, sigAsset: sigAsset}})
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

// VerifyRelease verifies that EVERY required signer produced a valid minisign
// signature over checksumsTxt (sigs maps sig-asset name -> bytes), then matches
// the archive SHA-256 against the signed manifest. Fail closed on any missing or
// invalid required signature.
func VerifyRelease(archivePath string, checksumsTxt []byte, sigs map[string][]byte) error {
	if len(requiredSigners) == 0 {
		return ErrSignatureInvalid
	}
	for _, s := range requiredSigners {
		sig, ok := sigs[s.sigAsset]
		if !ok {
			return fmt.Errorf("%w: missing required signature %s", ErrAssetNotFound, s.sigAsset)
		}
		if err := Verify(s.pubKey, checksumsTxt, sig); err != nil {
			return err
		}
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
