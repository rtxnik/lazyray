package doctor

import (
	"context"
	"testing"

	"github.com/rtxnik/lazyray/internal/lifecycle"
	"github.com/rtxnik/lazyray/internal/platform"
)

func stateWithRouting(sysProxy bool) *lifecycle.State {
	st := runningState()
	st.Routing = lifecycle.Routing{SystemProxy: sysProxy}
	return st
}

func TestCheckProxyDesync(t *testing.T) {
	tests := []struct {
		name  string
		state *lifecycle.State
		os    *platform.ProxyStatus
		osErr bool
		want  Severity
	}{
		{"stopped", nil, &platform.ProxyStatus{}, false, SeverityInfo},
		{"both-on-consistent", stateWithRouting(true), &platform.ProxyStatus{HTTPEnabled: true}, false, SeverityOK},
		{"both-off-consistent", stateWithRouting(false), &platform.ProxyStatus{}, false, SeverityOK},
		{"state-on-os-off", stateWithRouting(true), &platform.ProxyStatus{}, false, SeverityWarn},
		{"state-off-os-on", stateWithRouting(false), &platform.ProxyStatus{HTTPEnabled: true}, false, SeverityWarn},
		{"os-read-error", stateWithRouting(true), nil, true, SeverityWarn},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			env := &Env{
				ReadState: func() (*lifecycle.State, error) { return tc.state, nil },
				ProxyStatus: func() (*platform.ProxyStatus, error) {
					if tc.osErr {
						return nil, context.DeadlineExceeded
					}
					return tc.os, nil
				},
			}
			got := checkProxyDesync(context.Background(), env)
			if got.Severity != tc.want {
				t.Errorf("severity = %v, want %v (detail=%q)", got.Severity, tc.want, got.Detail)
			}
			if got.Group != "routing" {
				t.Errorf("group = %q, want routing", got.Group)
			}
		})
	}
}

func TestCheckHeadless(t *testing.T) {
	tests := []struct {
		name    string
		state   *lifecycle.State
		goos    string
		desktop string
		want    Severity
	}{
		{"stopped", nil, "linux", "", SeverityInfo},
		{"linux-desktop-present", stateWithRouting(true), "linux", "gnome", SeverityOK},
		{"linux-headless-with-proxy-intent", stateWithRouting(true), "linux", "", SeverityWarn},
		{"linux-headless-no-proxy", stateWithRouting(false), "linux", "", SeverityInfo},
		{"darwin-running-with-proxy", stateWithRouting(true), "darwin", "", SeverityOK},
		{"windows-running-with-proxy", stateWithRouting(true), "windows", "", SeverityOK},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			env := &Env{
				GOOS:       tc.goos,
				ReadState:  func() (*lifecycle.State, error) { return tc.state, nil },
				DesktopEnv: func() string { return tc.desktop },
			}
			got := checkHeadless(context.Background(), env)
			if got.Severity != tc.want {
				t.Errorf("severity = %v, want %v (detail=%q)", got.Severity, tc.want, got.Detail)
			}
		})
	}
}

func TestCheckForeignXray(t *testing.T) {
	tests := []struct {
		name    string
		state   *lifecycle.State
		scanPID int
		ours    bool
		want    Severity
	}{
		{"none-running", nil, 0, false, SeverityOK},
		{"only-ours", runningState(), 200, true, SeverityOK},
		{"foreign-present", runningState(), 999, false, SeverityWarn},
		{"foreign-no-state", nil, 999, false, SeverityWarn},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			env := &Env{
				ReadState:   func() (*lifecycle.State, error) { return tc.state, nil },
				ScanXrayPID: func() int { return tc.scanPID },
				IsOurXray:   func(pid int) bool { return tc.ours },
			}
			got := checkForeignXray(context.Background(), env)
			if got.Severity != tc.want {
				t.Errorf("severity = %v, want %v (detail=%q)", got.Severity, tc.want, got.Detail)
			}
			if got.Group != "foreign" {
				t.Errorf("group = %q, want foreign", got.Group)
			}
		})
	}
}

func TestCheckStartupError(t *testing.T) {
	tests := []struct {
		name string
		se   *lifecycle.StartupError
		want Severity
	}{
		{"none", nil, SeverityOK},
		{"present", &lifecycle.StartupError{Stage: "routing", Message: "no desktop env"}, SeverityFail},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			env := &Env{
				ReadStartupError: func() (*lifecycle.StartupError, error) { return tc.se, nil },
			}
			got := checkStartupError(context.Background(), env)
			if got.Severity != tc.want {
				t.Errorf("severity = %v, want %v (detail=%q)", got.Severity, tc.want, got.Detail)
			}
			if got.Group != "startup" {
				t.Errorf("group = %q, want startup", got.Group)
			}
		})
	}
}
