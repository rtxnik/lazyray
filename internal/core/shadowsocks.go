package core

import (
	"encoding/base64"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/rtxnik/lazyray/internal/config"
)

// Supported Shadowsocks encryption methods.
var supportedSSMethods = map[string]bool{
	"aes-256-gcm":             true,
	"chacha20-ietf-poly1305":  true,
	"2022-blake3-aes-256-gcm": true,
}

// ParseShadowsocks parses a Shadowsocks URL into a Profile.
// Supports SIP002 format: ss://base64(method:password)@host:port#name
// Also supports plaintext userinfo: ss://method:password@host:port#name
func ParseShadowsocks(rawURL string) (*config.Profile, error) {
	if !strings.HasPrefix(rawURL, "ss://") {
		return nil, fmt.Errorf("invalid Shadowsocks URL: must start with ss://")
	}

	// Split off the fragment (profile name) before parsing
	body := rawURL[len("ss://"):]
	fragment := ""
	if idx := strings.LastIndex(body, "#"); idx >= 0 {
		fragment = body[idx+1:]
		body = body[:idx]
	}

	var method, password, host string
	var port int

	// Try SIP002 format: base64(method:password)@host:port
	if atIdx := strings.LastIndex(body, "@"); atIdx >= 0 {
		userinfo := body[:atIdx]
		hostPort := body[atIdx+1:]

		// Try base64 decode the userinfo
		decodedBytes, err := decodeBase64Any(userinfo)
		dec := string(decodedBytes)
		if err != nil {
			// Not base64, try plaintext method:password
			dec = userinfo
		}

		parts := strings.SplitN(dec, ":", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid Shadowsocks userinfo: expected method:password")
		}
		method = parts[0]
		password = parts[1]

		// Parse host:port
		host, port, err = parseHostPort(hostPort)
		if err != nil {
			return nil, fmt.Errorf("invalid Shadowsocks host:port: %w", err)
		}
	} else {
		// Legacy format: base64(method:password@host:port)
		decodedBytes, err := decodeBase64Any(body)
		if err != nil {
			return nil, fmt.Errorf("invalid Shadowsocks URL: cannot decode body")
		}
		dec := string(decodedBytes)

		atIdx := strings.LastIndex(dec, "@")
		if atIdx < 0 {
			return nil, fmt.Errorf("invalid Shadowsocks URL: missing @ separator")
		}

		parts := strings.SplitN(dec[:atIdx], ":", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid Shadowsocks URL: expected method:password")
		}
		method = parts[0]
		password = parts[1]

		host, port, err = parseHostPort(dec[atIdx+1:])
		if err != nil {
			return nil, fmt.Errorf("invalid Shadowsocks host:port: %w", err)
		}
	}

	if method == "" {
		return nil, fmt.Errorf("shadowsocks URL missing encryption method")
	}
	if password == "" {
		return nil, fmt.Errorf("shadowsocks URL missing password")
	}
	if host == "" {
		return nil, fmt.Errorf("missing host in Shadowsocks URL")
	}
	if port <= 0 || port > 65535 {
		return nil, fmt.Errorf("invalid port in Shadowsocks URL")
	}

	if !supportedSSMethods[method] {
		return nil, fmt.Errorf("unsupported Shadowsocks method %q (supported: aes-256-gcm, chacha20-ietf-poly1305, 2022-blake3-aes-256-gcm)", method)
	}

	// URL-decode the fragment for the name
	remark := fragment
	if decoded, err := url.PathUnescape(remark); err == nil {
		remark = decoded
	}
	if remark == "" {
		remark = defaultRemark(host, port)
	}

	profile := &config.Profile{
		Name: remark,
		Server: config.ServerConfig{
			Address:    host,
			Port:       port,
			UUID:       password, // Shadowsocks stores password in UUID field
			Encryption: method,
			Protocol:   "shadowsocks",
			Transport: config.TransportConfig{
				Network: "tcp",
			},
			Security: config.SecurityConfig{
				Type: "none",
			},
		},
	}

	return profile, nil
}

// ToShadowsocksURL converts a Profile back to a Shadowsocks URL (SIP002 format).
func ToShadowsocksURL(p *config.Profile) string {
	userinfo := p.Server.Encryption + ":" + p.Server.UUID
	encoded := base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString([]byte(userinfo))
	fragment := url.PathEscape(p.Name)

	return fmt.Sprintf("ss://%s@%s:%d#%s",
		encoded, p.Server.Address, p.Server.Port, fragment)
}

// parseHostPort splits "host:port" into host and an integer port.
func parseHostPort(s string) (string, int, error) {
	host, portSpec, err := splitHostPort(s)
	if err != nil {
		return "", 0, err
	}
	if portSpec == "" {
		if strings.HasPrefix(s, "[") {
			return "", 0, fmt.Errorf("missing port after IPv6 address")
		}
		return "", 0, fmt.Errorf("missing port separator")
	}
	port, err := strconv.Atoi(portSpec)
	if err != nil {
		return "", 0, fmt.Errorf("invalid port: %w", err)
	}
	return host, port, nil
}
