//go:build !windows

package cmd

import (
	"strings"
	"testing"

	"github.com/rtxnik/lazyray/internal/config"
)

func resetTrustFlags(t *testing.T) {
	t.Helper()
	prev := tunnelTrustFingerprints
	t.Cleanup(func() {
		tunnelTrustFingerprints = prev
		f := tunnelTrustCmd.Flags().Lookup("fingerprint")
		f.Changed = false
	})
	tunnelTrustFingerprints = nil
}

func TestTunnelTrustFingerprintMatchPinsSubset(t *testing.T) {
	cleanup := setupTestHome(t)
	defer cleanup()
	resetTrustFlags(t)
	key := testCoreHostKey(t)
	_ = stubTrustSeams(t, false, "", key)
	servers := trustTestServers(nil)
	if err := config.SaveServers(servers); err != nil {
		t.Fatal(err)
	}
	fp, err := key.Fingerprint()
	if err != nil {
		t.Fatal(err)
	}
	if err := tunnelTrust(servers, "alpha", []string{fp}); err != nil {
		t.Fatal(err)
	}
	loaded, err := config.LoadServers()
	if err != nil {
		t.Fatal(err)
	}
	if got := loaded.Profiles[0].SSH.HostKeys; len(got) != 1 || got[0] != key.String() {
		t.Fatalf("verified fingerprint must pin exactly that key, got %v", got)
	}
}

func TestTunnelTrustFingerprintDedupsRepeated(t *testing.T) {
	cleanup := setupTestHome(t)
	defer cleanup()
	resetTrustFlags(t)
	key := testCoreHostKey(t)
	_ = stubTrustSeams(t, false, "", key)
	servers := trustTestServers(nil)
	if err := config.SaveServers(servers); err != nil {
		t.Fatal(err)
	}
	fp, err := key.Fingerprint()
	if err != nil {
		t.Fatal(err)
	}
	if err := tunnelTrust(servers, "alpha", []string{fp, fp}); err != nil {
		t.Fatal(err)
	}
	loaded, err := config.LoadServers()
	if err != nil {
		t.Fatal(err)
	}
	if got := loaded.Profiles[0].SSH.HostKeys; len(got) != 1 || got[0] != key.String() {
		t.Fatalf("repeating the same fingerprint must pin exactly one key, got %v", got)
	}
}

func TestTunnelTrustFingerprintMismatchPinsNothing(t *testing.T) {
	cleanup := setupTestHome(t)
	defer cleanup()
	resetTrustFlags(t)
	_ = stubTrustSeams(t, false, "", testCoreHostKey(t))
	servers := trustTestServers(nil)
	err := tunnelTrust(servers, "alpha", []string{"SHA256:doesnotmatchanything0000000000000000000000000"})
	if err == nil || !strings.Contains(err.Error(), "does not match") {
		t.Fatalf("mismatched fingerprint must error, got %v", err)
	}
	if len(servers.Profiles[0].SSH.HostKeys) != 0 {
		t.Fatal("nothing may be pinned on mismatch")
	}
}

func TestTunnelTrustNonTTYWithoutFingerprintRefuses(t *testing.T) {
	cleanup := setupTestHome(t)
	defer cleanup()
	resetTrustFlags(t)
	_ = stubTrustSeams(t, false, "", testCoreHostKey(t))
	servers := trustTestServers(nil)
	if err := tunnelTrust(servers, "alpha", nil); err == nil {
		t.Fatal("non-TTY without --fingerprint must refuse")
	}
}

func TestTunnelTrustInteractiveRepinShowsOldAndPins(t *testing.T) {
	cleanup := setupTestHome(t)
	defer cleanup()
	resetTrustFlags(t)
	oldKey := testCoreHostKey(t)
	newKey := testCoreHostKeySecond(t)
	_ = stubTrustSeams(t, true, "y\n", newKey)
	servers := trustTestServers([]string{oldKey.String()})
	if err := config.SaveServers(servers); err != nil {
		t.Fatal(err)
	}
	if err := tunnelTrust(servers, "alpha", nil); err != nil {
		t.Fatal(err)
	}
	loaded, err := config.LoadServers()
	if err != nil {
		t.Fatal(err)
	}
	if got := loaded.Profiles[0].SSH.HostKeys; len(got) != 1 || got[0] != newKey.String() {
		t.Fatalf("re-pin must replace the old key, got %v", got)
	}
}

func TestTunnelTrustDeclineKeepsOldPin(t *testing.T) {
	cleanup := setupTestHome(t)
	defer cleanup()
	resetTrustFlags(t)
	oldKey := testCoreHostKey(t)
	_ = stubTrustSeams(t, true, "n\n", testCoreHostKeySecond(t))
	servers := trustTestServers([]string{oldKey.String()})
	if err := tunnelTrust(servers, "alpha", nil); err == nil {
		t.Fatal("declined confirmation must error")
	}
	if got := servers.Profiles[0].SSH.HostKeys; len(got) != 1 || got[0] != oldKey.String() {
		t.Fatal("declining must keep the previous pin")
	}
}
