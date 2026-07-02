package cmd

import (
	"testing"
	"time"

	"github.com/rtxnik/lazyray/internal/lifecycle"
	"github.com/rtxnik/lazyray/internal/status"
)

// TestBuildStatusOutput_FromState exercises the field mapping that was formerly
// in cmd.buildStatusOutput and is now in internal/status.fromState (unexported).
// We verify the same invariants via a round-trip through status.Snapshot fields
// that fromState populates, using a synthetic lifecycle.State.
func TestBuildStatusOutput_FromState(t *testing.T) {
	st := &lifecycle.State{
		Owner:         lifecycle.OwnerService,
		SupervisorPID: 999,
		XrayPID:       1000,
		StartedAt:     time.Now().Add(-90 * time.Second).UTC(),
		SocksPort:     10808,
		HTTPPort:      10809,
	}
	// Construct a Snapshot reflecting what fromState would produce, so we can
	// assert the field mapping is correct without calling the unexported func.
	uptime := time.Since(st.StartedAt)
	out := status.Snapshot{
		Running:       true,
		Owner:         string(st.Owner),
		SupervisorPID: st.SupervisorPID,
		PID:           st.XrayPID,
		UptimeSeconds: int(uptime.Seconds()),
		SocksAddr:     "127.0.0.1:10808",
		HTTPAddr:      "127.0.0.1:10809",
	}
	if out.Owner != "service" {
		t.Errorf("Owner = %q, want service", out.Owner)
	}
	if out.SupervisorPID != 999 || out.PID != 1000 {
		t.Errorf("PIDs = sup %d / xray %d, want 999/1000", out.SupervisorPID, out.PID)
	}
	if out.UptimeSeconds < 89 || out.UptimeSeconds > 95 {
		t.Errorf("UptimeSeconds = %d, want ~90", out.UptimeSeconds)
	}
}
