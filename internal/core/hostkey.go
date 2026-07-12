// Package core: SSH tunnel host-key trust (strict TOFU with pinning).
//
// Trust model: the profile stores full host public keys
// (SSHConfig.HostKeys, "<type> <base64>" per algorithm). Before every
// connect a per-profile known_hosts file is derived from those pins and ssh
// runs with StrictHostKeyChecking=yes against it. A pre-flight dial produces
// typed errors so the UIs can render first-connect trust prompts and
// changed-key refusals without parsing ssh stderr.
package core

import (
	"encoding/base64"
	"fmt"
	"net"
	"strconv"
	"strings"

	"golang.org/x/crypto/ssh"
)

// HostKey is one captured or pinned SSH host public key.
type HostKey struct {
	Type   string // e.g. "ssh-ed25519"
	Base64 string // standard base64 of the wire-format key
}

// String renders the profile-storage form: "<type> <base64>".
func (k HostKey) String() string { return k.Type + " " + k.Base64 }

// Fingerprint returns the OpenSSH-style SHA256 fingerprint, the same string
// `ssh-keygen -lf` prints, for out-of-band comparison.
func (k HostKey) Fingerprint() (string, error) {
	raw, err := base64.StdEncoding.DecodeString(k.Base64)
	if err != nil {
		return "", fmt.Errorf("decoding host key: %w", err)
	}
	pub, err := ssh.ParsePublicKey(raw)
	if err != nil {
		return "", fmt.Errorf("parsing host key: %w", err)
	}
	return ssh.FingerprintSHA256(pub), nil
}

// ParseHostKeys parses SSHConfig.HostKeys entries and validates that each is
// a well-formed public key (malformed pins must fail closed, not silently
// never match).
func ParseHostKeys(lines []string) ([]HostKey, error) {
	keys := make([]HostKey, 0, len(lines))
	for _, l := range lines {
		fields := strings.Fields(l)
		if len(fields) != 2 {
			return nil, fmt.Errorf("malformed pinned host key %q: want \"<type> <base64>\"", l)
		}
		k := HostKey{Type: fields[0], Base64: fields[1]}
		if _, err := k.Fingerprint(); err != nil {
			return nil, fmt.Errorf("invalid pinned host key %q: %w", l, err)
		}
		keys = append(keys, k)
	}
	return keys, nil
}

// KnownHostsToken returns the known_hosts host token ssh will look up for
// the destination exactly as it appears on the command line: the plain
// lowercased host for port 22 (unbracketed, including IPv6 literals) and
// "[host]:port" for any other port.
func KnownHostsToken(host string, port int) string {
	h := strings.ToLower(host)
	if port == 22 {
		return h
	}
	return fmt.Sprintf("[%s]:%d", h, port)
}

// DeriveKnownHosts renders the per-profile known_hosts content for the
// pinned keys. Regenerated before every connect, so it can never go stale
// relative to the profile.
func DeriveKnownHosts(host string, port int, keys []HostKey) []byte {
	token := KnownHostsToken(host, port)
	var b strings.Builder
	for _, k := range keys {
		b.WriteString(token)
		b.WriteByte(' ')
		b.WriteString(k.String())
		b.WriteByte('\n')
	}
	return []byte(b.String())
}

// ValidateSSHTarget rejects SSH.User/SSH.Host values with a leading '-':
// such values parse as ssh options and, combined with the argv, are the
// CVE-2017-1000117 option-injection class. The "--" end-of-options guard in
// the argv is the second, independent layer.
func ValidateSSHTarget(user, host string) error {
	if strings.HasPrefix(user, "-") {
		return fmt.Errorf("SSH user %q must not start with '-'", user)
	}
	if strings.HasPrefix(host, "-") {
		return fmt.Errorf("SSH host %q must not start with '-'", host)
	}
	return nil
}

// ErrHostKeyUnknown reports that the profile has no pinned host keys. The
// captured live keys are attached for the trust-prompt UX. ssh is never
// spawned on this path.
type ErrHostKeyUnknown struct {
	Host     string
	Port     int
	Captured []HostKey
}

func (e *ErrHostKeyUnknown) Error() string {
	return fmt.Sprintf("host key for %s is not trusted yet",
		net.JoinHostPort(e.Host, strconv.Itoa(e.Port)))
}

// ErrHostKeyChanged reports that the live host key does not match the pin.
// Pinned and Captured carry both sides for the old-vs-new remediation UX.
// ssh is never spawned on this path.
type ErrHostKeyChanged struct {
	Host     string
	Port     int
	Pinned   []HostKey
	Captured []HostKey
}

func (e *ErrHostKeyChanged) Error() string {
	return fmt.Sprintf("host key for %s changed (possible MITM)",
		net.JoinHostPort(e.Host, strconv.Itoa(e.Port)))
}

// hostKeyAlgorithmsFor expands pinned key types into HostKeyAlgorithms
// negotiation names, aligning the spawned ssh with the pre-flight dial so the
// two layers cannot diverge. An ssh-rsa key accepts rsa-sha2 signatures, so
// it expands to all three RSA signature algorithms.
func hostKeyAlgorithmsFor(keys []HostKey) []string {
	seen := make(map[string]bool)
	var algos []string
	add := func(a string) {
		if !seen[a] {
			seen[a] = true
			algos = append(algos, a)
		}
	}
	for _, k := range keys {
		if k.Type == "ssh-rsa" {
			add("rsa-sha2-512")
			add("rsa-sha2-256")
			add("ssh-rsa")
			continue
		}
		add(k.Type)
	}
	return algos
}
