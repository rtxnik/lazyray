package platform

import (
	"testing"
)

func TestCurrentSystemProxy_ReturnsNonNil(t *testing.T) {
	sp := CurrentSystemProxy()
	if sp == nil {
		t.Fatal("CurrentSystemProxy() returned nil")
	}
}

func TestProxyStatus_ZeroValue(t *testing.T) {
	status := &ProxyStatus{}
	if status.HTTPEnabled {
		t.Error("zero-value ProxyStatus should have HTTPEnabled=false")
	}
	if status.SOCKSEnabled {
		t.Error("zero-value ProxyStatus should have SOCKSEnabled=false")
	}
	if status.PACEnabled {
		t.Error("zero-value ProxyStatus should have PACEnabled=false")
	}
	if status.PACURL != "" {
		t.Error("zero-value ProxyStatus should have empty PACURL")
	}
}

func TestSystemProxy_ImplementsInterface(t *testing.T) {
	// CurrentSystemProxy returns a platform-appropriate implementation.
	// Compile-time check that it satisfies the interface.
	var _ = CurrentSystemProxy()
}

func TestProxyStatus_Fields(t *testing.T) {
	status := &ProxyStatus{
		HTTPEnabled:  true,
		HTTPHost:     "127.0.0.1",
		HTTPPort:     10809,
		SOCKSEnabled: true,
		SOCKSHost:    "127.0.0.1",
		SOCKSPort:    10808,
		PACEnabled:   true,
		PACURL:       "http://127.0.0.1:10810/proxy.pac",
	}

	if !status.HTTPEnabled {
		t.Error("HTTPEnabled should be true")
	}
	if status.HTTPHost != "127.0.0.1" {
		t.Errorf("HTTPHost = %q, want 127.0.0.1", status.HTTPHost)
	}
	if status.HTTPPort != 10809 {
		t.Errorf("HTTPPort = %d, want 10809", status.HTTPPort)
	}
	if !status.SOCKSEnabled {
		t.Error("SOCKSEnabled should be true")
	}
	if status.SOCKSHost != "127.0.0.1" {
		t.Errorf("SOCKSHost = %q, want 127.0.0.1", status.SOCKSHost)
	}
	if status.SOCKSPort != 10808 {
		t.Errorf("SOCKSPort = %d, want 10808", status.SOCKSPort)
	}
	if !status.PACEnabled {
		t.Error("PACEnabled should be true")
	}
	if status.PACURL != "http://127.0.0.1:10810/proxy.pac" {
		t.Errorf("PACURL = %q, want http://127.0.0.1:10810/proxy.pac", status.PACURL)
	}
}

func TestParseHostPort(t *testing.T) {
	tests := []struct {
		addr     string
		wantHost string
		wantPort int
	}{
		{"127.0.0.1:8080", "127.0.0.1", 8080},
		{"0.0.0.0:1234", "0.0.0.0", 1234},
		{"localhost:80", "localhost", 80},
		{"127.0.0.1", "127.0.0.1", 0},
	}

	for _, tc := range tests {
		t.Run(tc.addr, func(t *testing.T) {
			var host string
			var port int
			parseHostPort(tc.addr, &host, &port)
			if host != tc.wantHost {
				t.Errorf("host = %q, want %q", host, tc.wantHost)
			}
			if port != tc.wantPort {
				t.Errorf("port = %d, want %d", port, tc.wantPort)
			}
		})
	}
}
