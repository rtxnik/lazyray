package cmd

import (
	"errors"
	"strings"
	"testing"

	"github.com/rtxnik/lazyray/internal/release"
)

// selfUpdateUserMessage is the helper cmd/selfupdate.go uses to turn an
// ApplySelfUpdate error into an actionable, user-facing message.
func TestSelfUpdateUserMessage(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{"nil", nil, ""},
		{"sig", release.ErrSignatureInvalid, "signature"},
		{"checksum", release.ErrChecksumMismatch, "checksum"},
		{"asset", release.ErrAssetNotFound, "asset"},
		{"other", errors.New("boom"), "boom"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := selfUpdateUserMessage(tt.err)
			if tt.want == "" {
				if got != "" {
					t.Fatalf("got %q, want empty", got)
				}
				return
			}
			if !strings.Contains(strings.ToLower(got), tt.want) {
				t.Errorf("message %q does not mention %q", got, tt.want)
			}
		})
	}
}

func TestSelfUpdateDecision(t *testing.T) {
	cases := []struct {
		latest, current string
		allowDowngrade  bool
		want            selfUpdateAction
	}{
		{"0.9.0", "0.9.0", false, upToDate},
		{"0.9.1", "0.9.0", false, proceed},
		{"0.8.0", "0.9.0", false, refuseDowngrade},
		{"0.8.0", "0.9.0", true, proceed},
	}
	for _, c := range cases {
		if got := selfUpdateDecision(c.latest, c.current, c.allowDowngrade); got != c.want {
			t.Errorf("selfUpdateDecision(%q,%q,%v)=%v want %v", c.latest, c.current, c.allowDowngrade, got, c.want)
		}
	}
}
