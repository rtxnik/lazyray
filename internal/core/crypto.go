package core

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"

	"github.com/rtxnik/lazyray/internal/config"
	"golang.org/x/crypto/pbkdf2"
)

const (
	// encryptedPrefix identifies encrypted export data.
	encryptedPrefix = "LZRENC1:"
	// pbkdf2Iterations for key derivation.
	pbkdf2Iterations = 100000
	// saltSize in bytes.
	saltSize = 16
)

// encryptedBundle is the JSON structure inside the encrypted export.
type encryptedBundle struct {
	Salt       string `json:"salt"`       // Base64-encoded PBKDF2 salt
	Nonce      string `json:"nonce"`      // Base64-encoded AES-GCM nonce
	Ciphertext string `json:"ciphertext"` // Base64-encoded encrypted data
}

// ExportEncrypted exports profiles as an encrypted string.
// The password is used to derive an AES-256-GCM key via PBKDF2.
func ExportEncrypted(profiles []config.Profile, password string) (string, error) {
	if len(profiles) == 0 {
		return "", fmt.Errorf("no profiles to export")
	}
	if password == "" {
		return "", fmt.Errorf("password is required for encrypted export")
	}

	// Serialize profiles to JSON
	data, err := json.Marshal(profiles)
	if err != nil {
		return "", fmt.Errorf("marshaling profiles: %w", err)
	}

	// Generate random salt
	salt := make([]byte, saltSize)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return "", fmt.Errorf("generating salt: %w", err)
	}

	// Derive key using PBKDF2
	key := pbkdf2.Key([]byte(password), salt, pbkdf2Iterations, 32, sha256.New)

	// Encrypt with AES-256-GCM
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("creating cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("creating GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("generating nonce: %w", err)
	}

	ciphertext := gcm.Seal(nil, nonce, data, nil)

	// Build the bundle
	bundle := encryptedBundle{
		Salt:       base64.StdEncoding.EncodeToString(salt),
		Nonce:      base64.StdEncoding.EncodeToString(nonce),
		Ciphertext: base64.StdEncoding.EncodeToString(ciphertext),
	}

	bundleJSON, err := json.Marshal(bundle)
	if err != nil {
		return "", fmt.Errorf("marshaling bundle: %w", err)
	}

	return encryptedPrefix + base64.StdEncoding.EncodeToString(bundleJSON), nil
}

// ImportEncrypted decrypts and imports profiles from an encrypted export string.
func ImportEncrypted(encrypted string, password string) ([]config.Profile, error) {
	if password == "" {
		return nil, fmt.Errorf("password is required for decryption")
	}

	if len(encrypted) <= len(encryptedPrefix) {
		return nil, fmt.Errorf("invalid encrypted data: too short")
	}

	if encrypted[:len(encryptedPrefix)] != encryptedPrefix {
		return nil, fmt.Errorf("invalid encrypted data: missing prefix")
	}

	bundleB64 := encrypted[len(encryptedPrefix):]
	bundleJSON, err := base64.StdEncoding.DecodeString(bundleB64)
	if err != nil {
		return nil, fmt.Errorf("decoding bundle: %w", err)
	}

	var bundle encryptedBundle
	if err := json.Unmarshal(bundleJSON, &bundle); err != nil {
		return nil, fmt.Errorf("parsing bundle: %w", err)
	}

	salt, err := base64.StdEncoding.DecodeString(bundle.Salt)
	if err != nil {
		return nil, fmt.Errorf("decoding salt: %w", err)
	}

	nonce, err := base64.StdEncoding.DecodeString(bundle.Nonce)
	if err != nil {
		return nil, fmt.Errorf("decoding nonce: %w", err)
	}

	ciphertext, err := base64.StdEncoding.DecodeString(bundle.Ciphertext)
	if err != nil {
		return nil, fmt.Errorf("decoding ciphertext: %w", err)
	}

	// Derive key
	key := pbkdf2.Key([]byte(password), salt, pbkdf2Iterations, 32, sha256.New)

	// Decrypt
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("creating cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("creating GCM: %w", err)
	}

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decryption failed (wrong password?): %w", err)
	}

	var profiles []config.Profile
	if err := json.Unmarshal(plaintext, &profiles); err != nil {
		return nil, fmt.Errorf("parsing decrypted profiles: %w", err)
	}

	return profiles, nil
}

// IsEncryptedExport checks if a string is an encrypted lazyray export.
func IsEncryptedExport(s string) bool {
	return len(s) > len(encryptedPrefix) && s[:len(encryptedPrefix)] == encryptedPrefix
}
