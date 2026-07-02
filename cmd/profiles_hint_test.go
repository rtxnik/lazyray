package cmd

import (
	"errors"
	"strings"
	"testing"

	"github.com/rtxnik/lazyray/internal/clihint"
)

// hintFor extracts the *clihint.Error from an error chain and returns its Hint.
// It fails the test if the error does not carry a *clihint.Error.
func hintFor(t *testing.T, err error) string {
	t.Helper()
	if err == nil {
		t.Fatalf("expected an error, got nil")
	}
	var ce *clihint.Error
	if !errors.As(err, &ce) {
		t.Fatalf("error %v does not wrap *clihint.Error", err)
	}
	return ce.Hint
}

func TestProfilesFamilyHints(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want string
	}{
		{
			name: "export no profiles configured",
			err:  errNoProfilesConfigured(),
			want: "import a profile with 'lzr import <url>'",
		},
		{
			name: "export profile not found",
			err:  errProfileNotFound("home"),
			want: "list profiles with 'lzr config list'",
		},
		{
			name: "export no default profile",
			err:  errNoDefaultProfile(),
			want: "pick one with 'lzr config switch <name>'",
		},
		{
			name: "config switch profile not found",
			err:  errProfileNotFound("staging"),
			want: "list profiles with 'lzr config list'",
		},
		{
			name: "config delete profile not found",
			err:  errProfileNotFound("old"),
			want: "list profiles with 'lzr config list'",
		},
		{
			name: "config duplicate profile not found",
			err:  errProfileNotFound("base"),
			want: "list profiles with 'lzr config list'",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := hintFor(t, tc.err)
			if !strings.Contains(got, tc.want) {
				t.Errorf("hint = %q, want substring %q", got, tc.want)
			}
		})
	}
}
