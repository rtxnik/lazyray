package doctor

import (
	"context"
	"testing"
	"time"

	"github.com/rtxnik/lazyray/internal/core"
	"github.com/rtxnik/lazyray/internal/lifecycle"
)

func TestConnectivityStopped(t *testing.T) {
	env := &Env{
		ReadState:      func() (*lifecycle.State, error) { return nil, nil },
		RunHealthCheck: func() *core.HealthReport { return nil },
	}
	got := connectivityChecks()
	if len(got) != 1 {
		t.Fatalf("expected exactly 1 connectivity check, got %d", len(got))
	}
	res := got[0](context.Background(), env)
	if res.Group != "connectivity" {
		t.Errorf("group = %q, want connectivity", res.Group)
	}
	if res.Severity != SeverityInfo {
		t.Errorf("stopped severity = %v, want INFO (detail=%q)", res.Severity, res.Detail)
	}
}

func TestConnectivityRunning(t *testing.T) {
	tests := []struct {
		name   string
		report *core.HealthReport
		want   Severity
	}{
		{
			name: "all-pass",
			report: &core.HealthReport{
				AllPassed: true,
				Timestamp: time.Now(),
				Checks: []core.CheckResult{
					{Name: "Exit IP", OK: true, Detail: "203.0.113.7"},
					{Name: "Latency", OK: true, Detail: "42ms", Latency: 42 * time.Millisecond},
				},
			},
			want: SeverityOK,
		},
		{
			name: "a-problem",
			report: &core.HealthReport{
				AllPassed: false,
				Timestamp: time.Now(),
				Checks: []core.CheckResult{
					{Name: "Exit IP", OK: true, Detail: "203.0.113.7"},
					{Name: "DNS Leak", OK: false, Detail: "resolution failed"},
				},
			},
			want: SeverityWarn,
		},
		{
			name:   "nil-report-while-running",
			report: nil,
			want:   SeverityWarn,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			env := &Env{
				ReadState:      func() (*lifecycle.State, error) { return runningState(), nil },
				RunHealthCheck: func() *core.HealthReport { return tc.report },
			}
			res := connectivityChecks()[0](context.Background(), env)
			if res.Severity != tc.want {
				t.Errorf("severity = %v, want %v (detail=%q)", res.Severity, tc.want, res.Detail)
			}
		})
	}
}
