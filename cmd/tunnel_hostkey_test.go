//go:build !windows

package cmd

import (
	"crypto/ed25519"
	"encoding/base64"
	"os/exec"
	"strings"
	"testing"

	"github.com/rtxnik/lazyray/internal/config"
	"github.com/rtxnik/lazyray/internal/core"
	"golang.org/x/crypto/ssh"
)

func trustTestServers(hostKeys []string) *config.ServersConfig {
	return &config.ServersConfig{ConfigVersion: config.CurrentConfigVersion, Profiles: []config.Profile{
		{
			Name:   "Alpha",
			Server: config.ServerConfig{Address: "203.0.113.7", Port: 443, UUID: "u1", Encryption: "none"},
			SSH: config.SSHConfig{Host: "203.0.113.7", Port: 22, User: "root", KeyPath: "/k",
				HostKeys: hostKeys, Panel: config.PanelConfig{Port: 2053}},
		},
		{
			Name:   "al",
			Server: config.ServerConfig{Address: "203.0.113.8", Port: 443, UUID: "u2", Encryption: "none"},
			SSH: config.SSHConfig{Host: "203.0.113.8", Port: 22, User: "root", KeyPath: "/k",
				Panel: config.PanelConfig{Port: 2053}},
		},
	}}
}

func stubTrustSeams(t *testing.T, tty bool, answer string, key core.HostKey) *[]string {
	t.Helper()
	prevTTY := stdinIsTerminal
	stdinIsTerminal = func() bool { return tty }
	prevRead := readTrustLine
	readTrustLine = func() (string, error) { return answer, nil }
	t.Cleanup(func() { stdinIsTerminal = prevTTY; readTrustLine = prevRead })

	restoreDial := core.SetHostKeyDialForTest(func(addr string, algos []string) (core.HostKey, error) {
		return key, nil
	})
	t.Cleanup(restoreDial)

	var spawnedArgs []string
	restoreSpawn := core.SetStartSSHProcessForTest(func(args []string) (*exec.Cmd, error) {
		spawnedArgs = args
		cmd := exec.Command("sleep", "30")
		if err := cmd.Start(); err != nil {
			return nil, err
		}
		return cmd, nil
	})
	t.Cleanup(restoreSpawn)
	t.Cleanup(func() { tunnelManager.DisconnectAll() })
	return &spawnedArgs
}

func testCoreHostKey(t *testing.T) core.HostKey {
	t.Helper()
	// Any valid key works; reuse the capture stub shape via a fixed ed25519 key.
	keys, err := core.ParseHostKeys([]string{testEd25519Line(t)})
	if err != nil {
		t.Fatal(err)
	}
	return keys[0]
}

func TestFindTunnelProfileExactBeatsPrefix(t *testing.T) {
	servers := trustTestServers(nil)
	p := findTunnelProfile(servers, "al")
	if p == nil || p.Name != "al" {
		t.Fatalf("exact match must win over earlier prefix match, got %+v", p)
	}
	if q := findTunnelProfile(servers, "alp"); q == nil || q.Name != "Alpha" {
		t.Fatalf("prefix fallback broken, got %+v", q)
	}
	if findTunnelProfile(servers, "zzz") != nil {
		t.Fatal("unknown name must return nil")
	}
}

func TestTunnelConnectUnknownHostNonTTYRefusesWithHint(t *testing.T) {
	cleanup := setupTestHome(t)
	defer cleanup()
	_ = stubTrustSeams(t, false, "", testCoreHostKey(t))
	servers := trustTestServers(nil)
	err := tunnelConnectByName(servers, "alpha")
	if err == nil {
		t.Fatal("non-TTY unknown host must refuse")
	}
	if !strings.Contains(err.Error(), "not trusted") && !strings.Contains(err.Error(), "trust") {
		t.Fatalf("refusal must point at the trust flow: %v", err)
	}
	if len(servers.Profiles[0].SSH.HostKeys) != 0 {
		t.Fatal("nothing may be pinned without confirmation")
	}
}

func TestTunnelConnectUnknownHostTTYAcceptPinsAndConnects(t *testing.T) {
	cleanup := setupTestHome(t)
	defer cleanup()
	key := testCoreHostKey(t)
	spawned := stubTrustSeams(t, true, "y\n", key)
	servers := trustTestServers(nil)
	if err := config.SaveServers(servers); err != nil {
		t.Fatal(err)
	}
	if err := tunnelConnectByName(servers, "alpha"); err != nil {
		t.Fatal(err)
	}
	if len(*spawned) == 0 {
		t.Fatal("accepting the prompt must connect")
	}
	loaded, err := config.LoadServers()
	if err != nil {
		t.Fatal(err)
	}
	if got := loaded.Profiles[0].SSH.HostKeys; len(got) != 1 || got[0] != key.String() {
		t.Fatalf("pin must persist to disk, got %v", got)
	}
}

func TestTunnelConnectUnknownHostTTYDeclineAborts(t *testing.T) {
	cleanup := setupTestHome(t)
	defer cleanup()
	spawned := stubTrustSeams(t, true, "n\n", testCoreHostKey(t))
	servers := trustTestServers(nil)
	if err := tunnelConnectByName(servers, "alpha"); err == nil {
		t.Fatal("declining the prompt must abort")
	}
	if len(*spawned) != 0 {
		t.Fatal("declined trust must not spawn ssh")
	}
}

func TestTunnelConnectChangedKeyAlwaysRefuses(t *testing.T) {
	cleanup := setupTestHome(t)
	defer cleanup()
	oldKey := testCoreHostKey(t)
	newKey := testCoreHostKeySecond(t)
	spawned := stubTrustSeams(t, true, "y\n", newKey)
	servers := trustTestServers([]string{oldKey.String()})
	err := tunnelConnectByName(servers, "alpha")
	if err == nil {
		t.Fatal("changed key must refuse even on a TTY")
	}
	if !strings.Contains(err.Error(), "changed") {
		t.Fatalf("error must name the key change: %v", err)
	}
	if len(*spawned) != 0 {
		t.Fatal("changed key must never spawn ssh from the connect command")
	}
	if got := servers.Profiles[0].SSH.HostKeys; len(got) != 1 || got[0] != oldKey.String() {
		t.Fatal("the old pin must not be replaced by the connect command")
	}
}

// testEd25519Line returns a valid "<type> <base64>" host-key line.
func testEd25519Line(t *testing.T) string {
	t.Helper()
	return testEd25519LineSeed(t, 1)
}

func testEd25519LineSeed(t *testing.T, seed byte) string {
	t.Helper()
	var raw [32]byte
	raw[0] = seed
	pub := make([]byte, 32)
	copy(pub, raw[:])
	sshPub, err := sshNewEd25519(pub)
	if err != nil {
		t.Fatal(err)
	}
	return sshPub
}

func testCoreHostKeySecond(t *testing.T) core.HostKey {
	t.Helper()
	keys, err := core.ParseHostKeys([]string{testEd25519LineSeed(t, 2)})
	if err != nil {
		t.Fatal(err)
	}
	return keys[0]
}

func sshNewEd25519(pub []byte) (string, error) {
	sshPub, err := ssh.NewPublicKey(ed25519.PublicKey(pub))
	if err != nil {
		return "", err
	}
	return sshPub.Type() + " " + base64.StdEncoding.EncodeToString(sshPub.Marshal()), nil
}
