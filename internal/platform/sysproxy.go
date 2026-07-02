package platform

import (
	"errors"
	"strconv"
	"strings"
)

// ErrNoDesktopEnv signals that no supported desktop environment was detected, so
// OS-level proxy configuration is unavailable. Callers may treat it as a
// non-fatal "degrade to env-var advice" condition rather than a hard failure.
var ErrNoDesktopEnv = errors.New("no supported desktop environment")

// SystemProxy defines cross-platform system proxy configuration operations.
type SystemProxy interface {
	// EnableHTTPProxy sets the system HTTP/HTTPS proxy.
	EnableHTTPProxy(host string, port int) error
	// EnableSOCKSProxy sets the system SOCKS proxy.
	EnableSOCKSProxy(host string, port int) error
	// EnablePACProxy sets the system proxy auto-configuration URL.
	EnablePACProxy(pacURL string) error
	// Disable removes all proxy settings applied by lazyray.
	Disable() error
	// Status returns the current system proxy state.
	Status() (*ProxyStatus, error)
}

// ProxyStatus represents the current state of system proxy settings.
type ProxyStatus struct {
	HTTPEnabled  bool
	HTTPHost     string
	HTTPPort     int
	SOCKSEnabled bool
	SOCKSHost    string
	SOCKSPort    int
	PACEnabled   bool
	PACURL       string
}

// CurrentSystemProxy returns the platform-specific system proxy implementation.
func CurrentSystemProxy() SystemProxy {
	return currentSystemProxy()
}

// parseHostPort splits "host:port" into separate components.
func parseHostPort(addr string, host *string, port *int) {
	idx := strings.LastIndex(addr, ":")
	if idx > 0 {
		*host = addr[:idx]
		*port, _ = strconv.Atoi(addr[idx+1:])
	} else {
		*host = addr
	}
}
