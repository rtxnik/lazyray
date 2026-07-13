package core

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
)

// ErrXrayVersionNotPinned means the requested xray tag is not in the embedded,
// lazyray-signed pin map, so no trusted hash exists to verify against.
var ErrXrayVersionNotPinned = errors.New("xray-core version not pinned in this lazyray release")

// ErrXrayAssetNotPinned means the platform asset is absent for a pinned version.
var ErrXrayAssetNotPinned = errors.New("xray-core asset not pinned for this version")

// xrayPins maps a pinned xray-core release tag to each platform archive's
// expected SHA-256. It is compiled into the lazyray binary, so it inherits the
// release signature (the download is verified against THIS value, never a
// co-downloaded checksum). Regenerate when bumping the pinned version. The
// hash values are public integrity data; the trailing markers keep the secret
// scanner from flagging them as high-entropy strings.
var xrayPins = map[string]map[string]string{
	"v26.3.27": {
		"Xray-linux-64.zip":          "23cd9af937744d97776ee35ecad4972cf4b2109d1e0fe6be9930467608f7c8ae", // gitleaks:allow
		"Xray-linux-arm64-v8a.zip":   "4d30283ae614e3057f730f67cd088a42be6fdf91f8639d82cb69e48cde80413c", // gitleaks:allow
		"Xray-macos-64.zip":          "f5b0471d3459eff1b82e48af0aeac186abcc3298210070afbbbd8437a4e8b203", // gitleaks:allow
		"Xray-macos-arm64-v8a.zip":   "2e93a67e8aa1936ecefb307e120830fcbd4c643ab9b1c46a2d0838d5f8409eaf", // gitleaks:allow
		"Xray-windows-64.zip":        "d004c39288ce9ada487c6f398c7c545f7d749e44bdfdd59dbc9f865afba4e1ad", // gitleaks:allow
		"Xray-windows-arm64-v8a.zip": "35d4ed6ec21224fb22b07c2c3f672e2350cd536f2c74d309150175a76365ea88", // gitleaks:allow
	},
}

// verifyXrayAgainstPins checks the SHA-256 of the zip at zipPath against the
// lazyray-signed expected value for (version, assetName) from the embedded pin
// map. Fail closed: an unpinned version/asset or any hash mismatch is an error
// and the archive must never be extracted or executed.
func verifyXrayAgainstPins(version, assetName, zipPath string) error {
	assets, ok := xrayPins[version]
	if !ok {
		return fmt.Errorf("%w: %s", ErrXrayVersionNotPinned, version)
	}
	want, ok := assets[assetName]
	if !ok {
		return fmt.Errorf("%w: %s@%s", ErrXrayAssetNotPinned, assetName, version)
	}
	f, err := os.Open(zipPath)
	if err != nil {
		return fmt.Errorf("%w: opening archive for pin check: %v", ErrXrayChecksumMismatch, err)
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return fmt.Errorf("%w: hashing archive: %v", ErrXrayChecksumMismatch, err)
	}
	got := hex.EncodeToString(h.Sum(nil))
	if got != want {
		return fmt.Errorf("%w: have %s, want %s (lazyray-pinned)", ErrXrayChecksumMismatch, got, want)
	}
	return nil
}
