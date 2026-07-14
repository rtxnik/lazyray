package cmd

import (
	"testing"

	"github.com/rtxnik/lazyray/internal/app"
	"github.com/rtxnik/lazyray/internal/config"
	"github.com/rtxnik/lazyray/internal/core"
)

func TestUpdateApply_RoutesThroughService_AndRefusesWhenSupervised(t *testing.T) {
	orig := applyXrayUpdate
	t.Cleanup(func() { applyXrayUpdate = orig })

	called := false
	applyXrayUpdate = func(_ *core.XrayProcess, _ *core.ReleaseInfo, _ string,
		_ *config.Settings, _, _ bool) error {
		called = true
		return app.ErrSupervisorRunning
	}

	err := runUpdateApply(&core.ReleaseInfo{TagName: "v99.9.9"}, "http://example/x.zip",
		config.DefaultSettings(), false, false)
	if !called {
		t.Error("apply did not route through applyXrayUpdate")
	}
	if err == nil {
		t.Error("supervised refusal must surface as an error")
	}
}

func TestXrayUpdateDecision(t *testing.T) {
	cases := []struct {
		target, installed string
		allowDowngrade    bool
		want              xrayUpdateGate
	}{
		{"v26.3.27", "v26.3.26", false, gateOK},
		{"v26.3.27", "v26.3.27", false, gateUpToDate}, // idempotent: equal is not a downgrade
		{"v26.3.26", "v26.3.27", false, gateDowngrade},
		{"v26.3.26", "v26.3.27", true, gateOK},
		{"v1.0.0", "not installed", false, gateBelowFloor},
		{"v26.3.27", "not installed", false, gateOK},
	}
	for _, c := range cases {
		if got := xrayUpdateDecision(c.target, c.installed, c.allowDowngrade); got != c.want {
			t.Errorf("xrayUpdateDecision(%q,%q,%v)=%v want %v", c.target, c.installed, c.allowDowngrade, got, c.want)
		}
	}
}
