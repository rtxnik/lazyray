package cmd

import (
	"errors"
	"strings"
	"testing"

	"github.com/rtxnik/lazyray/internal/clihint"
)

// TestUpdatesFamilyHints asserts that the user-actionable error sites in the
// updates family (service install/locate) carry the canonical clihint Hint.
// The xray-core "missing/old" hint for update apply is exercised indirectly:
// update apply surfaces a *clihint.Error when the pinned xray-core is missing
// or too old; we assert that constructor here via a small helper site.
func TestUpdatesFamilyHints(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		err      error
		wantHint string
	}{
		{
			name:     "service locate binary",
			err:      serviceLocateError(errors.New("exec: not found")),
			wantHint: "services are user-scoped — see 'lzr doctor' and TROUBLESHOOTING.md",
		},
		{
			name:     "service install",
			err:      serviceInstallError(errors.New("permission denied")),
			wantHint: "services are user-scoped — see 'lzr doctor' and TROUBLESHOOTING.md",
		},
		{
			name:     "xray-core missing or too old",
			err:      xrayMissingError(errors.New("xray-core not found")),
			wantHint: "fetch xray-core with 'lzr update apply'",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var ce *clihint.Error
			if !errors.As(tc.err, &ce) {
				t.Fatalf("error %v does not unwrap to *clihint.Error", tc.err)
			}
			if !strings.Contains(ce.Hint, tc.wantHint) {
				t.Errorf("hint = %q, want substring %q", ce.Hint, tc.wantHint)
			}
		})
	}
}
