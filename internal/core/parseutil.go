package core

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// anyToInt coerces a JSON value that may arrive as float64, string, or
// json.Number into an int. Non-numeric / nil / bool yield 0. Shared by the
// VMess parser (port + alterID), which receives untyped JSON.
func anyToInt(v any) int {
	switch n := v.(type) {
	case float64:
		return int(n)
	case string:
		i, _ := strconv.Atoi(n)
		return i
	case json.Number:
		i, _ := n.Int64()
		return int(i)
	}
	return 0
}

// defaultRemark is the fallback profile name when a share link carries no
// fragment: "host:port".
func defaultRemark(host string, port int) string {
	return host + ":" + strconv.Itoa(port)
}

// splitHostPort splits "host:port" / "[ipv6]:portspec" / "hostonly" into the
// host and the raw port text (everything after the first ':' past the host).
// portSpec keeps multi-port hysteria2 specs ("443,5000-6000") intact; callers
// parse it. portSpec is "" when no port is present.
func splitHostPort(s string) (host, portSpec string, err error) {
	if strings.HasPrefix(s, "[") {
		closeIdx := strings.Index(s, "]")
		if closeIdx < 0 {
			return "", "", fmt.Errorf("unclosed IPv6 bracket")
		}
		host = s[1:closeIdx]
		if rest := s[closeIdx+1:]; strings.HasPrefix(rest, ":") {
			portSpec = rest[1:]
		}
		return host, portSpec, nil
	}
	if c := strings.LastIndex(s, ":"); c >= 0 {
		return s[:c], s[c+1:], nil
	}
	return s, "", nil
}

// parseUserinfoURL reparses a userinfo-style share link ("scheme://user@host:port?q#frag")
// as an https URL so net/url handles userinfo/host/port/query/fragment. Shared by
// the VLESS and Trojan parsers, which have the identical URL shape. scheme is the
// lowercase URL scheme used for the prefix; display is the human name used in error
// text (e.g. "VLESS"/"Trojan"), preserving each parser's original message casing.
func parseUserinfoURL(scheme, display, raw string) (*url.URL, error) {
	prefix := scheme + "://"
	if !strings.HasPrefix(raw, prefix) {
		return nil, fmt.Errorf("invalid %s URL: must start with %s", display, prefix)
	}
	asHTTPS := "https://" + raw[len(prefix):]
	u, err := url.Parse(asHTTPS)
	if err != nil {
		return nil, fmt.Errorf("parsing %s URL: %w", display, err)
	}
	return u, nil
}

// bracketIPv6 wraps an IPv6 literal in brackets for use in a URL authority
// (RFC 3986). Hosts without ':' and already-bracketed literals pass through.
func bracketIPv6(host string) string {
	if strings.ContainsRune(host, ':') && !strings.HasPrefix(host, "[") {
		return "[" + host + "]"
	}
	return host
}

// decodeBase64Any decodes s under standard and URL base64, padded or raw,
// returning the first success. Covers v2rayN VMess payloads (standard) and
// SIP002 Shadowsocks userinfo (URL, no padding).
func decodeBase64Any(s string) ([]byte, error) {
	for _, enc := range []*base64.Encoding{
		base64.StdEncoding,
		base64.RawStdEncoding,
		base64.RawURLEncoding,
		base64.URLEncoding,
	} {
		if b, err := enc.DecodeString(s); err == nil {
			return b, nil
		}
	}
	return nil, fmt.Errorf("not valid base64")
}
