package core

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"time"

	"golang.org/x/net/proxy"
)

// userAgent is sent on every direct (non-proxied) request so upstreams that
// reject empty User-Agents (e.g. some GitHub edges) keep working.
const userAgent = "lazyray/1.0 (+https://github.com/rtxnik/lazyray)"

// maxRedirects bounds redirect chains; the same value as the net/http default,
// but enforced explicitly together with per-hop SSRF re-validation.
const maxRedirects = 10

// lookupIP is the DNS resolution seam used by the SSRF guard. It is a package
// variable so tests can inject a deterministic resolver; production uses the
// default system resolver.
var lookupIP = func(host string) ([]net.IP, error) {
	return net.DefaultResolver.LookupIP(context.Background(), "ip", host)
}

// dialContext is the dial seam installed on the directClient transport. It
// defaults to pinnedDialContext (the SSRF dial pin) and is a package variable
// only so end-to-end tests that must reach a local httptest server (loopback,
// which the pin correctly refuses) can substitute a plain dialer. Production
// never reassigns it. Mirrors the lookupIP seam above.
var dialContext = pinnedDialContext

// directTLSConfig is the TLS-client seam for the directClient transport. It is
// nil in production (the transport uses Go's default roots); end-to-end tests
// set it so directClient trusts an httptest server's self-signed certificate.
// Mirrors the dialContext/lookupIP seams.
var directTLSConfig *tls.Config

// cgnatNet is RFC 6598 shared address space (100.64.0.0/10). Go has no stdlib
// predicate for it, so isBlockedIP tests membership explicitly. It is used by
// carrier-grade NAT management planes and overlay networks (e.g. Tailscale), so
// it is treated as a non-public SSRF target.
var cgnatNet = mustCIDR("100.64.0.0/10")

func mustCIDR(s string) *net.IPNet {
	_, n, err := net.ParseCIDR(s)
	if err != nil {
		panic(fmt.Sprintf("invalid CIDR %q: %v", s, err))
	}
	return n
}

// isBlockedIP reports whether ip is a non-routable or otherwise dangerous SSRF
// target (loopback, link-local, RFC1918 private, IPv6 ULA fc00::/7, unspecified,
// or multicast). Public unicast addresses return false.
func isBlockedIP(ip net.IP) bool {
	if ip == nil {
		return true
	}
	if ip.IsLoopback() ||
		ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() ||
		ip.IsPrivate() ||
		ip.IsUnspecified() ||
		ip.IsMulticast() {
		return true
	}
	// RFC 6598 CGNAT and the IPv4 limited-broadcast address have no stdlib
	// predicate. To4() unwraps IPv4-mapped IPv6 so ::ffff:100.64.0.1 is caught.
	if ip4 := ip.To4(); ip4 != nil {
		if cgnatNet.Contains(ip4) || ip4.Equal(net.IPv4bcast) {
			return true
		}
	}
	return false
}

// validateHostIPs resolves host and returns an error if any resolved address is
// a blocked SSRF target (see isBlockedIP), or if resolution yields no usable IP.
func validateHostIPs(host string) error {
	ips, err := lookupIP(host)
	if err != nil {
		return fmt.Errorf("resolving %q: %w", host, err)
	}
	if len(ips) == 0 {
		return fmt.Errorf("no addresses resolved for %q", host)
	}
	for _, ip := range ips {
		if isBlockedIP(ip) {
			return fmt.Errorf("host %q resolves to blocked address %s", host, ip)
		}
	}
	return nil
}

// pinnedDialContext resolves the target host through the lookupIP seam, rejects
// any address that isBlockedIP flags, and dials the validated literal IP
// directly. Because the IP that is checked is the exact IP that is dialed, this
// collapses the check-then-dial pair into a single resolution and closes the
// DNS-rebinding TOCTOU window: the transport can no longer re-resolve the
// hostname to a different address at dial time. It is installed as the
// directClient transport's DialContext, so it fires on every connection the
// client makes, including redirect hops. The proxy (data-plane) client does NOT
// use this — it intentionally tunnels arbitrary hosts and dials only localhost.
func pinnedDialContext(ctx context.Context, network, address string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, fmt.Errorf("splitting dial address %q: %w", address, err)
	}
	var d net.Dialer

	// Literal IP: validate directly, no resolution needed.
	if ip := net.ParseIP(host); ip != nil {
		if isBlockedIP(ip) {
			return nil, fmt.Errorf("refusing to dial blocked address %s", ip)
		}
		return d.DialContext(ctx, network, address)
	}

	ips, err := lookupIP(host)
	if err != nil {
		return nil, fmt.Errorf("resolving %q: %w", host, err)
	}
	if len(ips) == 0 {
		return nil, fmt.Errorf("no addresses resolved for %q", host)
	}
	// Reject if ANY resolved address is blocked (matches validateHostIPs policy).
	for _, ip := range ips {
		if isBlockedIP(ip) {
			return nil, fmt.Errorf("refusing to dial blocked address %s for host %q", ip, host)
		}
	}
	// Dial the validated literal IP(s); TLS SNI/verification still uses the
	// original hostname from the request URL (http.Transport handles that).
	var lastErr error
	for _, ip := range ips {
		conn, derr := d.DialContext(ctx, network, net.JoinHostPort(ip.String(), port))
		if derr == nil {
			return conn, nil
		}
		lastErr = derr
	}
	return nil, lastErr
}

// redirectGuard enforces the redirect cap and re-runs the SSRF host check on
// every hop. It is installed as http.Client.CheckRedirect by directClient and
// can be assigned onto any client that should follow the same policy.
func redirectGuard(req *http.Request, via []*http.Request) error {
	if len(via) >= maxRedirects {
		return fmt.Errorf("stopped after %d redirects", maxRedirects)
	}
	if req.URL.Scheme != "https" {
		return fmt.Errorf("redirect to non-https URL %q refused", req.URL.String())
	}
	if err := validateHostIPs(req.URL.Hostname()); err != nil {
		return fmt.Errorf("redirect blocked: %w", err)
	}
	return nil
}

// proxyDialer builds a SOCKS5 dialer toward socksAddr with a per-dial timeout.
// It is kept distinct from proxyClient because checkDNSLeak/checkLatency dial
// raw TCP without an http.Client.
func proxyDialer(socksAddr string, timeout time.Duration) (proxy.Dialer, error) {
	d, err := proxy.SOCKS5("tcp", socksAddr, nil, &net.Dialer{Timeout: timeout})
	if err != nil {
		return nil, fmt.Errorf("creating SOCKS5 dialer: %w", err)
	}
	return d, nil
}

// proxyClient builds an *http.Client that tunnels through the SOCKS5 proxy at
// socksAddr. timeout is the whole-request budget (the "download-vs-probe" axis
// the caller chooses). MaxIdleConnsPerHost mirrors the speedtest transports.
func proxyClient(socksAddr string, timeout time.Duration) (*http.Client, error) {
	d, err := proxyDialer(socksAddr, timeout)
	if err != nil {
		return nil, err
	}
	transport := &http.Transport{
		Dial:                d.Dial,
		MaxIdleConnsPerHost: 4,
	}
	return &http.Client{
		Transport:     transport,
		Timeout:       timeout,
		CheckRedirect: redirectGuard,
	}, nil
}

// directClient builds an un-proxied *http.Client with a User-Agent and the
// redirect guard. timeout is the whole-request budget.
func directClient(timeout time.Duration) *http.Client {
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		// Wrapped in a closure (not DialContext: dialContext) so the var is read
		// per-dial: tests that reassign the dialContext seam after directClient
		// is constructed are still observed. Do not inline this.
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return dialContext(ctx, network, addr)
		},
		TLSClientConfig:       directTLSConfig,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
	return &http.Client{
		Timeout:       timeout,
		CheckRedirect: redirectGuard,
		Transport:     userAgentTransport{base: transport},
	}
}

// userAgentTransport injects the lazyray User-Agent on outbound requests that do
// not already set one.
type userAgentTransport struct {
	base http.RoundTripper
}

func (t userAgentTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Header.Get("User-Agent") == "" {
		// Clone to avoid mutating the caller's request.
		r2 := req.Clone(req.Context())
		r2.Header.Set("User-Agent", userAgent)
		req = r2
	}
	return t.base.RoundTrip(req)
}

// safeGet performs a context-bound GET through c with the SSRF guard:
// https-only, the URL host's resolved IPs are validated before the request, and
// redirects are re-validated by the client's CheckRedirect. The returned
// response Body is wrapped in an io.LimitReader so callers cannot over-read.
func safeGet(ctx context.Context, c *http.Client, rawURL string, maxBytes int64) (*http.Response, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("parsing URL: %w", err)
	}
	if u.Scheme != "https" {
		return nil, fmt.Errorf("refusing non-https URL %q (scheme %q)", rawURL, u.Scheme)
	}
	if err := validateHostIPs(u.Hostname()); err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	resp.Body = limitedBody{r: io.LimitReader(resp.Body, maxBytes), c: resp.Body}
	return resp, nil
}

// limitedBody is a ReadCloser that reads at most maxBytes but closes the
// underlying body so connections are released.
type limitedBody struct {
	r io.Reader
	c io.Closer
}

func (b limitedBody) Read(p []byte) (int, error) { return b.r.Read(p) }
func (b limitedBody) Close() error               { return b.c.Close() }
