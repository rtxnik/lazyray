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
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"time"

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

// errHostKeyCaptured aborts the probe handshake once the host key has been
// recorded; no authentication is ever attempted.
var errHostKeyCaptured = errors.New("host key captured")

var hostKeyDialTimeout = 10 * time.Second

// hostKeyDial is the capture/pre-flight dial seam. It connects to addr,
// records the host key the server presents (negotiation restricted to algos
// when non-nil), and aborts before authentication.
var hostKeyDial = realHostKeyDial

// SetHostKeyDialForTest swaps the capture dial and returns a restore func.
func SetHostKeyDialForTest(fn func(addr string, algos []string) (HostKey, error)) (restore func()) {
	prev := hostKeyDial
	hostKeyDial = fn
	return func() { hostKeyDial = prev }
}

func realHostKeyDial(addr string, algos []string) (HostKey, error) {
	var captured HostKey
	cfg := &ssh.ClientConfig{
		User: "lazyray-hostkey-probe",
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			captured = HostKey{
				Type:   key.Type(),
				Base64: base64.StdEncoding.EncodeToString(key.Marshal()),
			}
			return errHostKeyCaptured
		},
		HostKeyAlgorithms: algos,
		Timeout:           hostKeyDialTimeout,
	}
	conn, err := net.DialTimeout("tcp", addr, hostKeyDialTimeout)
	if err != nil {
		return HostKey{}, err
	}
	defer conn.Close()
	// ClientConfig.Timeout bounds only the TCP dial; the SSH handshake has no
	// deadline of its own, so a stalled peer would hang the pre-flight check
	// forever. Bound the whole probe with a connection deadline instead.
	_ = conn.SetDeadline(time.Now().Add(hostKeyDialTimeout))
	c, chans, reqs, err := ssh.NewClientConn(conn, addr, cfg)
	if err == nil {
		// Not reachable in practice: the callback always aborts the handshake.
		go ssh.DiscardRequests(reqs)
		go func() {
			for ch := range chans {
				_ = ch.Reject(ssh.Prohibited, "probe only")
			}
		}()
		_ = c.Close()
	}
	if captured.Type != "" {
		return captured, nil
	}
	if err == nil {
		err = errors.New("server presented no host key")
	}
	return HostKey{}, err
}

// captureFamilies: one probe per key-algorithm family so multi-algorithm
// servers are pinned in full (robust against a server later dropping one
// algorithm).
var captureFamilies = [][]string{
	{"ssh-ed25519"},
	{"ecdsa-sha2-nistp256", "ecdsa-sha2-nistp384", "ecdsa-sha2-nistp521"},
	{"rsa-sha2-512", "rsa-sha2-256", "ssh-rsa"},
}

// CaptureHostKeys probes host:port once per key-algorithm family and returns
// the deduplicated set of presented keys. It fails only when no family
// yields a key (host unreachable / not an SSH server).
func CaptureHostKeys(host string, port int) ([]HostKey, error) {
	addr := net.JoinHostPort(host, strconv.Itoa(port))
	var keys []HostKey
	seen := make(map[string]bool)
	var lastErr error
	for _, family := range captureFamilies {
		k, err := hostKeyDial(addr, family)
		if err != nil {
			lastErr = err
			if isTransportError(err) {
				break // dead host: don't burn two more dial timeouts
			}
			continue
		}
		if !seen[k.String()] {
			seen[k.String()] = true
			keys = append(keys, k)
		}
	}
	if len(keys) == 0 {
		return nil, fmt.Errorf("capturing host key from %s: %w", addr, lastErr)
	}
	return keys, nil
}

// isTransportError reports whether err is a network-level failure (dial
// refused, DNS failure, timeout, immediate connection drop) as opposed to an
// SSH protocol/negotiation failure. Misclassifying a transport failure as
// negotiation would surface a false "host key changed" MITM warning for a
// merely dead or flaky server, so this errs on the transport side: io.EOF
// (x/crypto/ssh returns it bare when the peer closes during handshake) and
// DNS errors are transport too.
func isTransportError(err error) bool {
	if errors.Is(err, io.EOF) {
		return true
	}
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return true
	}
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		return true
	}
	var nErr net.Error
	return errors.As(err, &nErr) && nErr.Timeout()
}

// verifyPinnedHostKey dials the host restricted to the pinned algorithms and
// confirms the presented key is one of the pins.
//   - match                       → nil (proceed to spawn ssh)
//   - different key / negotiation → *ErrHostKeyChanged carrying the live key
//     set (one unrestricted capture, so the old-vs-new UX always has data —
//     a server rotated to an unpinned algorithm would otherwise be blank)
//   - unreachable                 → plain error, fail closed
func verifyPinnedHostKey(host string, port int, pinned []HostKey) error {
	addr := net.JoinHostPort(host, strconv.Itoa(port))
	presented, dialErr := hostKeyDial(addr, hostKeyAlgorithmsFor(pinned))
	if dialErr == nil {
		for _, p := range pinned {
			if p == presented {
				return nil
			}
		}
	} else if isTransportError(dialErr) {
		return fmt.Errorf("cannot reach %s to verify its identity: %w", addr, dialErr)
	}
	changed := &ErrHostKeyChanged{Host: host, Port: port, Pinned: pinned}
	if captured, err := CaptureHostKeys(host, port); err == nil {
		changed.Captured = captured
	} else if dialErr == nil {
		changed.Captured = []HostKey{presented}
	}
	return changed
}
