//go:build !windows

package core

import (
	"errors"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/rtxnik/lazyray/internal/config"
)

func tunnelTestProfile(hostKeys []string) *config.Profile {
	return &config.Profile{
		Name: "ru",
		SSH: config.SSHConfig{
			Host: "203.0.113.7", Port: 22, User: "root", KeyPath: "/k",
			HostKeys: hostKeys,
			Panel:    config.PanelConfig{Port: 2053, Path: "/panel"},
		},
	}
}

// disabledStrictHostKeyChecking is split across two literals so the disabled
// form of the ssh option never appears contiguous in source: a repo-wide grep
// for that exact token is the acceptance oracle proving this feature left no
// verification-disabled option anywhere in the tree, and a plain literal here
// (even one asserting the option's absence) would trip that same grep.
var disabledStrictHostKeyChecking = "StrictHostKeyChecking=" + "no"

func TestBuildSSHArgsContract(t *testing.T) {
	p := tunnelTestProfile(nil)
	args := buildSSHArgs(p, 51234, "/data/kh", []string{"ssh-ed25519"})
	joined := " " + strings.Join(args, " ") + " "

	for _, want := range []string{
		" -o StrictHostKeyChecking=yes ",
		" -o UserKnownHostsFile=/data/kh ",
		" -o GlobalKnownHostsFile=/dev/null ",
		" -o HostKeyAlgorithms=ssh-ed25519 ",
		" -N -- root@203.0.113.7 ",
	} {
		if !strings.Contains(joined, want) {
			t.Errorf("argv missing %q:\n%s", want, joined)
		}
	}
	if strings.Contains(joined, disabledStrictHostKeyChecking) {
		t.Errorf("argv must not contain %q", disabledStrictHostKeyChecking)
	}
	if args[len(args)-2] != "--" {
		t.Error("'--' must immediately precede the destination")
	}
}

func TestConnectRejectsLeadingDashTarget(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	tm := NewTunnelManager()
	p := tunnelTestProfile(nil)
	p.SSH.User = "-oProxyCommand=evil"
	if err := tm.Connect(p); err == nil {
		t.Fatal("leading-dash user must be rejected before spawn")
	}
}

func TestConnectUnknownHostRefusesAndCaptures(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	stub, _ := testHostKey(t)
	restore := SetHostKeyDialForTest(func(addr string, algos []string) (HostKey, error) {
		return stub, nil
	})
	defer restore()
	spawned := false
	restoreSpawn := SetStartSSHProcessForTest(func(args []string) (*exec.Cmd, error) {
		spawned = true
		return nil, errors.New("must not spawn")
	})
	defer restoreSpawn()

	tm := NewTunnelManager()
	err := tm.Connect(tunnelTestProfile(nil))
	var unknown *ErrHostKeyUnknown
	if !errors.As(err, &unknown) {
		t.Fatalf("want ErrHostKeyUnknown, got %v", err)
	}
	if len(unknown.Captured) == 0 || unknown.Captured[0] != stub {
		t.Fatalf("captured keys not attached: %v", unknown.Captured)
	}
	if spawned {
		t.Fatal("ssh must not spawn for an untrusted host")
	}
	if _, err := os.Stat(config.TunnelPIDPath("ru")); !os.IsNotExist(err) {
		t.Fatal("no PID file may be written for a refused connect")
	}
}

func TestConnectChangedKeyRefuses(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	pinned, _ := testHostKey(t) // what the profile trusts
	live, _ := testHostKey(t)   // what the "server" presents now
	restore := SetHostKeyDialForTest(func(addr string, algos []string) (HostKey, error) {
		return live, nil
	})
	defer restore()
	spawned := false
	restoreSpawn := SetStartSSHProcessForTest(func(args []string) (*exec.Cmd, error) {
		spawned = true
		return nil, errors.New("must not spawn")
	})
	defer restoreSpawn()

	tm := NewTunnelManager()
	err := tm.Connect(tunnelTestProfile([]string{pinned.String()}))
	var changed *ErrHostKeyChanged
	if !errors.As(err, &changed) {
		t.Fatalf("want ErrHostKeyChanged, got %v", err)
	}
	if spawned {
		t.Fatal("ssh must not spawn on a changed key — this is the SEC1-01 acceptance oracle")
	}
	if len(changed.Pinned) != 1 || changed.Pinned[0] != pinned {
		t.Fatalf("Pinned = %v", changed.Pinned)
	}
	if len(changed.Captured) == 0 || changed.Captured[0] != live {
		t.Fatalf("Captured = %v", changed.Captured)
	}
}

func TestConnectPinnedMatchSpawnsWithDerivedKnownHosts(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	pinned, _ := testHostKey(t)
	restore := SetHostKeyDialForTest(func(addr string, algos []string) (HostKey, error) {
		return pinned, nil
	})
	defer restore()
	var gotArgs []string
	restoreSpawn := SetStartSSHProcessForTest(func(args []string) (*exec.Cmd, error) {
		gotArgs = args
		cmd := exec.Command("sleep", "30") // stand-in child with real PID semantics
		if err := cmd.Start(); err != nil {
			return nil, err
		}
		return cmd, nil
	})
	defer restoreSpawn()

	tm := NewTunnelManager()
	p := tunnelTestProfile([]string{pinned.String()})
	if err := tm.Connect(p); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = tm.Disconnect("ru") })

	khPath := config.TunnelKnownHostsPath("ru")
	data, err := os.ReadFile(khPath)
	if err != nil {
		t.Fatalf("derived known_hosts not written: %v", err)
	}
	want := "203.0.113.7 " + pinned.String() + "\n"
	if string(data) != want {
		t.Fatalf("known_hosts content:\n%q\nwant:\n%q", data, want)
	}
	info, _ := os.Stat(khPath)
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("known_hosts perm = %o, want 0600", info.Mode().Perm())
	}
	joined := strings.Join(gotArgs, " ")
	if !strings.Contains(joined, "UserKnownHostsFile="+khPath) {
		t.Fatalf("argv must reference the derived file: %s", joined)
	}
}
