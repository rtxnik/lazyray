package core

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/rtxnik/lazyray/internal/config"
)

// VMess URL format (standard v2rayN):
// vmess://base64({json with v, ps, add, port, id, aid, scy, net, type, host, path, tls, sni, fp})
//
// VMess URL format (alternative):
// vmess://uuid@host:port?params#remark (rare, but some tools use it)

// vmessJSON represents the v2rayN VMess JSON format.
type vmessJSON struct {
	V    string `json:"v"`   // Version (usually "2")
	PS   string `json:"ps"`  // Remark/name
	Add  string `json:"add"` // Server address
	Port any    `json:"port"`
	ID   string `json:"id"`  // UUID
	Aid  any    `json:"aid"` // Alter ID
	Scy  string `json:"scy"` // Security/cipher
	Net  string `json:"net"` // Network (tcp, ws, grpc, etc.)
	Type string `json:"type"`
	Host string `json:"host"`
	Path string `json:"path"`
	TLS  string `json:"tls"` // "tls" or ""
	SNI  string `json:"sni"`
	ALPN string `json:"alpn"`
	FP   string `json:"fp"` // Fingerprint
}

// ParseVMess parses a VMess URL into a Profile.
func ParseVMess(rawURL string) (*config.Profile, error) {
	if !strings.HasPrefix(rawURL, "vmess://") {
		return nil, fmt.Errorf("invalid VMess URL: must start with vmess://")
	}

	encoded := rawURL[len("vmess://"):]
	encoded = strings.TrimSpace(encoded)

	// Try base64 decode (standard v2rayN format)
	decoded, err := decodeBase64Any(encoded)
	if err != nil {
		return nil, fmt.Errorf("failed to decode VMess URL: not valid base64")
	}

	var vj vmessJSON
	if err := json.Unmarshal(decoded, &vj); err != nil {
		return nil, fmt.Errorf("parsing VMess JSON: %w", err)
	}

	if vj.ID == "" {
		return nil, fmt.Errorf("VMess URL missing UUID")
	}
	if vj.Add == "" {
		return nil, fmt.Errorf("VMess URL missing server address")
	}

	port := anyToInt(vj.Port)
	if port <= 0 {
		return nil, fmt.Errorf("VMess URL has invalid port")
	}

	alterID := anyToInt(vj.Aid)

	network := vj.Net
	if network == "" {
		network = "tcp"
	}

	security := vj.Scy
	if security == "" {
		security = "auto"
	}

	remark := vj.PS
	if remark == "" {
		remark = defaultRemark(vj.Add, port)
	}

	secType := "none"
	if vj.TLS == "tls" {
		secType = "tls"
	}

	profile := &config.Profile{
		Name: remark,
		Server: config.ServerConfig{
			Address:    vj.Add,
			Port:       port,
			UUID:       vj.ID,
			Encryption: security,
			Flow:       "",
			Protocol:   "vmess",
			AlterID:    alterID,
			Transport: config.TransportConfig{
				Network: network,
				Path:    vj.Path,
				Host:    vj.Host,
			},
			Security: config.SecurityConfig{
				Type:        secType,
				SNI:         vj.SNI,
				Fingerprint: vj.FP,
			},
		},
	}

	return profile, nil
}

// ToVMessURL converts a Profile back to a VMess URL (v2rayN format).
func ToVMessURL(p *config.Profile) string {
	tls := ""
	if p.Server.Security.Type == "tls" {
		tls = "tls"
	}

	vj := vmessJSON{
		V:    "2",
		PS:   p.Name,
		Add:  p.Server.Address,
		Port: p.Server.Port,
		ID:   p.Server.UUID,
		Aid:  p.Server.AlterID,
		Scy:  p.Server.Encryption,
		Net:  p.Server.Transport.Network,
		Host: p.Server.Transport.Host,
		Path: p.Server.Transport.Path,
		TLS:  tls,
		SNI:  p.Server.Security.SNI,
		FP:   p.Server.Security.Fingerprint,
	}

	data, _ := json.Marshal(vj)
	return "vmess://" + base64.StdEncoding.EncodeToString(data)
}

// ParseProxyURL auto-detects the protocol and parses accordingly.
func ParseProxyURL(rawURL string) (*config.Profile, error) {
	rawURL = strings.TrimSpace(rawURL)
	if i := strings.Index(rawURL, "://"); i >= 0 {
		if proto, ok := schemeToProtocol[rawURL[:i]]; ok {
			p, err := protocols[proto].Parse(rawURL)
			if err != nil {
				return nil, err
			}
			SanitizeProfileDisplay(p)
			return p, nil
		}
	}
	return nil, fmt.Errorf("unsupported protocol URL (expected vless://, vmess://, trojan://, ss://, or hysteria2://)")
}

// ToProxyURL converts a Profile back to its protocol-specific URL.
func ToProxyURL(p *config.Profile) string {
	if spec, ok := protocolFor(p.Server.Protocol); ok {
		return spec.ToURL(p)
	}
	return ToVLESSURL(p) // default, matching prior behavior
}

// ParseTrojan parses a Trojan URL into a Profile.
// Format: trojan://password@host:port?params#remark
func ParseTrojan(rawURL string) (*config.Profile, error) {
	parsed, err := parseUserinfoURL("trojan", "Trojan", rawURL)
	if err != nil {
		return nil, err
	}

	password := parsed.User.Username()
	if password == "" {
		return nil, fmt.Errorf("missing password in Trojan URL")
	}

	host := parsed.Hostname()
	if host == "" {
		return nil, fmt.Errorf("missing host in Trojan URL")
	}

	portStr := parsed.Port()
	if portStr == "" {
		portStr = "443" // Trojan default port
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return nil, fmt.Errorf("invalid port: %w", err)
	}

	q := parsed.Query()

	remark := parsed.Fragment
	if remark == "" {
		remark = defaultRemark(host, port)
	}

	network := q.Get("type")
	if network == "" {
		network = "tcp"
	}

	security := q.Get("security")
	if security == "" {
		security = "tls"
	}

	sni := q.Get("sni")
	if sni == "" {
		sni = host
	}
	fp := q.Get("fp")
	path := q.Get("path")
	transportHost := q.Get("host")
	alpn := q.Get("alpn")

	profile := &config.Profile{
		Name: remark,
		Server: config.ServerConfig{
			Address:    host,
			Port:       port,
			UUID:       password, // Trojan uses password in UUID field
			Encryption: "none",
			Protocol:   "trojan",
			Transport: config.TransportConfig{
				Network: network,
				Path:    path,
				Host:    transportHost,
			},
			Security: config.SecurityConfig{
				Type:        security,
				SNI:         sni,
				Fingerprint: fp,
				ALPN:        alpn,
			},
		},
	}

	return profile, nil
}

// ToTrojanURL converts a Profile back to a Trojan URL.
func ToTrojanURL(p *config.Profile) string {
	params := url.Values{}

	if p.Server.Transport.Network != "" && p.Server.Transport.Network != "tcp" {
		params.Set("type", p.Server.Transport.Network)
	}
	if p.Server.Security.Type != "" && p.Server.Security.Type != "tls" {
		params.Set("security", p.Server.Security.Type)
	}
	if p.Server.Security.SNI != "" {
		params.Set("sni", p.Server.Security.SNI)
	}
	if p.Server.Security.Fingerprint != "" {
		params.Set("fp", p.Server.Security.Fingerprint)
	}
	if p.Server.Transport.Path != "" {
		params.Set("path", p.Server.Transport.Path)
	}
	if p.Server.Transport.Host != "" {
		params.Set("host", p.Server.Transport.Host)
	}
	if p.Server.Security.ALPN != "" {
		params.Set("alpn", p.Server.Security.ALPN)
	}

	fragment := url.PathEscape(p.Name)
	query := params.Encode()
	if query != "" {
		query = "?" + query
	}

	return fmt.Sprintf("trojan://%s@%s:%d%s#%s",
		p.Server.UUID, bracketIPv6(p.Server.Address), p.Server.Port,
		query, fragment)
}
