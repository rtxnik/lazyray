package lifecycle

import (
	"testing"

	"github.com/rtxnik/lazyray/internal/config"
)

func TestState_ActiveProfileRoundTrips(t *testing.T) {
	withTempData(t)
	if err := WriteState(&State{XrayPID: 123, ActiveProfile: "work", SocksPort: 10808}); err != nil {
		t.Fatalf("WriteState: %v", err)
	}
	st, err := ReadState()
	if err != nil || st == nil {
		t.Fatalf("ReadState: %v", err)
	}
	if st.ActiveProfile != "work" {
		t.Errorf("ActiveProfile = %q, want %q", st.ActiveProfile, "work")
	}
}

func TestProbeContextFor_ActiveWhenStateMatches(t *testing.T) {
	withTempData(t)
	_ = WriteState(&State{XrayPID: 123, ActiveProfile: "work", SocksPort: 10808})
	s := config.DefaultSettings()

	pc := ProbeContextFor("work", s)
	if !pc.Active {
		t.Error("routed profile must be Active")
	}
	if pc.SocksAddr == "" {
		t.Error("SocksAddr must be set")
	}

	if ProbeContextFor("other", s).Active {
		t.Error("non-routed profile must not be Active")
	}
}

func TestProbeContextFor_InactiveWhenNoState(t *testing.T) {
	withTempData(t)
	if ProbeContextFor("work", config.DefaultSettings()).Active {
		t.Error("no state → not Active")
	}
}
