// internal/lifecycle/routing_test.go
package lifecycle

import (
	"fmt"
	"testing"

	"github.com/rtxnik/lazyray/internal/config"
	"github.com/rtxnik/lazyray/internal/platform"
)

// fakeProxy is an in-memory platform.SystemProxy for assertions.
type fakeProxy struct {
	status    platform.ProxyStatus
	calls     []string
	failSOCKS bool
}

func (f *fakeProxy) EnableHTTPProxy(host string, port int) error {
	f.calls = append(f.calls, "enableHTTP")
	f.status.HTTPEnabled, f.status.HTTPHost, f.status.HTTPPort = true, host, port
	return nil
}
func (f *fakeProxy) EnableSOCKSProxy(host string, port int) error {
	f.calls = append(f.calls, "enableSOCKS")
	if f.failSOCKS {
		return fmt.Errorf("gsettings: command not found")
	}
	f.status.SOCKSEnabled, f.status.SOCKSHost, f.status.SOCKSPort = true, host, port
	return nil
}
func (f *fakeProxy) EnablePACProxy(url string) error {
	f.calls = append(f.calls, "enablePAC")
	f.status.PACEnabled, f.status.PACURL = true, url
	return nil
}
func (f *fakeProxy) Disable() error {
	f.calls = append(f.calls, "disable")
	f.status = platform.ProxyStatus{}
	return nil
}
func (f *fakeProxy) Status() (*platform.ProxyStatus, error) {
	s := f.status
	return &s, nil
}

func testSettings() *config.Settings {
	s := config.DefaultSettings()
	s.Local.Listen = "127.0.0.1"
	s.Local.HTTPPort = 10809
	s.Local.SocksPort = 10808
	return s
}

func TestApplyRouting_EnablesHTTPAndSOCKS_SnapshotsPrior(t *testing.T) {
	fp := &fakeProxy{}
	r, err := ApplyRouting(fp, testSettings(), ProxyForceOn)
	if err != nil {
		t.Fatalf("ApplyRouting() = %v", err)
	}
	if !r.SystemProxy {
		t.Error("Routing.SystemProxy = false, want true")
	}
	if r.Prior == nil {
		t.Fatal("Routing.Prior = nil, want snapshot")
	}
	if !contains(fp.calls, "enableHTTP") || !contains(fp.calls, "enableSOCKS") {
		t.Errorf("calls = %v, want enableHTTP + enableSOCKS", fp.calls)
	}
}

func TestApplyRouting_DisabledIsNoop(t *testing.T) {
	fp := &fakeProxy{}
	r, err := ApplyRouting(fp, testSettings(), ProxyForceOff)
	if err != nil {
		t.Fatalf("ApplyRouting() = %v", err)
	}
	if r.SystemProxy {
		t.Error("Routing.SystemProxy = true, want false when disabled")
	}
	if len(fp.calls) != 0 {
		t.Errorf("calls = %v, want none", fp.calls)
	}
}

func TestRevertRouting_CallsDisable(t *testing.T) {
	fp := &fakeProxy{}
	_, _ = ApplyRouting(fp, testSettings(), ProxyForceOn)
	fp.calls = nil
	if err := RevertRouting(fp, Routing{SystemProxy: true, Prior: &ProxySnapshot{}}); err != nil {
		t.Fatalf("RevertRouting() = %v", err)
	}
	if !contains(fp.calls, "disable") {
		t.Errorf("calls = %v, want disable", fp.calls)
	}
}

func TestRevertRouting_RestoresPriorProxy(t *testing.T) {
	fp := &fakeProxy{}
	prior := &ProxySnapshot{HTTPEnabled: true, HTTPHost: "10.0.0.1", HTTPPort: 8080}
	if err := RevertRouting(fp, Routing{SystemProxy: true, Prior: prior}); err != nil {
		t.Fatalf("RevertRouting() = %v", err)
	}
	st, _ := fp.Status()
	if !st.HTTPEnabled || st.HTTPHost != "10.0.0.1" || st.HTTPPort != 8080 {
		t.Errorf("prior not restored: %+v", st)
	}
}

func TestApplyRouting_RevertsHTTPOnSOCKSFailure(t *testing.T) {
	fp := &fakeProxy{failSOCKS: true}
	r, err := ApplyRouting(fp, testSettings(), ProxyForceOn)
	if err == nil {
		t.Fatal("ApplyRouting() = nil error, want SOCKS failure error")
	}
	if r.SystemProxy {
		t.Error("Routing.SystemProxy = true, want false after partial failure")
	}
	if !contains(fp.calls, "enableHTTP") {
		t.Errorf("calls = %v, want enableHTTP to have been called", fp.calls)
	}
	if !contains(fp.calls, "disable") {
		t.Errorf("calls = %v, want disable called to revert partial HTTP enable", fp.calls)
	}
}

func contains(ss []string, want string) bool {
	for _, s := range ss {
		if s == want {
			return true
		}
	}
	return false
}
