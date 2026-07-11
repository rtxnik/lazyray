package core

import (
	"encoding/hex"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/rtxnik/lazyray/internal/config"
)

// normalizePinSHA256 canonicalizes a hysteria2 pinSHA256 value: lowercase, with
// ':' and '-' separators stripped, per the apernet/hysteria reference client.
func normalizePinSHA256(s string) string {
	if s == "" {
		return ""
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.ToLower(strings.TrimSpace(p))
		p = strings.ReplaceAll(p, ":", "")
		p = strings.ReplaceAll(p, "-", "")
		if p != "" {
			out = append(out, p)
		}
	}
	return strings.Join(out, ",")
}

// ParseHysteria2 parses a hysteria2:// (or hy2://) share link into a Profile.
// Format: hysteria2://auth@host:port/?sni=&insecure=&obfs=&obfs-password=&alpn=&fp=#name
func ParseHysteria2(rawURL string) (*config.Profile, error) {
	raw := strings.TrimSpace(rawURL)
	var rest string
	switch {
	case strings.HasPrefix(raw, "hysteria2://"):
		rest = raw[len("hysteria2://"):]
	case strings.HasPrefix(raw, "hy2://"):
		rest = raw[len("hy2://"):]
	default:
		return nil, fmt.Errorf("invalid Hysteria2 URL: must start with hysteria2:// or hy2://")
	}

	// Split fragment and query manually so port-hopping (host:443,5000-6000)
	// does not break standard URL port parsing.
	fragment := ""
	if i := strings.Index(rest, "#"); i >= 0 {
		fragment = rest[i+1:]
		rest = rest[:i]
	}
	rawQuery := ""
	if i := strings.Index(rest, "?"); i >= 0 {
		rawQuery = rest[i+1:]
		rest = rest[:i]
	}
	rest = strings.TrimSuffix(rest, "/")

	at := strings.LastIndex(rest, "@")
	if at <= 0 {
		return nil, fmt.Errorf("missing auth in Hysteria2 URL")
	}
	auth := rest[:at]
	if dec, err := url.PathUnescape(auth); err == nil {
		auth = dec
	}

	host, port, hop, err := parseHysteriaHostPort(rest[at+1:])
	if err != nil {
		return nil, err
	}

	q, _ := url.ParseQuery(rawQuery)

	sni := q.Get("sni")
	if sni == "" {
		sni = host
	}
	insecure := q.Get("insecure") == "1" || strings.EqualFold(q.Get("insecure"), "true")

	name := fragment
	if dec, err := url.PathUnescape(name); err == nil {
		name = dec
	}
	if name == "" {
		name = defaultRemark(host, port)
	}

	return &config.Profile{
		Name: name,
		Server: config.ServerConfig{
			Address:      host,
			Port:         port,
			UUID:         auth, // Hysteria2 stores the auth string in the UUID field
			Encryption:   "none",
			Protocol:     "hysteria2",
			PortHopping:  hop,
			Obfs:         q.Get("obfs"),
			ObfsPassword: q.Get("obfs-password"),
			Transport:    config.TransportConfig{Network: "hysteria"},
			Security: config.SecurityConfig{
				Type:          "tls",
				SNI:           sni,
				Fingerprint:   q.Get("fp"),
				ALPN:          q.Get("alpn"),
				AllowInsecure: insecure,
				PinSHA256:     normalizePinSHA256(q.Get("pinSHA256")),
			},
		},
	}, nil
}

// ToHysteria2URL converts a Profile back to a hysteria2:// share link.
func ToHysteria2URL(p *config.Profile) string {
	params := url.Values{}
	if p.Server.Security.SNI != "" {
		params.Set("sni", p.Server.Security.SNI)
	}
	// Pin and insecure are mutually exclusive on export.
	if p.Server.Security.PinSHA256 != "" {
		params.Set("pinSHA256", p.Server.Security.PinSHA256)
	} else if p.Server.Security.AllowInsecure {
		params.Set("insecure", "1")
	}
	if p.Server.Obfs != "" {
		params.Set("obfs", p.Server.Obfs)
	}
	if p.Server.ObfsPassword != "" {
		params.Set("obfs-password", p.Server.ObfsPassword)
	}
	if p.Server.Security.ALPN != "" {
		params.Set("alpn", p.Server.Security.ALPN)
	}
	if p.Server.Security.Fingerprint != "" {
		params.Set("fp", p.Server.Security.Fingerprint)
	}

	query := params.Encode()
	if query != "" {
		query = "?" + query
	}
	host := bracketIPv6(p.Server.Address)
	portPart := strconv.Itoa(p.Server.Port)
	if hop := p.Server.PortHopping; hop != "" {
		if base, err := hopBasePort(hop); err == nil && base == p.Server.Port {
			portPart = hop // spec already leads with the base port
		} else {
			// Emit base port + hop set losslessly ("443,5000-6000"); re-parse
			// recovers the base via the same hopBasePort derivation.
			portPart = strconv.Itoa(p.Server.Port) + "," + hop
		}
	}
	return fmt.Sprintf("hysteria2://%s@%s:%s%s#%s",
		p.Server.UUID, host, portPart, query, url.PathEscape(p.Name))
}

// hopBasePort returns the leading single port of a multi-port hopping spec
// ("443,5000-6000" -> 443, "5000-6000" -> 5000). Shared by parse and export
// so an exported authority always re-parses to the same base port.
func hopBasePort(spec string) (int, error) {
	base := spec
	if comma := strings.Index(base, ","); comma >= 0 {
		base = base[:comma]
	}
	if dash := strings.Index(base, "-"); dash >= 0 {
		base = base[:dash]
	}
	return strconv.Atoi(base)
}

// parseHysteriaHostPort splits a hysteria2 authority into host, base port, and
// the raw port-hopping spec (empty if none). The port slot may carry a
// comma/dash multi-port spec (e.g. "443,5000-6000"); the base port is the first
// single port (or the low end of the first range). Default port is 443.
func parseHysteriaHostPort(s string) (host string, port int, hop string, err error) {
	host, portSpec, err := splitHostPort(s)
	if err != nil {
		return "", 0, "", fmt.Errorf("%w in Hysteria2 URL", err)
	}
	if host == "" {
		return "", 0, "", fmt.Errorf("missing host in Hysteria2 URL")
	}
	if portSpec == "" {
		return host, 443, "", nil
	}
	if strings.ContainsAny(portSpec, ",-") {
		hop = portSpec
	}
	port, e := hopBasePort(portSpec)
	if e != nil {
		return "", 0, "", fmt.Errorf("invalid port %q in Hysteria2 URL", portSpec)
	}
	return host, port, hop, nil
}

// Hysteria2HasPortHopping reports whether a hysteria2 URL specifies port-hopping
// ranges, which v1 import does not preserve (only the base port is kept).
func Hysteria2HasPortHopping(rawURL string) bool {
	raw := strings.TrimSpace(rawURL)
	for _, pfx := range []string{"hysteria2://", "hy2://"} {
		if strings.HasPrefix(raw, pfx) {
			rest := raw[len(pfx):]
			if i := strings.IndexAny(rest, "#?"); i >= 0 {
				rest = rest[:i]
			}
			if at := strings.LastIndex(rest, "@"); at >= 0 {
				rest = rest[at+1:]
			}
			return strings.Contains(rest, ",")
		}
	}
	return false
}

// validatePinSHA256 checks a normalized (separator-free) pinSHA256: each
// comma-separated entry must be exactly 64 hex chars (a 32-byte SHA-256).
func validatePinSHA256(s string) error {
	for _, p := range strings.Split(s, ",") {
		p = strings.TrimSpace(p)
		if len(p) != 64 {
			return fmt.Errorf("invalid pinSHA256 %q: expected 64 hex chars, got %d", p, len(p))
		}
		if _, err := hex.DecodeString(p); err != nil {
			return fmt.Errorf("invalid pinSHA256 %q: not hexadecimal", p)
		}
	}
	return nil
}

// validatePortHopping checks a hysteria2 multi-port spec: comma-separated single
// ports and/or "lo-hi" ranges, each in 1..65535.
func validatePortHopping(s string) error {
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if lo, hi, ok := strings.Cut(part, "-"); ok {
			loN, err := checkHopPort(lo)
			if err != nil {
				return err
			}
			hiN, err := checkHopPort(hi)
			if err != nil {
				return err
			}
			if loN > hiN {
				return fmt.Errorf("invalid port-hopping range %q: start %d is greater than end %d", part, loN, hiN)
			}
			continue
		}
		if _, err := checkHopPort(part); err != nil {
			return err
		}
	}
	return nil
}

func checkHopPort(s string) (int, error) {
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil || n < 1 || n > 65535 {
		return 0, fmt.Errorf("invalid port %q in port-hopping spec (must be 1-65535)", s)
	}
	return n, nil
}
