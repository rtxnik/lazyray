package core

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"

	"github.com/rtxnik/lazyray/internal/config"
)

// StripControl removes terminal-dangerous control characters from an untrusted
// display string. It operates on runes so multi-byte UTF-8 (Cyrillic, emoji) is
// preserved; only C0 (0x00-0x1F, incl. TAB/CR/LF), DEL (0x7F), and C1 (0x80-0x9F)
// code points are dropped. Idempotent.
func StripControl(s string) string {
	if s == "" {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if r < 0x20 || r == 0x7F || (r >= 0x80 && r <= 0x9F) {
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// dnsSchemes are the DoH/DoT URL prefixes lazyray already promotes to advanced
// DNS server objects (see isAdvancedDNS in config.go). An imported DNS resolver
// must be a plain IP or one of these; anything else is rejected.
var dnsSchemes = []string{"https://", "https+local://", "tcp://"}

// ValidateDNSServer enforces an allowlist on an imported DNS resolver string:
// a plain IP literal with an optional port, or a DoH/DoT URL with an allowlisted
// scheme and a non-empty host. Bare hostnames and any other scheme are rejected.
func ValidateDNSServer(server string) error {
	if server == "" {
		return fmt.Errorf("empty DNS server")
	}
	if net.ParseIP(server) != nil {
		return nil
	}
	if host, port, err := net.SplitHostPort(server); err == nil && net.ParseIP(host) != nil {
		if n, perr := strconv.Atoi(port); perr == nil && n >= 1 && n <= 65535 {
			return nil
		}
		return fmt.Errorf("DNS server %q has an invalid port", server)
	}
	for _, scheme := range dnsSchemes {
		if strings.HasPrefix(server, scheme) {
			u, err := url.Parse(server)
			if err != nil {
				return fmt.Errorf("invalid DNS server URL %q: %w", server, err)
			}
			if u.Hostname() == "" {
				return fmt.Errorf("DNS server URL %q has no host", server)
			}
			return nil
		}
	}
	return fmt.Errorf("DNS server %q is not an allowed IP or DoH/DoT URL", server)
}

// HasRoutingOverrides reports whether a profile carries per-profile routing or
// DNS overrides (the trust-changing fields an encrypted import can smuggle in).
func HasRoutingOverrides(p *config.Profile) bool {
	return len(p.Routing.Bypass) > 0 || len(p.Routing.Block) > 0 || len(p.Routing.DNSRules) > 0
}

// SanitizeProfileDisplay strips terminal-dangerous control characters from the
// attacker-controlled display fields of a profile (and its chain nodes), in place.
func SanitizeProfileDisplay(p *config.Profile) {
	p.Name = StripControl(p.Name)
	p.Server.Address = StripControl(p.Server.Address)
	p.Server.Transport.Network = StripControl(p.Server.Transport.Network)
	p.SSH.Host = StripControl(p.SSH.Host)
	p.SSH.User = StripControl(p.SSH.User)
	for i := range p.Chain {
		p.Chain[i].Address = StripControl(p.Chain[i].Address)
		p.Chain[i].Transport.Network = StripControl(p.Chain[i].Transport.Network)
	}
}
