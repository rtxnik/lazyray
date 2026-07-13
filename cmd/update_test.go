package cmd

import "testing"

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
