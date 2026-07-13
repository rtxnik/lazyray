package core

import (
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

func writeTempZip(t *testing.T, content []byte) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "xray.zip")
	if err := os.WriteFile(p, content, 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	return p
}

func TestXrayPins_ShapeIsSha256Hex(t *testing.T) {
	// No literal hashes in the test (they would trip the secret scanner too);
	// assert the shipped map is non-empty and every value is 64-char lowercase hex.
	hex64 := regexp.MustCompile(`^[a-f0-9]{64}$`)
	assets, ok := xrayPins["v26.3.27"]
	if !ok || len(assets) == 0 {
		t.Fatal("xrayPins has no entries for the pinned version v26.3.27")
	}
	for name, sum := range assets {
		if !hex64.MatchString(sum) {
			t.Fatalf("pin %s = %q is not 64-char lowercase hex", name, sum)
		}
	}
}

func TestVerifyXrayAgainstPins_WrongBytesRejected(t *testing.T) {
	zip := writeTempZip(t, []byte("not the real xray zip"))
	if err := verifyXrayAgainstPins("v26.3.27", "Xray-linux-64.zip", zip); !errors.Is(err, ErrXrayChecksumMismatch) {
		t.Fatalf("wrong-bytes zip: got %v, want ErrXrayChecksumMismatch", err)
	}
}

func TestVerifyXrayAgainstPins_VersionNotPinned(t *testing.T) {
	zip := writeTempZip(t, []byte("x"))
	if err := verifyXrayAgainstPins("v1.0.0", "Xray-linux-64.zip", zip); !errors.Is(err, ErrXrayVersionNotPinned) {
		t.Fatalf("unpinned version: got %v, want ErrXrayVersionNotPinned", err)
	}
}

func TestVerifyXrayAgainstPins_AssetNotPinned(t *testing.T) {
	zip := writeTempZip(t, []byte("x"))
	if err := verifyXrayAgainstPins("v26.3.27", "Xray-nonsense.zip", zip); !errors.Is(err, ErrXrayAssetNotPinned) {
		t.Fatalf("unpinned asset: got %v, want ErrXrayAssetNotPinned", err)
	}
}

func TestVerifyXrayAgainstPins_ReadErrorIsChecksumMismatch(t *testing.T) {
	// A pinned version+asset but an unreadable file must fail closed AS a
	// checksum-class error (errors.Is contract), not a bare OS error.
	missing := filepath.Join(t.TempDir(), "does-not-exist.zip")
	if err := verifyXrayAgainstPins("v26.3.27", "Xray-linux-64.zip", missing); !errors.Is(err, ErrXrayChecksumMismatch) {
		t.Fatalf("read error: got %v, want ErrXrayChecksumMismatch", err)
	}
}
