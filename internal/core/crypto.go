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
	"strings"

	"github.com/rtxnik/lazyray/internal/config"
	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/pbkdf2"
)

const (
	// legacyPrefix identifies v1 encrypted data (PBKDF2-SHA256, 100k
	// iterations, implicit parameters). Decrypt-only: new data is always
	// written as LZRENC2.
	legacyPrefix = "LZRENC1:"
	// encryptedPrefix identifies v2 encrypted data (Argon2id, parameters
	// carried in the envelope).
	encryptedPrefix = "LZRENC2:"

	// legacyPBKDF2Iterations is the fixed KDF cost of LZRENC1 blobs.
	legacyPBKDF2Iterations = 100000
	// saltSize in bytes for newly generated salts.
	saltSize = 16

	// Argon2id parameters written into new LZRENC2 envelopes (RFC 9106
	// second recommended parameter set).
	argonTime      = 3
	argonMemoryKiB = 64 * 1024
	argonThreads   = 4

	// Decrypt-side ceilings for envelope-supplied parameters. They sit
	// near the emitted defaults so a crafted envelope cannot force a large
	// pre-authentication allocation; raising the emitted parameters past
	// these ceilings requires shipping a ceiling raise one release first.
	maxArgonTime      = 8
	maxArgonMemoryKiB = 128 * 1024
	maxArgonThreads   = 8
	minSaltLen        = 8
	maxSaltLen        = 64
)

// aadV2 binds v2 ciphertexts to the format version.
var aadV2 = []byte("LZRENC2")

// encryptedBundle is the JSON structure inside a legacy LZRENC1 blob.
type encryptedBundle struct {
	Salt       string `json:"salt"`       // Base64-encoded PBKDF2 salt
	Nonce      string `json:"nonce"`      // Base64-encoded AES-GCM nonce
	Ciphertext string `json:"ciphertext"` // Base64-encoded encrypted data
}

// envelopeV2 is the JSON structure inside an LZRENC2 blob. KDF parameters
// travel with the data so future cost bumps need no new format version.
type envelopeV2 struct {
	KDF        string `json:"kdf"`
	Time       uint32 `json:"time"`
	MemoryKiB  uint32 `json:"memory_kib"`
	Threads    uint8  `json:"threads"`
	Salt       string `json:"salt"`
	Nonce      string `json:"nonce"`
	Ciphertext string `json:"ciphertext"`
}

// EncryptData encrypts arbitrary bytes under a passphrase into an LZRENC2
// text blob: Argon2id key derivation, AES-256-GCM, parameters in the envelope.
func EncryptData(plaintext []byte, password string) (string, error) {
	if password == "" {
		return "", fmt.Errorf("password is required for encryption")
	}

	salt := make([]byte, saltSize)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return "", fmt.Errorf("generating salt: %w", err)
	}

	key := argon2.IDKey([]byte(password), salt, argonTime, argonMemoryKiB, argonThreads, 32)

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

	ciphertext := gcm.Seal(nil, nonce, plaintext, aadV2)

	env := envelopeV2{
		KDF:        "argon2id",
		Time:       argonTime,
		MemoryKiB:  argonMemoryKiB,
		Threads:    argonThreads,
		Salt:       base64.StdEncoding.EncodeToString(salt),
		Nonce:      base64.StdEncoding.EncodeToString(nonce),
		Ciphertext: base64.StdEncoding.EncodeToString(ciphertext),
	}
	envJSON, err := json.Marshal(env)
	if err != nil {
		return "", fmt.Errorf("marshaling envelope: %w", err)
	}
	return encryptedPrefix + base64.StdEncoding.EncodeToString(envJSON), nil
}

// DecryptData decrypts an LZRENC2 or legacy LZRENC1 text blob.
func DecryptData(blob string, password string) ([]byte, error) {
	if password == "" {
		return nil, fmt.Errorf("password is required for decryption")
	}
	switch {
	case strings.HasPrefix(blob, encryptedPrefix):
		return decryptV2(blob[len(encryptedPrefix):], password)
	case strings.HasPrefix(blob, legacyPrefix):
		return decryptV1(blob[len(legacyPrefix):], password)
	default:
		return nil, fmt.Errorf("invalid encrypted data: missing prefix")
	}
}

func decryptV2(body string, password string) ([]byte, error) {
	envJSON, err := base64.StdEncoding.DecodeString(body)
	if err != nil {
		return nil, fmt.Errorf("invalid encrypted data: decoding envelope: %w", err)
	}
	var env envelopeV2
	if err := json.Unmarshal(envJSON, &env); err != nil {
		return nil, fmt.Errorf("invalid encrypted data: parsing envelope: %w", err)
	}

	salt, err := base64.StdEncoding.DecodeString(env.Salt)
	if err != nil {
		return nil, fmt.Errorf("invalid encrypted data: decoding salt: %w", err)
	}
	// Parameter validation MUST run before any KDF work: a crafted envelope
	// must not be able to force a large allocation before authentication.
	if env.KDF != "argon2id" {
		return nil, fmt.Errorf("invalid encrypted data: unsupported kdf %q", env.KDF)
	}
	if env.Time == 0 || env.Time > maxArgonTime ||
		env.MemoryKiB == 0 || env.MemoryKiB > maxArgonMemoryKiB ||
		env.Threads == 0 || env.Threads > maxArgonThreads {
		return nil, fmt.Errorf("invalid encrypted data: kdf parameters out of range")
	}
	if len(salt) < minSaltLen || len(salt) > maxSaltLen {
		return nil, fmt.Errorf("invalid encrypted data: salt length out of range")
	}

	nonce, err := base64.StdEncoding.DecodeString(env.Nonce)
	if err != nil {
		return nil, fmt.Errorf("invalid encrypted data: decoding nonce: %w", err)
	}
	ciphertext, err := base64.StdEncoding.DecodeString(env.Ciphertext)
	if err != nil {
		return nil, fmt.Errorf("invalid encrypted data: decoding ciphertext: %w", err)
	}

	key := argon2.IDKey([]byte(password), salt, env.Time, env.MemoryKiB, env.Threads, 32)

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("creating cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("creating GCM: %w", err)
	}
	if len(nonce) != gcm.NonceSize() {
		return nil, fmt.Errorf("invalid encrypted data: bad nonce length")
	}
	plaintext, err := gcm.Open(nil, nonce, ciphertext, aadV2)
	if err != nil {
		return nil, fmt.Errorf("decryption failed (wrong password?): %w", err)
	}
	return plaintext, nil
}

func decryptV1(body string, password string) ([]byte, error) {
	bundleJSON, err := base64.StdEncoding.DecodeString(body)
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

	key := pbkdf2.Key([]byte(password), salt, legacyPBKDF2Iterations, 32, sha256.New)

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("creating cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("creating GCM: %w", err)
	}
	if len(nonce) != gcm.NonceSize() {
		return nil, fmt.Errorf("invalid encrypted data: bad nonce length")
	}
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decryption failed (wrong password?): %w", err)
	}
	return plaintext, nil
}

// ExportEncrypted exports profiles as an encrypted LZRENC2 string.
func ExportEncrypted(profiles []config.Profile, password string) (string, error) {
	if len(profiles) == 0 {
		return "", fmt.Errorf("no profiles to export")
	}
	if password == "" {
		return "", fmt.Errorf("password is required for encrypted export")
	}
	data, err := json.Marshal(profiles)
	if err != nil {
		return "", fmt.Errorf("marshaling profiles: %w", err)
	}
	return EncryptData(data, password)
}

// ImportEncrypted decrypts and imports profiles from an encrypted export
// string in either the current (LZRENC2) or legacy (LZRENC1) format.
func ImportEncrypted(encrypted string, password string) ([]config.Profile, error) {
	plaintext, err := DecryptData(encrypted, password)
	if err != nil {
		return nil, err
	}
	var profiles []config.Profile
	if err := json.Unmarshal(plaintext, &profiles); err != nil {
		return nil, fmt.Errorf("parsing decrypted profiles: %w", err)
	}
	for i := range profiles {
		SanitizeProfileDisplay(&profiles[i])
	}
	return profiles, nil
}

// IsEncryptedExport checks if a string is an encrypted lazyray export in any
// supported format version.
func IsEncryptedExport(s string) bool {
	return strings.HasPrefix(s, encryptedPrefix) || strings.HasPrefix(s, legacyPrefix)
}
