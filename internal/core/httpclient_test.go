package core

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestIsBlockedIP(t *testing.T) {
	tests := []struct {
		name string
		ip   string
		want bool
	}{
		// Blocked: loopback
		{"ipv4 loopback", "127.0.0.1", true},
		{"ipv4 loopback range", "127.5.5.5", true},
		{"ipv6 loopback", "::1", true},
		// Blocked: link-local (incl. cloud metadata 169.254.169.254)
		{"ipv4 link-local metadata", "169.254.169.254", true},
		{"ipv4 link-local", "169.254.0.1", true},
		{"ipv6 link-local", "fe80::1", true},
		// Blocked: RFC1918 private
		{"rfc1918 10", "10.0.0.1", true},
		{"rfc1918 172.16", "172.16.0.1", true},
		{"rfc1918 172.31", "172.31.255.255", true},
		{"rfc1918 192.168", "192.168.1.1", true},
		// Blocked: ULA fc00::/7
		{"ipv6 ula fc", "fc00::1", true},
		{"ipv6 ula fd", "fd12:3456::1", true},
		// Blocked: unspecified + multicast
		{"ipv4 unspecified", "0.0.0.0", true},
		{"ipv6 unspecified", "::", true},
		{"ipv4 multicast", "224.0.0.1", true},
		// Allowed: public
		{"public ipv4 cloudflare", "1.1.1.1", false},
		{"public ipv4 google", "8.8.8.8", false},
		{"public ipv4 generic", "203.0.113.10", false},
		{"public ipv6 google", "2001:4860:4860::8888", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ip := net.ParseIP(tc.ip)
			if ip == nil {
				t.Fatalf("net.ParseIP(%q) returned nil", tc.ip)
			}
			if got := isBlockedIP(ip); got != tc.want {
				t.Errorf("isBlockedIP(%s) = %v, want %v", tc.ip, got, tc.want)
			}
		})
	}
}

func TestIsBlockedIP_CGNATAndBroadcast(t *testing.T) {
	tests := []struct {
		name string
		ip   string
		want bool
	}{
		{"cgnat low 100.64", "100.64.0.1", true},
		{"cgnat high 100.127", "100.127.255.255", true},
		{"just below cgnat is public", "100.63.255.255", false},
		{"just above cgnat is public", "100.128.0.1", false},
		{"limited broadcast", "255.255.255.255", true},
		{"ipv4-mapped cgnat unwraps and blocks", "::ffff:100.64.0.1", true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ip := net.ParseIP(tc.ip)
			if ip == nil {
				t.Fatalf("net.ParseIP(%q) returned nil", tc.ip)
			}
			if got := isBlockedIP(ip); got != tc.want {
				t.Errorf("isBlockedIP(%s) = %v, want %v", tc.ip, got, tc.want)
			}
		})
	}
}

func TestPinnedDialContext_BlocksResolvedPrivateIP(t *testing.T) {
	orig := lookupIP
	t.Cleanup(func() { lookupIP = orig })
	lookupIP = func(host string) ([]net.IP, error) {
		return []net.IP{net.ParseIP("169.254.169.254")}, nil
	}
	_, err := pinnedDialContext(context.Background(), "tcp", "metadata.example:443")
	if err == nil {
		t.Fatal("pinnedDialContext dialed a host resolving to link-local; want block")
	}
	if !strings.Contains(err.Error(), "blocked") {
		t.Errorf("error %q should indicate a blocked dial address", err.Error())
	}
}

func TestPinnedDialContext_BlocksLiteralPrivateIP(t *testing.T) {
	// A literal private IP must be rejected without any resolution.
	orig := lookupIP
	t.Cleanup(func() { lookupIP = orig })
	lookupIP = func(host string) ([]net.IP, error) {
		t.Fatalf("lookupIP must not be called for a literal IP (got %q)", host)
		return nil, nil
	}
	_, err := pinnedDialContext(context.Background(), "tcp", "10.0.0.5:443")
	if err == nil {
		t.Fatal("pinnedDialContext dialed a literal private IP; want block")
	}
	if !strings.Contains(err.Error(), "blocked") {
		t.Errorf("error %q should indicate a blocked dial address", err.Error())
	}
}

func TestSafeGet_PinsDialedIP_RebindBlockedAtDial(t *testing.T) {
	// The headline TOCTOU test: the pre-check resolution returns a PUBLIC IP
	// (passes validateHostIPs), but the dial-time resolution returns a PRIVATE
	// IP (DNS rebinding). The dial-time pin must catch it.
	orig := lookupIP
	t.Cleanup(func() { lookupIP = orig })
	var calls int
	lookupIP = func(host string) ([]net.IP, error) {
		calls++
		if calls == 1 {
			return []net.IP{net.ParseIP("203.0.113.10")}, nil // public on CHECK
		}
		return []net.IP{net.ParseIP("169.254.169.254")}, nil // private on DIAL (rebind)
	}
	c := directClient(2 * time.Second)
	_, err := safeGet(context.Background(), c, "https://rebind.example/", 1<<20)
	if err == nil {
		t.Fatal("safeGet dialed a rebinding host (public-then-private); want block at dial")
	}
	if !strings.Contains(err.Error(), "blocked") {
		t.Errorf("error %q should indicate a blocked dial address", err.Error())
	}
	if calls < 2 {
		t.Errorf("expected a second (dial-time) resolution; got %d lookups", calls)
	}
}

func TestValidateHostIPs(t *testing.T) {
	// Save and restore the resolver seam.
	orig := lookupIP
	t.Cleanup(func() { lookupIP = orig })

	tests := []struct {
		name      string
		host      string
		resolved  []net.IP
		resolvErr error
		wantErr   bool
	}{
		{
			name:     "public host allowed",
			host:     "example.com",
			resolved: []net.IP{net.ParseIP("203.0.113.10")},
			wantErr:  false,
		},
		{
			name:     "loopback host blocked",
			host:     "localhost",
			resolved: []net.IP{net.ParseIP("127.0.0.1")},
			wantErr:  true,
		},
		{
			name:     "metadata host blocked",
			host:     "metadata.internal",
			resolved: []net.IP{net.ParseIP("169.254.169.254")},
			wantErr:  true,
		},
		{
			name:     "any blocked IP among results blocks all",
			host:     "rebind.example",
			resolved: []net.IP{net.ParseIP("203.0.113.10"), net.ParseIP("10.0.0.1")},
			wantErr:  true,
		},
		{
			name:      "resolution failure is an error",
			host:      "nx.example",
			resolvErr: &net.DNSError{Err: "no such host", Name: "nx.example"},
			wantErr:   true,
		},
		{
			name:     "empty resolution is an error",
			host:     "void.example",
			resolved: nil,
			wantErr:  true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			lookupIP = func(host string) ([]net.IP, error) {
				return tc.resolved, tc.resolvErr
			}
			err := validateHostIPs(tc.host)
			if (err != nil) != tc.wantErr {
				t.Errorf("validateHostIPs(%q) error = %v, wantErr %v", tc.host, err, tc.wantErr)
			}
		})
	}
}

func TestDirectClient_SetsUserAgentAndRedirectCap(t *testing.T) {
	c := directClient(5 * time.Second)
	if c == nil {
		t.Fatal("directClient returned nil")
	}
	if c.Timeout != 5*time.Second {
		t.Errorf("directClient timeout = %v, want 5s", c.Timeout)
	}
	if c.CheckRedirect == nil {
		t.Error("directClient CheckRedirect is nil, want a redirect cap")
	}
	// 11th redirect must be refused (cap at 10).
	via := make([]*http.Request, 10)
	req, _ := http.NewRequest("GET", "https://example.com", nil)
	if err := c.CheckRedirect(req, via); err == nil {
		t.Error("CheckRedirect allowed an 11th redirect, want error")
	}
}

func TestProxyClient_BadAddr(t *testing.T) {
	// proxy.SOCKS5 with a syntactically valid addr still constructs (it dials
	// lazily), so construction should succeed; we only assert the shape.
	c, err := proxyClient("127.0.0.1:1080", 7*time.Second)
	if err != nil {
		t.Fatalf("proxyClient error = %v", err)
	}
	if c.Timeout != 7*time.Second {
		t.Errorf("proxyClient timeout = %v, want 7s", c.Timeout)
	}
	tr, ok := c.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("proxyClient transport type = %T, want *http.Transport", c.Transport)
	}
	if tr.MaxIdleConnsPerHost != 4 {
		t.Errorf("proxyClient MaxIdleConnsPerHost = %d, want 4", tr.MaxIdleConnsPerHost)
	}
}

func TestProxyDialer_Constructs(t *testing.T) {
	d, err := proxyDialer("127.0.0.1:1080", 5*time.Second)
	if err != nil {
		t.Fatalf("proxyDialer error = %v", err)
	}
	if d == nil {
		t.Fatal("proxyDialer returned nil dialer")
	}
}

func TestSafeGet_RejectsNonHTTPS(t *testing.T) {
	orig := lookupIP
	t.Cleanup(func() { lookupIP = orig })
	lookupIP = func(host string) ([]net.IP, error) {
		return []net.IP{net.ParseIP("203.0.113.10")}, nil
	}
	c := directClient(2 * time.Second)
	_, err := safeGet(context.Background(), c, "http://example.com/", 1<<20)
	if err == nil {
		t.Fatal("safeGet allowed http:// URL, want https-only rejection")
	}
	if !strings.Contains(err.Error(), "https") {
		t.Errorf("error %q should mention https", err.Error())
	}
}

func TestSafeGet_BlocksPrivateHost(t *testing.T) {
	orig := lookupIP
	t.Cleanup(func() { lookupIP = orig })
	lookupIP = func(host string) ([]net.IP, error) {
		return []net.IP{net.ParseIP("169.254.169.254")}, nil
	}
	c := directClient(2 * time.Second)
	_, err := safeGet(context.Background(), c, "https://metadata.example/", 1<<20)
	if err == nil {
		t.Fatal("safeGet allowed a host resolving to link-local, want block")
	}
}

func TestSafeGet_CapsBody(t *testing.T) {
	// Public httptest server; force its host past the resolver as public.
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(make([]byte, 4096)) // 4 KB body
	}))
	defer srv.Close()

	orig := lookupIP
	t.Cleanup(func() { lookupIP = orig })
	lookupIP = func(host string) ([]net.IP, error) {
		return []net.IP{net.ParseIP("203.0.113.10")}, nil // pretend public
	}

	c := srv.Client() // trusts the httptest TLS cert
	c.Timeout = 2 * time.Second
	c.CheckRedirect = redirectGuard

	resp, err := safeGet(context.Background(), c, srv.URL, 1024) // cap at 1 KB
	if err != nil {
		t.Fatalf("safeGet error = %v", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("reading body: %v", err)
	}
	if len(body) != 1024 {
		t.Errorf("body length = %d, want 1024 (capped)", len(body))
	}
}

func TestSafeGet_RedirectToPrivateBlocked(t *testing.T) {
	// Target server redirects to a private host; safeGet must block on the hop.
	target := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "https://internal.example/secret", http.StatusFound)
	}))
	defer target.Close()

	orig := lookupIP
	t.Cleanup(func() { lookupIP = orig })
	lookupIP = func(host string) ([]net.IP, error) {
		if strings.HasPrefix(host, "internal") {
			return []net.IP{net.ParseIP("10.0.0.5")}, nil // private on redirect
		}
		return []net.IP{net.ParseIP("203.0.113.10")}, nil // public first hop
	}

	c := target.Client()
	c.Timeout = 2 * time.Second
	c.CheckRedirect = redirectGuard

	_, err := safeGet(context.Background(), c, target.URL, 1<<20)
	if err == nil {
		t.Fatal("safeGet followed a redirect to a private host, want block")
	}
}
