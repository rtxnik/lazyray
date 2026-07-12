package config

import (
	"os"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func testHome(t *testing.T) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("APPDATA", home)
	t.Setenv("LOCALAPPDATA", home)
	if err := EnsureDirs(); err != nil {
		t.Fatal(err)
	}
}

func sshProfile(name string, hostKeys []string) Profile {
	return Profile{
		Name:   name,
		Server: ServerConfig{Address: "203.0.113.7", Port: 443, UUID: "u", Encryption: "none"},
		SSH: SSHConfig{
			Host: "203.0.113.7", Port: 22, User: "root", KeyPath: "/k",
			HostKeys: hostKeys,
			Panel:    PanelConfig{Port: 2053, Path: "/panel"},
		},
	}
}

func TestHostKeysSurviveSaveLoad(t *testing.T) {
	testHome(t)
	pin := []string{"ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIPlaceholderKeyBody0000000000000000000000"}
	cfg := &ServersConfig{Profiles: []Profile{sshProfile("ru", pin)}}
	if err := SaveServers(cfg); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadServers()
	if err != nil {
		t.Fatal(err)
	}
	got := loaded.Profiles[0].SSH.HostKeys
	if len(got) != 1 || got[0] != pin[0] {
		t.Fatalf("HostKeys did not survive round-trip: %#v", got)
	}
}

func TestHostKeysOmittedWhenEmpty(t *testing.T) {
	cfg := &ServersConfig{Profiles: []Profile{sshProfile("ru", nil)}}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "hostKeys") {
		t.Fatalf("empty HostKeys must be omitted from YAML:\n%s", data)
	}
}

func TestCloneDeepCopiesHostKeys(t *testing.T) {
	p := sshProfile("ru", []string{"ssh-ed25519 AAAA-original"})
	c := p.Clone()
	c.SSH.HostKeys[0] = "ssh-ed25519 AAAA-mutated"
	if p.SSH.HostKeys[0] != "ssh-ed25519 AAAA-original" {
		t.Fatal("Clone shares HostKeys backing array with the original")
	}
}

func TestTunnelKnownHostsPathIsPerProfile(t *testing.T) {
	a := TunnelKnownHostsPath("prod/eu")
	b := TunnelKnownHostsPath("prod_eu")
	if a == b {
		t.Fatalf("distinct profile names map to the same trust file: %s", a)
	}
	for _, p := range []string{a, b} {
		base := p[strings.LastIndex(p, string(os.PathSeparator))+1:]
		if !strings.HasPrefix(base, "tunnel-prod_eu-") || !strings.HasSuffix(base, ".known_hosts") {
			t.Fatalf("unexpected known_hosts filename: %s", base)
		}
	}
}
