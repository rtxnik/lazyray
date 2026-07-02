package commands

import "testing"

func TestToggleMetricRegisteredWithStatusScope(t *testing.T) {
	reg := New(DefaultKeyMap())
	for _, c := range reg.All() {
		if c.ID == "ToggleMetric" {
			if c.Scope != ScopeStatus {
				t.Fatalf("ToggleMetric scope = %v, want ScopeStatus", c.Scope)
			}
			if KeyDisplay(c.Binding) != "m" {
				t.Fatalf("ToggleMetric key = %q, want \"m\"", KeyDisplay(c.Binding))
			}
			return
		}
	}
	t.Fatal("ToggleMetric not found in registry")
}

func TestToggleMetricIsLaunchable(t *testing.T) {
	reg := New(DefaultKeyMap())
	for _, c := range reg.Launchable() {
		if c.ID == "ToggleMetric" {
			return
		}
	}
	t.Fatal("ToggleMetric should be launchable from the palette")
}
