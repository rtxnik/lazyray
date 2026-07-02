package doctor

import (
	"context"
	"testing"
	"time"

	"github.com/rtxnik/lazyray/internal/lifecycle"
	"github.com/rtxnik/lazyray/internal/status"
)

func runningState() *lifecycle.State {
	return &lifecycle.State{
		Owner:         "daemon",
		SupervisorPID: 100,
		XrayPID:       200,
		StartedAt:     time.Now().Add(-time.Minute),
		SocksPort:     10808,
		HTTPPort:      10809,
	}
}

func TestCheckSupervisorConsistency(t *testing.T) {
	tests := []struct {
		name  string
		alive bool
		state *lifecycle.State
		want  Severity
	}{
		{"stopped-clean", false, nil, SeverityInfo},
		{"running-consistent", true, runningState(), SeverityOK},
		{"alive-no-state", true, nil, SeverityWarn},
		{"state-no-supervisor", false, runningState(), SeverityWarn},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			env := &Env{
				SupervisorAlive: func() bool { return tc.alive },
				ReadState:       func() (*lifecycle.State, error) { return tc.state, nil },
			}
			got := checkSupervisor(context.Background(), env)
			if got.Severity != tc.want {
				t.Errorf("severity = %v, want %v (detail=%q)", got.Severity, tc.want, got.Detail)
			}
			if got.Group != "session" {
				t.Errorf("group = %q, want session", got.Group)
			}
		})
	}
}

func TestCheckXrayProcess(t *testing.T) {
	tests := []struct {
		name  string
		state *lifecycle.State
		alive bool
		ours  bool
		want  Severity
	}{
		{"stopped", nil, false, false, SeverityInfo},
		{"alive-and-ours", runningState(), true, true, SeverityOK},
		{"dead", runningState(), false, false, SeverityFail},
		{"alive-not-ours", runningState(), true, false, SeverityWarn},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			env := &Env{
				ReadState:      func() (*lifecycle.State, error) { return tc.state, nil },
				IsProcessAlive: func(pid int) bool { return tc.alive },
				IsOurXray:      func(pid int) bool { return tc.ours },
			}
			got := checkXrayProcess(context.Background(), env)
			if got.Severity != tc.want {
				t.Errorf("severity = %v, want %v (detail=%q)", got.Severity, tc.want, got.Detail)
			}
		})
	}
}

func TestCheckPortsOpen(t *testing.T) {
	tests := []struct {
		name    string
		state   *lifecycle.State
		snap    *status.Snapshot
		snapErr bool
		want    Severity
	}{
		{"stopped", nil, &status.Snapshot{Running: false}, false, SeverityInfo},
		{"both-open", runningState(), &status.Snapshot{Running: true, SocksOK: true, HTTPOK: true}, false, SeverityOK},
		{"socks-down", runningState(), &status.Snapshot{Running: true, SocksOK: false, HTTPOK: true}, false, SeverityWarn},
		{"snapshot-error", runningState(), nil, true, SeverityWarn},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			env := &Env{
				ReadState: func() (*lifecycle.State, error) { return tc.state, nil },
				StatusSnapshot: func() (*status.Snapshot, error) {
					if tc.snapErr {
						return nil, context.DeadlineExceeded
					}
					return tc.snap, nil
				},
			}
			got := checkPortsOpen(context.Background(), env)
			if got.Severity != tc.want {
				t.Errorf("severity = %v, want %v (detail=%q)", got.Severity, tc.want, got.Detail)
			}
		})
	}
}
