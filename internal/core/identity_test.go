//go:build !windows

package core

import (
	"os"
	"testing"

	"github.com/rtxnik/lazyray/internal/config"
)

func TestIsOurXray_MatchesManagedBinary(t *testing.T) {
	restore := SetProcessCmdlineForTest(func(int) (string, error) {
		return config.XrayBinaryPath() + " run -c cfg.json", nil
	})
	defer restore()
	if !IsOurXray(4242) {
		t.Error("IsOurXray should match the managed xray binary path")
	}
}

func TestIsOurXray_RejectsForeignAndNonPositive(t *testing.T) {
	restore := SetProcessCmdlineForTest(func(int) (string, error) {
		return "/usr/bin/something-else --x", nil
	})
	defer restore()
	if IsOurXray(4242) {
		t.Error("IsOurXray should reject a foreign cmdline")
	}
	if IsOurXray(0) {
		t.Error("IsOurXray(0) must be false")
	}
}

func TestIsOurTunnel_MatchesSSHForward(t *testing.T) {
	restore := SetProcessCmdlineForTest(func(int) (string, error) {
		return "ssh -L 51234:127.0.0.1:443 -p 22 -i /k -o StrictHostKeyChecking=yes -N -- user@host", nil
	})
	defer restore()
	if !IsOurTunnel(4242) {
		t.Error("IsOurTunnel should match an `ssh -N -L` port-forward")
	}
}

func TestIsOurTunnel_RejectsNonTunnel(t *testing.T) {
	cases := map[string]string{
		"plain ssh (no -N/-L)": "ssh user@host",
		"sshd daemon":          "/usr/sbin/sshd -D",
		"foreign process":      "/usr/bin/python server.py",
	}
	for name, cmd := range cases {
		cmd := cmd
		restore := SetProcessCmdlineForTest(func(int) (string, error) { return cmd, nil })
		if IsOurTunnel(4242) {
			t.Errorf("IsOurTunnel should reject %s", name)
		}
		restore()
	}
	if IsOurTunnel(0) {
		t.Error("IsOurTunnel(0) must be false")
	}
}

func TestIdentity_CmdlineReadErrorIsFalse(t *testing.T) {
	restore := SetProcessCmdlineForTest(func(int) (string, error) {
		return "", os.ErrNotExist
	})
	defer restore()
	if IsOurXray(4242) || IsOurTunnel(4242) {
		t.Error("identity must be false when the cmdline cannot be read")
	}
}
