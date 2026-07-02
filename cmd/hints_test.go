package cmd

import (
	"errors"
	"strings"
	"testing"

	"github.com/rtxnik/lazyray/internal/clihint"
)

func TestSharedHintConstructors(t *testing.T) {
	cases := []struct {
		name     string
		err      error
		wantMsg  string
		wantHint string
	}{
		{"no profiles", errNoProfilesConfigured(), "no profiles configured", "lzr import <url>"},
		{"not found", errProfileNotFound("home"), `profile "home" not found`, "lzr config list"},
		{"no default", errNoDefaultProfile(), "no default profile", "lzr config switch"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var ce *clihint.Error
			if !errors.As(tc.err, &ce) {
				t.Fatalf("%v does not wrap *clihint.Error", tc.err)
			}
			if ce.Msg != tc.wantMsg {
				t.Errorf("Msg = %q, want %q", ce.Msg, tc.wantMsg)
			}
			if !strings.Contains(ce.Hint, tc.wantHint) {
				t.Errorf("Hint = %q, want substring %q", ce.Hint, tc.wantHint)
			}
		})
	}
}

// TestTunnelSSHAlias asserts the backward-compatible alias resolves to tunnel.
func TestTunnelSSHAlias(t *testing.T) {
	c, _, err := RootCmd().Find([]string{"ssh-tunnel"})
	if err != nil {
		t.Fatalf("Find(ssh-tunnel) error: %v", err)
	}
	if c.Name() != "tunnel" {
		t.Errorf("alias resolved to %q, want \"tunnel\"", c.Name())
	}
	if strings.TrimSpace(c.Long) == "" {
		t.Error("tunnel Long is empty")
	}
}
