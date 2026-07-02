package cmd

import (
	"errors"
	"strings"
	"testing"

	"github.com/rtxnik/lazyray/internal/clihint"
)

// TestLifecycleHints asserts that the user-actionable lifecycle error sites
// carry the canonical "supervisor did not start/restart" hint and stay
// exit-code aware (ExitConfig) by composing ExitError around *clihint.Error.
func TestLifecycleHints(t *testing.T) {
	const wantHint = "check 'lzr status' and logs, then 'lzr doctor'"

	cases := []struct {
		name    string
		err     error
		wantMsg string
	}{
		{
			name:    "start supervisor did not come up",
			err:     supervisorNotUpError("supervisor did not start"),
			wantMsg: "supervisor did not start",
		},
		{
			name:    "restart supervisor did not come up",
			err:     supervisorNotUpError("supervisor did not restart"),
			wantMsg: "supervisor did not restart",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// The site is exit-code aware: an *ExitError wraps the *clihint.Error.
			var ee *ExitError
			if !errors.As(tc.err, &ee) {
				t.Fatalf("error %v is not wrapped in *ExitError", tc.err)
			}
			if ee.Code != ExitConfig {
				t.Errorf("exit code = %d, want %d (ExitConfig)", ee.Code, ExitConfig)
			}

			var ce *clihint.Error
			if !errors.As(tc.err, &ce) {
				t.Fatalf("error %v does not unwrap to *clihint.Error", tc.err)
			}
			if ce.Msg != tc.wantMsg {
				t.Errorf("Msg = %q, want %q", ce.Msg, tc.wantMsg)
			}
			if !strings.Contains(ce.Hint, wantHint) {
				t.Errorf("Hint = %q, want it to contain %q", ce.Hint, wantHint)
			}
		})
	}
}
