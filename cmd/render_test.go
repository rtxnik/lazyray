package cmd

import (
	"errors"
	"testing"

	"github.com/rtxnik/lazyray/internal/clihint"
)

// TestRootSilencesCobraOutput asserts the root owns error rendering: cobra's own
// "Error:"/usage prints are silenced so clihint.Render is the single voice.
func TestRootSilencesCobraOutput(t *testing.T) {
	if !rootCmd.SilenceErrors {
		t.Error("rootCmd.SilenceErrors = false, want true (clihint renders errors)")
	}
	if !rootCmd.SilenceUsage {
		t.Error("rootCmd.SilenceUsage = false, want true (no usage dump on error)")
	}
}

// TestExitCodeComposition asserts a hinted error wrapped in ExitError keeps its
// code, and a bare hinted error maps to the generic code.
func TestExitCodeComposition(t *testing.T) {
	coded := &ExitError{Code: ExitConfig, Err: clihint.Errorf("run 'lzr doctor'", "bad config")}
	if got := exitCodeFor(coded); got != ExitConfig {
		t.Errorf("exitCodeFor(coded) = %d, want %d", got, ExitConfig)
	}
	bare := clihint.Errorf("run 'lzr doctor'", "generic failure")
	if got := exitCodeFor(bare); got != ExitGeneric {
		t.Errorf("exitCodeFor(bare) = %d, want %d", got, ExitGeneric)
	}
	var ce *clihint.Error
	if !errors.As(coded, &ce) {
		t.Fatal("ExitError-wrapped clihint.Error not found via errors.As")
	}
}
