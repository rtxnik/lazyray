package core

import (
	"strings"
	"testing"

	"github.com/rtxnik/lazyray/internal/config"
)

func TestBuildSSHArgsSetsExitOnForwardFailure(t *testing.T) {
	p := &config.Profile{}
	p.SSH.Port = 22
	p.SSH.Panel.Port = 8080
	args := buildSSHArgs(p, 10800, "/tmp/kh", []string{"ssh-ed25519"})
	if !strings.Contains(strings.Join(args, " "), "ExitOnForwardFailure=yes") {
		t.Fatalf("buildSSHArgs missing ExitOnForwardFailure=yes: %v", args)
	}
}
