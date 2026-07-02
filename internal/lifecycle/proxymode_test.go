// internal/lifecycle/proxymode_test.go
package lifecycle

import (
	"errors"
	"fmt"
	"testing"

	"github.com/rtxnik/lazyray/internal/platform"
)

// headlessProxy fails HTTP-enable with an ErrNoDesktopEnv-wrapped error, like a
// real headless linux box. genericFail toggles a non-headless failure instead.
type headlessProxy struct {
	calls       []string
	genericFail bool
}

func (h *headlessProxy) EnableHTTPProxy(host string, port int) error {
	h.calls = append(h.calls, "enableHTTP")
	if h.genericFail {
		return fmt.Errorf("gsettings exploded")
	}
	return fmt.Errorf("no supported desktop environment detected: %w", platform.ErrNoDesktopEnv)
}
func (h *headlessProxy) EnableSOCKSProxy(host string, port int) error {
	h.calls = append(h.calls, "enableSOCKS")
	return nil
}
func (h *headlessProxy) EnablePACProxy(url string) error {
	h.calls = append(h.calls, "enablePAC")
	return nil
}
func (h *headlessProxy) Disable() error {
	h.calls = append(h.calls, "disable")
	return nil
}
func (h *headlessProxy) Status() (*platform.ProxyStatus, error) {
	return &platform.ProxyStatus{}, nil
}

func TestApplyRouting_DefaultHeadless_DegradesNonFatal(t *testing.T) {
	hp := &headlessProxy{}
	r, err := ApplyRouting(hp, testSettings(), ProxyDefault)
	if err != nil {
		t.Fatalf("ApplyRouting(ProxyDefault, headless) = %v, want nil (degrade)", err)
	}
	if r.SystemProxy {
		t.Error("Routing.SystemProxy = true, want false after headless degrade")
	}
	if !contains(hp.calls, "enableHTTP") {
		t.Errorf("calls = %v, want enableHTTP attempted", hp.calls)
	}
}

func TestApplyRouting_ForceOnHeadless_Fatal(t *testing.T) {
	hp := &headlessProxy{}
	_, err := ApplyRouting(hp, testSettings(), ProxyForceOn)
	if err == nil {
		t.Fatal("ApplyRouting(ProxyForceOn, headless) = nil, want fatal error")
	}
	if !errors.Is(err, platform.ErrNoDesktopEnv) {
		t.Errorf("error = %v, want wrapping ErrNoDesktopEnv", err)
	}
}

func TestApplyRouting_ForceOff_NoRouting(t *testing.T) {
	hp := &headlessProxy{}
	r, err := ApplyRouting(hp, testSettings(), ProxyForceOff)
	if err != nil {
		t.Fatalf("ApplyRouting(ProxyForceOff) = %v", err)
	}
	if r.SystemProxy {
		t.Error("Routing.SystemProxy = true, want false for ProxyForceOff")
	}
	if len(hp.calls) != 0 {
		t.Errorf("calls = %v, want none for ProxyForceOff", hp.calls)
	}
}

func TestApplyRouting_DefaultGenericFailure_Fatal(t *testing.T) {
	// A non-headless failure under ProxyDefault must still be fatal.
	hp := &headlessProxy{genericFail: true}
	_, err := ApplyRouting(hp, testSettings(), ProxyDefault)
	if err == nil {
		t.Fatal("ApplyRouting(ProxyDefault, generic failure) = nil, want fatal")
	}
	if errors.Is(err, platform.ErrNoDesktopEnv) {
		t.Errorf("error = %v, want a non-headless error", err)
	}
}

func TestApplyRouting_DefaultSuccess_AppliesProxy(t *testing.T) {
	fp := &fakeProxy{}
	r, err := ApplyRouting(fp, testSettings(), ProxyDefault)
	if err != nil {
		t.Fatalf("ApplyRouting(ProxyDefault, desktop) = %v", err)
	}
	if !r.SystemProxy {
		t.Error("Routing.SystemProxy = false, want true on a healthy desktop")
	}
}
