package cmd

import (
	"errors"
	"strings"
	"testing"

	"github.com/rtxnik/lazyray/internal/clihint"
	"github.com/rtxnik/lazyray/internal/config"
)

// TestTestingFamilyHints asserts that the user-actionable error sites in the
// testing family (test/speedtest/stats) carry the canonical clihint hints.
func TestTestingFamilyHints(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		wantHint string
	}{
		{
			name:     "test profile not found",
			err:      errTestProfileNotFound("missing"),
			wantHint: "list profiles with 'lzr config list'",
		},
		{
			name:     "test no default profile",
			err:      errTestNoProfiles(),
			wantHint: "import a profile with 'lzr import <url>'",
		},
		{
			name:     "test all no profiles",
			err:      testAllProfiles(&config.ServersConfig{}),
			wantHint: "import a profile with 'lzr import <url>'",
		},
		{
			name:     "test connection FAIL",
			err:      errTestConnFailed(errors.New("dial tcp: timeout")),
			wantHint: "diagnose with 'lzr doctor'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err == nil {
				t.Fatalf("expected an error, got nil")
			}
			var ce *clihint.Error
			if !errors.As(tt.err, &ce) {
				t.Fatalf("error %v does not unwrap to *clihint.Error", tt.err)
			}
			if !strings.Contains(ce.Hint, tt.wantHint) {
				t.Errorf("hint = %q, want substring %q", ce.Hint, tt.wantHint)
			}
		})
	}
}
