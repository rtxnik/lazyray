package cmd

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/rtxnik/lazyray/internal/clihint"
	"github.com/rtxnik/lazyray/internal/doctor"
)

// TestDiagnosticsFamilyHints asserts that each user-actionable error site in the
// diagnostics family (health, doctor) returns an error whose *clihint.Error
// carries the expected imperative hint. health routes to 'lzr doctor'; doctor
// emits 'see the report above' to avoid a self-referential hint.
func TestDiagnosticsFamilyHints(t *testing.T) {
	cases := []struct {
		name     string
		err      error
		wantMsg  string
		wantHint string
	}{
		{
			name:     "health no profiles configured",
			err:      errNoProfilesConfigured(),
			wantMsg:  "no profiles configured",
			wantHint: "import a profile with 'lzr import <url>'",
		},
		{
			name:     "health connectivity failure",
			err:      errHealthChecksFailed(2),
			wantMsg:  "health check failed",
			wantHint: "diagnose with 'lzr doctor'",
		},
		{
			name:     "doctor checks failed",
			err:      errDoctorChecksFailed(3),
			wantMsg:  "check(s) failed",
			wantHint: "see the report above for details",
		},
		{
			name:     "doctor strict warnings",
			err:      errDoctorStrictWarnings(1),
			wantMsg:  "warning(s) with --strict",
			wantHint: "see the report above for details",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.err == nil {
				t.Fatalf("constructor returned nil error")
			}
			var ce *clihint.Error
			if !errors.As(tc.err, &ce) {
				t.Fatalf("error %v does not wrap *clihint.Error", tc.err)
			}
			if !strings.Contains(ce.Msg, tc.wantMsg) {
				t.Errorf("Msg = %q, want substring %q", ce.Msg, tc.wantMsg)
			}
			if !strings.Contains(ce.Hint, tc.wantHint) {
				t.Errorf("Hint = %q, want substring %q", ce.Hint, tc.wantHint)
			}
		})
	}
}

// errDoctorChecksFailed and errDoctorStrictWarnings must remain exit-code aware:
// the *clihint.Error is wrapped in *ExitError so exitCodeFor still resolves the
// process code while Render still finds the hint.
func TestDoctorHintErrorsKeepExitCode(t *testing.T) {
	for _, tc := range []struct {
		name string
		err  error
		want int
	}{
		{"checks failed", errDoctorChecksFailed(1), ExitGeneric},
		{"strict warnings", errDoctorStrictWarnings(1), ExitGeneric},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := exitCodeFor(tc.err); got != tc.want {
				t.Errorf("exitCodeFor = %d, want %d", got, tc.want)
			}
			var ce *clihint.Error
			if !errors.As(tc.err, &ce) {
				t.Errorf("error does not wrap *clihint.Error")
			}
		})
	}
}

// ensure the doctor seam type stays imported/usable from this test file.
var _ = func(ctx context.Context, env *doctor.Env) doctor.Report { return doctor.RunAll(ctx, env) }
