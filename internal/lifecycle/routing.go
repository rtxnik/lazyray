// internal/lifecycle/routing.go
package lifecycle

import (
	"errors"
	"fmt"
	"os"

	"github.com/rtxnik/lazyray/internal/config"
	"github.com/rtxnik/lazyray/internal/platform"
)

// ProxyMode is the tri-state intent for OS system-proxy configuration.
type ProxyMode int

const (
	// ProxyDefault honors Settings.AutoSystemProxy and degrades non-fatally when
	// the host is headless (no desktop env to configure).
	ProxyDefault ProxyMode = iota
	// ProxyForceOn (--proxy) requires the OS proxy; headless is fatal.
	ProxyForceOn
	// ProxyForceOff (--no-proxy) never configures the OS proxy.
	ProxyForceOff
)

// proxyEnabled resolves the tri-state mode plus AutoSystemProxy into a bool.
func proxyEnabled(mode ProxyMode, autoSystemProxy bool) bool {
	switch mode {
	case ProxyForceOn:
		return true
	case ProxyForceOff:
		return false
	default: // ProxyDefault
		return autoSystemProxy
	}
}

// ApplyRouting snapshots the current system-proxy state then, if the resolved
// mode enables proxying, enables HTTP+SOCKS proxying through lazyray's local
// ports. Under ProxyDefault on a headless host (ErrNoDesktopEnv) it degrades
// non-fatally: no OS proxy is set, routing.SystemProxy stays false, and the
// caller continues with xray running and env-var advice logged. The returned
// Routing must be persisted in State so teardown/self-heal can revert it.
func ApplyRouting(sp platform.SystemProxy, settings *config.Settings, mode ProxyMode) (Routing, error) {
	r := Routing{}
	if !proxyEnabled(mode, settings.AutoSystemProxy) {
		return r, nil
	}
	if st, err := sp.Status(); err == nil && st != nil {
		r.Prior = &ProxySnapshot{
			HTTPEnabled: st.HTTPEnabled, HTTPHost: st.HTTPHost, HTTPPort: st.HTTPPort,
			SOCKSEnabled: st.SOCKSEnabled, SOCKSHost: st.SOCKSHost, SOCKSPort: st.SOCKSPort,
			PACEnabled: st.PACEnabled, PACURL: st.PACURL,
		}
	}
	if err := sp.EnableHTTPProxy(settings.Local.Listen, settings.Local.HTTPPort); err != nil {
		if errors.Is(err, platform.ErrNoDesktopEnv) && mode == ProxyDefault {
			fmt.Fprintf(os.Stderr,
				"lazyray: no desktop environment detected; skipping OS proxy. Set:\n  export http_proxy=http://%s:%d\n  export https_proxy=http://%s:%d\n  export all_proxy=socks5://%s:%d\n",
				settings.Local.Listen, settings.Local.HTTPPort,
				settings.Local.Listen, settings.Local.HTTPPort,
				settings.Local.Listen, settings.Local.SocksPort)
			r.SystemProxy = false
			return r, nil
		}
		return r, err
	}
	if err := sp.EnableSOCKSProxy(settings.Local.Listen, settings.Local.SocksPort); err != nil {
		if errors.Is(err, platform.ErrNoDesktopEnv) && mode == ProxyDefault {
			_ = sp.Disable() // revert the HTTP proxy we just set
			fmt.Fprintf(os.Stderr,
				"lazyray: no desktop environment detected; skipping OS proxy. Set:\n  export all_proxy=socks5://%s:%d\n",
				settings.Local.Listen, settings.Local.SocksPort)
			r.SystemProxy = false
			return r, nil
		}
		_ = sp.Disable() // revert the HTTP proxy we just set; don't leak a half-applied state
		return r, err
	}
	r.SystemProxy = true
	return r, nil
}

// RevertRouting removes lazyray's proxy settings and, best-effort, restores any
// proxy that was configured before lazyray took over.
func RevertRouting(sp platform.SystemProxy, r Routing) error {
	if !r.SystemProxy && !r.PAC {
		return nil
	}
	if err := sp.Disable(); err != nil {
		return err
	}
	if r.Prior != nil {
		if r.Prior.HTTPEnabled {
			_ = sp.EnableHTTPProxy(r.Prior.HTTPHost, r.Prior.HTTPPort)
		}
		if r.Prior.SOCKSEnabled {
			_ = sp.EnableSOCKSProxy(r.Prior.SOCKSHost, r.Prior.SOCKSPort)
		}
		if r.Prior.PACEnabled {
			_ = sp.EnablePACProxy(r.Prior.PACURL)
		}
	}
	return nil
}
