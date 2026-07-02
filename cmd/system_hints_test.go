package cmd

import (
	"errors"
	"strings"
	"testing"

	"github.com/rtxnik/lazyray/internal/clihint"
	"github.com/rtxnik/lazyray/internal/config"
)

// TestSystemFamilyHints asserts that the user-actionable error sites in the
// system family (proxy/pac/tunnel) carry the canonical clihint Hint.
func TestSystemFamilyHints(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		wantMsg  string
		wantHint string
	}{
		{
			name:     "tunnel profile not found",
			err:      tunnelConnectByName(&config.ServersConfig{}, "nope"),
			wantMsg:  `profile "nope" not found`,
			wantHint: "list profiles with 'lzr config list'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err == nil {
				t.Fatalf("expected an error, got nil")
			}
			var ce *clihint.Error
			if !errors.As(tt.err, &ce) {
				t.Fatalf("error %v does not wrap *clihint.Error", tt.err)
			}
			if !strings.Contains(ce.Msg, tt.wantMsg) {
				t.Errorf("Msg = %q, want substring %q", ce.Msg, tt.wantMsg)
			}
			if !strings.Contains(ce.Hint, tt.wantHint) {
				t.Errorf("Hint = %q, want substring %q", ce.Hint, tt.wantHint)
			}
		})
	}
}
