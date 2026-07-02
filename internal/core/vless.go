package core

import (
	"fmt"
	"net/url"
	"strconv"

	"github.com/rtxnik/lazyray/internal/config"
)

// ParseVLESS parses a VLESS URL into a Profile.
// Format: vless://UUID@HOST:PORT?params#REMARK
func ParseVLESS(rawURL string) (*config.Profile, error) {
	parsed, err := parseUserinfoURL("vless", "VLESS", rawURL)
	if err != nil {
		return nil, err
	}

	uuid := parsed.User.Username()
	if uuid == "" {
		return nil, fmt.Errorf("missing UUID in VLESS URL")
	}

	host := parsed.Hostname()
	if host == "" {
		return nil, fmt.Errorf("missing host in VLESS URL")
	}

	portStr := parsed.Port()
	if portStr == "" {
		return nil, fmt.Errorf("missing port in VLESS URL")
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return nil, fmt.Errorf("invalid port: %w", err)
	}

	q := parsed.Query()

	// Extract remark from fragment
	remark := parsed.Fragment
	if remark == "" {
		remark = defaultRemark(host, port)
	}

	// Transport settings
	network := q.Get("type")
	if network == "" {
		network = "tcp"
	}

	path := q.Get("path")
	mode := q.Get("mode")
	transportHost := q.Get("host")

	// Security settings
	security := q.Get("security")
	if security == "" {
		security = "none"
	}

	sni := q.Get("sni")
	fingerprint := q.Get("fp")
	publicKey := q.Get("pbk")
	shortID := q.Get("sid")
	spiderX := q.Get("spx")
	flow := q.Get("flow")

	encryption := q.Get("encryption")
	if encryption == "" {
		encryption = "none"
	}

	profile := &config.Profile{
		Name: remark,
		Server: config.ServerConfig{
			Address:    host,
			Port:       port,
			UUID:       uuid,
			Encryption: encryption,
			Flow:       flow,
			Transport: config.TransportConfig{
				Network: network,
				Path:    path,
				Mode:    mode,
				Host:    transportHost,
			},
			Security: config.SecurityConfig{
				Type:        security,
				SNI:         sni,
				Fingerprint: fingerprint,
				PublicKey:   publicKey,
				ShortID:     shortID,
				SpiderX:     spiderX,
			},
		},
	}

	return profile, nil
}

// ToVLESSURL converts a Profile back to a VLESS URL.
func ToVLESSURL(p *config.Profile) string {
	params := url.Values{}

	if p.Server.Transport.Network != "" {
		params.Set("type", p.Server.Transport.Network)
	}
	if p.Server.Security.Type != "" {
		params.Set("security", p.Server.Security.Type)
	}
	if p.Server.Transport.Path != "" {
		params.Set("path", p.Server.Transport.Path)
	}
	if p.Server.Transport.Mode != "" {
		params.Set("mode", p.Server.Transport.Mode)
	}
	if p.Server.Transport.Host != "" {
		params.Set("host", p.Server.Transport.Host)
	}
	if p.Server.Security.SNI != "" {
		params.Set("sni", p.Server.Security.SNI)
	}
	if p.Server.Security.Fingerprint != "" {
		params.Set("fp", p.Server.Security.Fingerprint)
	}
	if p.Server.Security.PublicKey != "" {
		params.Set("pbk", p.Server.Security.PublicKey)
	}
	if p.Server.Security.ShortID != "" {
		params.Set("sid", p.Server.Security.ShortID)
	}
	if p.Server.Security.SpiderX != "" {
		params.Set("spx", p.Server.Security.SpiderX)
	}
	if p.Server.Flow != "" {
		params.Set("flow", p.Server.Flow)
	}
	if p.Server.Encryption != "" && p.Server.Encryption != "none" {
		params.Set("encryption", p.Server.Encryption)
	}

	fragment := url.PathEscape(p.Name)

	return fmt.Sprintf("vless://%s@%s:%d?%s#%s",
		p.Server.UUID, p.Server.Address, p.Server.Port,
		params.Encode(), fragment)
}
