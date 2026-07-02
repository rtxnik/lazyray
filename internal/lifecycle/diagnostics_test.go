package lifecycle

import "testing"

func TestIsOurXray_NonPositivePID(t *testing.T) {
	if IsOurXray(0) {
		t.Error("IsOurXray(0) = true, want false")
	}
	if IsOurXray(-7) {
		t.Error("IsOurXray(-7) = true, want false")
	}
}
