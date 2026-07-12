//go:build !windows

package core

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
	"net"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"
)

// startTestSSHServer runs a minimal in-process SSH server that presents one
// ed25519 host key. No auth ever succeeds — the probe aborts after key
// exchange, which is exactly what capture needs.
func startTestSSHServer(t *testing.T) (host string, port int, key HostKey) {
	t.Helper()
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	signer, err := ssh.NewSignerFromKey(priv)
	if err != nil {
		t.Fatal(err)
	}
	// NoClientAuth lets the server complete key exchange without an auth
	// callback configured; x/crypto/ssh otherwise rejects the handshake
	// before kex starts. The probe still never authenticates: its
	// HostKeyCallback aborts the handshake right after kex.
	cfg := &ssh.ServerConfig{NoClientAuth: true}
	cfg.AddHostKey(signer)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln.Close() })
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				sconn, chans, reqs, err := ssh.NewServerConn(c, cfg)
				if err != nil {
					return
				}
				go ssh.DiscardRequests(reqs)
				for ch := range chans {
					_ = ch.Reject(ssh.Prohibited, "no channels")
				}
				_ = sconn.Close()
			}(conn)
		}
	}()

	addr := ln.Addr().(*net.TCPAddr)
	pub := signer.PublicKey()
	return "127.0.0.1", addr.Port, HostKey{
		Type:   pub.Type(),
		Base64: base64.StdEncoding.EncodeToString(pub.Marshal()),
	}
}

func TestCaptureHostKeysFromLiveServer(t *testing.T) {
	host, port, want := startTestSSHServer(t)
	keys, err := CaptureHostKeys(host, port)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, k := range keys {
		if k == want {
			found = true
		}
	}
	if !found {
		t.Fatalf("captured %v, want to include %v", keys, want)
	}
}

func TestCaptureHostKeysUnreachable(t *testing.T) {
	// Reserve a port and close it: nothing listens there.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	_ = ln.Close()
	if _, err := CaptureHostKeys("127.0.0.1", port); err == nil {
		t.Fatal("capture from a closed port must fail")
	}
}

func TestVerifyPinnedHostKeyMatch(t *testing.T) {
	host, port, key := startTestSSHServer(t)
	if err := verifyPinnedHostKey(host, port, []HostKey{key}); err != nil {
		t.Fatalf("matching pin must verify: %v", err)
	}
}

func TestVerifyPinnedHostKeyChanged(t *testing.T) {
	host, port, _ := startTestSSHServer(t)
	other, _ := testHostKey(t) // different ed25519 key
	err := verifyPinnedHostKey(host, port, []HostKey{other})
	var changed *ErrHostKeyChanged
	if !errors.As(err, &changed) {
		t.Fatalf("want ErrHostKeyChanged, got %v", err)
	}
	if len(changed.Captured) == 0 {
		t.Fatal("ErrHostKeyChanged must carry the live key set for the old-vs-new UX")
	}
	if len(changed.Pinned) != 1 || changed.Pinned[0] != other {
		t.Fatalf("ErrHostKeyChanged.Pinned = %v", changed.Pinned)
	}
}

func TestVerifyPinnedHostKeyUnreachableFailsClosed(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	_ = ln.Close()
	pin, _ := testHostKey(t)
	err = verifyPinnedHostKey("127.0.0.1", port, []HostKey{pin})
	if err == nil {
		t.Fatal("unreachable host must fail closed")
	}
	var changed *ErrHostKeyChanged
	if errors.As(err, &changed) {
		t.Fatal("unreachable is a reachability failure, not a key change")
	}
}

func TestCaptureHostKeysStalledPeerTimesOut(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln.Close() })
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			// Accept and never write: a stalled peer.
			defer conn.Close()
		}
	}()
	prev := hostKeyDialTimeout
	hostKeyDialTimeout = 300 * time.Millisecond
	t.Cleanup(func() { hostKeyDialTimeout = prev })

	addr := ln.Addr().(*net.TCPAddr)
	start := time.Now()
	_, err = CaptureHostKeys("127.0.0.1", addr.Port)
	if err == nil {
		t.Fatal("stalled peer must fail the capture")
	}
	if elapsed := time.Since(start); elapsed > 3*time.Second {
		t.Fatalf("capture did not respect the probe deadline: %v", elapsed)
	}
}

func TestIsTransportErrorClassifiesEOFAndDNS(t *testing.T) {
	if !isTransportError(io.EOF) {
		t.Fatal("io.EOF must classify as a transport error")
	}
	if !isTransportError(&net.DNSError{}) {
		t.Fatal("*net.DNSError must classify as a transport error")
	}
	if isTransportError(errors.New("negotiation failed")) {
		t.Fatal("a non-transport sentinel error must not classify as a transport error")
	}
}

// TestVerifyPinnedHostKeyChangedFallbackCaptureFails covers the branch where
// the restricted dial presents a key that doesn't match the pin, and the
// follow-up unrestricted CaptureHostKeys (used to populate the old-vs-new UX)
// itself fails. ErrHostKeyChanged must still surface, falling back to the one
// key the restricted dial did present rather than leaving Captured empty.
func TestVerifyPinnedHostKeyChangedFallbackCaptureFails(t *testing.T) {
	pinned, _ := testHostKey(t)
	presented, _ := testHostKey(t) // different key from pinned by construction
	var calls int
	restore := SetHostKeyDialForTest(func(addr string, algos []string) (HostKey, error) {
		calls++
		if calls == 1 {
			return presented, nil // the restricted verification dial
		}
		return HostKey{}, errors.New("capture boom") // the unrestricted re-capture, every family
	})
	defer restore()

	err := verifyPinnedHostKey("host.example", 22, []HostKey{pinned})
	var changed *ErrHostKeyChanged
	if !errors.As(err, &changed) {
		t.Fatalf("want ErrHostKeyChanged, got %v", err)
	}
	if len(changed.Captured) != 1 || changed.Captured[0] != presented {
		t.Fatalf("fallback capture must carry the presented key, got %v", changed.Captured)
	}
}

func TestSetHostKeyDialForTestRestores(t *testing.T) {
	sentinel := HostKey{Type: "ssh-ed25519", Base64: "c3R1Yg=="}
	restore := SetHostKeyDialForTest(func(addr string, algos []string) (HostKey, error) {
		return sentinel, nil
	})
	k, err := hostKeyDial("ignored:22", nil)
	if err != nil || k != sentinel {
		t.Fatalf("stub not active: %v %v", k, err)
	}
	restore()
}
