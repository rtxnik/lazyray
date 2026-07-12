//go:build !windows

package modals

import (
	"crypto/ed25519"
	"encoding/base64"
	"os/exec"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/rtxnik/lazyray/internal/config"
	"github.com/rtxnik/lazyray/internal/core"
	"golang.org/x/crypto/ssh"
)

func modalHostKey(t *testing.T, seed byte) core.HostKey {
	t.Helper()
	pub := make([]byte, ed25519.PublicKeySize)
	pub[0] = seed
	sshPub, err := coreSSHPublicKey(pub)
	if err != nil {
		t.Fatal(err)
	}
	return core.HostKey{Type: sshPub.Type(), Base64: base64.StdEncoding.EncodeToString(sshPub.Marshal())}
}

func modalServers(hostKeys []string) *config.ServersConfig {
	return &config.ServersConfig{ConfigVersion: config.CurrentConfigVersion, Profiles: []config.Profile{{
		Name:   "ru",
		Server: config.ServerConfig{Address: "203.0.113.7", Port: 443, UUID: "u", Encryption: "none"},
		SSH: config.SSHConfig{Host: "203.0.113.7", Port: 22, User: "root", KeyPath: "/k",
			HostKeys: hostKeys, Panel: config.PanelConfig{Port: 2053}},
	}}}
}

func stubModalSeams(t *testing.T, live core.HostKey) {
	t.Helper()
	restoreDial := core.SetHostKeyDialForTest(func(addr string, algos []string) (core.HostKey, error) {
		return live, nil
	})
	t.Cleanup(restoreDial)
	restoreSpawn := core.SetStartSSHProcessForTest(func(args []string) (*exec.Cmd, error) {
		cmd := exec.Command("sleep", "30")
		if err := cmd.Start(); err != nil {
			return nil, err
		}
		return cmd, nil
	})
	t.Cleanup(restoreSpawn)
}

func key(s string) tea.KeyMsg {
	if s == "enter" {
		return tea.KeyMsg{Type: tea.KeyEnter}
	}
	if s == "esc" {
		return tea.KeyMsg{Type: tea.KeyEsc}
	}
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

func TestModalUnknownHostShowsTrustPromptAndPinsOnYes(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("APPDATA", home)
	t.Setenv("LOCALAPPDATA", home)
	if err := config.EnsureDirs(); err != nil {
		t.Fatal(err)
	}
	live := modalHostKey(t, 1)
	stubModalSeams(t, live)
	servers := modalServers(nil)
	if err := config.SaveServers(servers); err != nil {
		t.Fatal(err)
	}
	tunnels := core.NewTunnelManager()
	t.Cleanup(tunnels.DisconnectAll)

	m := NewTunnelModal(servers, tunnels, 100, 40)
	m.toggleIndex(0)
	if m.state != tunnelStateTrustPrompt {
		t.Fatalf("state = %v, want trust prompt", m.state)
	}
	view := m.View()
	fp, _ := live.Fingerprint()
	if !strings.Contains(view, fp) {
		t.Fatal("trust prompt must show the fingerprint")
	}

	_, _ = m.Update(key("y"))
	if m.state != tunnelStateList {
		t.Fatal("confirming must return to the list state")
	}
	loaded, err := config.LoadServers()
	if err != nil {
		t.Fatal(err)
	}
	if got := loaded.Profiles[0].SSH.HostKeys; len(got) != 1 || got[0] != live.String() {
		t.Fatalf("pin must persist, got %v", got)
	}
}

func TestModalUnknownHostEscDeclines(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("APPDATA", home)
	t.Setenv("LOCALAPPDATA", home)
	if err := config.EnsureDirs(); err != nil {
		t.Fatal(err)
	}
	stubModalSeams(t, modalHostKey(t, 1))
	servers := modalServers(nil)
	tunnels := core.NewTunnelManager()
	t.Cleanup(tunnels.DisconnectAll)

	m := NewTunnelModal(servers, tunnels, 100, 40)
	m.toggleIndex(0)
	_, _ = m.Update(key("esc"))
	if m.state != tunnelStateList {
		t.Fatal("esc must cancel the trust prompt")
	}
	if len(servers.Profiles[0].SSH.HostKeys) != 0 {
		t.Fatal("declining must not pin")
	}
	if m.Done {
		t.Fatal("declining the trust prompt must not close the whole modal")
	}
}

func TestModalChangedKeyShowsOldAndNew(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("APPDATA", home)
	t.Setenv("LOCALAPPDATA", home)
	if err := config.EnsureDirs(); err != nil {
		t.Fatal(err)
	}
	oldKey := modalHostKey(t, 1)
	newKey := modalHostKey(t, 2)
	stubModalSeams(t, newKey)
	servers := modalServers([]string{oldKey.String()})
	if err := config.SaveServers(servers); err != nil {
		t.Fatal(err)
	}
	tunnels := core.NewTunnelManager()
	t.Cleanup(tunnels.DisconnectAll)

	m := NewTunnelModal(servers, tunnels, 100, 40)
	m.toggleIndex(0)
	if m.state != tunnelStateKeyChanged {
		t.Fatalf("state = %v, want key-changed", m.state)
	}
	view := m.View()
	oldFP, _ := oldKey.Fingerprint()
	newFP, _ := newKey.Fingerprint()
	if !strings.Contains(view, oldFP) || !strings.Contains(view, newFP) {
		t.Fatal("key-changed view must show old and new fingerprints")
	}

	_, _ = m.Update(key("y"))
	loaded, err := config.LoadServers()
	if err != nil {
		t.Fatal(err)
	}
	if got := loaded.Profiles[0].SSH.HostKeys; len(got) != 1 || got[0] != newKey.String() {
		t.Fatalf("explicit re-trust must replace the pin, got %v", got)
	}
}

func TestModalConfirmWithEmptyCaptureKeepsPin(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("APPDATA", home)
	t.Setenv("LOCALAPPDATA", home)
	if err := config.EnsureDirs(); err != nil {
		t.Fatal(err)
	}
	oldKey := modalHostKey(t, 1)
	servers := modalServers([]string{oldKey.String()})
	if err := config.SaveServers(servers); err != nil {
		t.Fatal(err)
	}
	tunnels := core.NewTunnelManager()

	m := NewTunnelModal(servers, tunnels, 100, 40)
	// Simulate a key-changed state whose fallback capture failed (no live keys).
	m.state = tunnelStateKeyChanged
	m.pendingName = "ru"
	m.pendingOld = []core.HostKey{oldKey}
	m.pendingNew = nil

	_, _ = m.Update(key("y"))
	if m.state != tunnelStateList {
		t.Fatal("confirm with empty capture must return to the list state")
	}
	if got := servers.Profiles[0].SSH.HostKeys; len(got) != 1 || got[0] != oldKey.String() {
		t.Fatalf("empty capture must never clear the existing pin, got %v", got)
	}
}

func coreSSHPublicKey(pub ed25519.PublicKey) (ssh.PublicKey, error) {
	return ssh.NewPublicKey(pub)
}
