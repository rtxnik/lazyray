package modals

import (
	"strings"
	"testing"
)

func TestNextNudge_URLMethod(t *testing.T) {
	n := nextNudge(nudgeInput{method: methodURL, startKey: "s"})
	if !strings.Contains(n.Text, "[s]") {
		t.Errorf("URL nudge must cite the start key, got %q", n.Text)
	}
	if !strings.Contains(n.Text, "start the proxy") {
		t.Errorf("URL nudge must mention starting the proxy, got %q", n.Text)
	}
}

func TestNextNudge_SubscriptionMethod(t *testing.T) {
	n := nextNudge(nudgeInput{method: methodSubscription, startKey: "s"})
	if !strings.Contains(n.Text, "importing") {
		t.Errorf("subscription nudge must mention importing, got %q", n.Text)
	}
	if !strings.Contains(n.Text, "[s]") {
		t.Errorf("subscription nudge must cite the start key, got %q", n.Text)
	}
}

func TestNextNudge_UsesInjectedKey(t *testing.T) {
	n := nextNudge(nudgeInput{method: methodURL, startKey: "x"})
	if !strings.Contains(n.Text, "[x]") {
		t.Errorf("nudge must use the injected key, got %q", n.Text)
	}
	if strings.Contains(n.Text, "[s]") {
		t.Errorf("nudge must not hardcode [s] when key is x, got %q", n.Text)
	}
}
