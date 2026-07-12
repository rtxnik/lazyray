package core

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"strings"
	"testing"

	"golang.org/x/crypto/ssh"
)

func testHostKey(t *testing.T) (HostKey, ssh.PublicKey) {
	t.Helper()
	pub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	sshPub, err := ssh.NewPublicKey(pub)
	if err != nil {
		t.Fatal(err)
	}
	return HostKey{
		Type:   sshPub.Type(),
		Base64: base64.StdEncoding.EncodeToString(sshPub.Marshal()),
	}, sshPub
}

func TestHostKeyFingerprintMatchesOpenSSHForm(t *testing.T) {
	k, sshPub := testHostKey(t)
	fp, err := k.Fingerprint()
	if err != nil {
		t.Fatal(err)
	}
	if fp != ssh.FingerprintSHA256(sshPub) {
		t.Fatalf("fingerprint mismatch: %s", fp)
	}
	if !strings.HasPrefix(fp, "SHA256:") {
		t.Fatalf("want OpenSSH SHA256 form, got %s", fp)
	}
}

func TestParseHostKeysRejectsMalformed(t *testing.T) {
	for _, bad := range []string{"", "ssh-ed25519", "ssh-ed25519 not-base64 extra", "ssh-ed25519 !!!"} {
		if _, err := ParseHostKeys([]string{bad}); err == nil {
			t.Errorf("ParseHostKeys(%q) must fail", bad)
		}
	}
	k, _ := testHostKey(t)
	keys, err := ParseHostKeys([]string{k.String()})
	if err != nil || len(keys) != 1 || keys[0] != k {
		t.Fatalf("round-trip failed: %v %v", keys, err)
	}
}

func TestKnownHostsToken(t *testing.T) {
	cases := []struct {
		host string
		port int
		want string
	}{
		{"Example.COM", 22, "example.com"},
		{"example.com", 2222, "[example.com]:2222"},
		{"2001:db8::1", 22, "2001:db8::1"},          // unbracketed on default port
		{"2001:db8::1", 2222, "[2001:db8::1]:2222"}, // bracketed only with a port
	}
	for _, c := range cases {
		if got := KnownHostsToken(c.host, c.port); got != c.want {
			t.Errorf("KnownHostsToken(%q,%d) = %q, want %q", c.host, c.port, got, c.want)
		}
	}
}

func TestDeriveKnownHosts(t *testing.T) {
	k, _ := testHostKey(t)
	content := string(DeriveKnownHosts("host.example", 2222, []HostKey{k}))
	want := "[host.example]:2222 " + k.String() + "\n"
	if content != want {
		t.Fatalf("derived content:\n%q\nwant:\n%q", content, want)
	}
}

func TestValidateSSHTarget(t *testing.T) {
	if err := ValidateSSHTarget("root", "host"); err != nil {
		t.Fatalf("valid target rejected: %v", err)
	}
	if err := ValidateSSHTarget("-oProxyCommand=evil", "host"); err == nil {
		t.Error("leading-dash user must be rejected")
	}
	if err := ValidateSSHTarget("root", "-oProxyCommand=evil"); err == nil {
		t.Error("leading-dash host must be rejected")
	}
}

func TestHostKeyAlgorithmsFor(t *testing.T) {
	got := hostKeyAlgorithmsFor([]HostKey{
		{Type: "ssh-ed25519"}, {Type: "ssh-rsa"}, {Type: "ssh-ed25519"},
	})
	want := []string{"ssh-ed25519", "rsa-sha2-512", "rsa-sha2-256", "ssh-rsa"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("algorithms = %v, want %v", got, want)
	}
}

func TestTypedErrorsAreDistinguishable(t *testing.T) {
	var uErr error = &ErrHostKeyUnknown{Host: "h", Port: 22}
	var cErr error = &ErrHostKeyChanged{Host: "h", Port: 22}
	if uErr.Error() == "" || cErr.Error() == "" {
		t.Fatal("errors must render messages")
	}
	if !strings.Contains(cErr.Error(), "changed") {
		t.Fatalf("changed-key error must say so: %s", cErr.Error())
	}
}
