package core

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/rtxnik/lazyray/internal/config"
	"github.com/rtxnik/lazyray/internal/fsutil"
)

// StatsAPIPort is the local port used by the xray stats gRPC API.
const StatsAPIPort = 10813

// XrayConfig represents the full xray configuration.
type XrayConfig struct {
	Log       XrayLog       `json:"log"`
	API       *XrayAPI      `json:"api,omitempty"`
	Stats     *XrayStats    `json:"stats,omitempty"`
	Policy    *XrayPolicy   `json:"policy,omitempty"`
	DNS       XrayDNS       `json:"dns,omitempty"`
	Inbounds  []Inbound     `json:"inbounds"`
	Outbounds []Outbound    `json:"outbounds"`
	Routing   RoutingConfig `json:"routing"`
}

// XrayAPI configures the xray gRPC API.
type XrayAPI struct {
	Tag      string   `json:"tag"`
	Services []string `json:"services"`
}

// XrayStats enables the stats collection (empty object is enough).
type XrayStats struct{}

// XrayPolicy configures stats collection policies.
type XrayPolicy struct {
	System XrayPolicySystem `json:"system"`
}

// XrayPolicySystem configures system-level stats.
type XrayPolicySystem struct {
	StatsInboundUplink    bool `json:"statsInboundUplink"`
	StatsInboundDownlink  bool `json:"statsInboundDownlink"`
	StatsOutboundUplink   bool `json:"statsOutboundUplink"`
	StatsOutboundDownlink bool `json:"statsOutboundDownlink"`
}

type XrayLog struct {
	LogLevel string `json:"loglevel"`
	Access   string `json:"access,omitempty"`
	Error    string `json:"error,omitempty"`
}

// XrayDNS configures xray DNS resolution. Servers can be plain IP addresses
// (strings) or advanced server objects with address, domains, and expectIPs
// for DoH/DoT and conditional DNS routing.
type XrayDNS struct {
	Servers []interface{} `json:"servers"`
}

// DNSServerObject represents an advanced xray DNS server entry.
// Used for DoH (https://), DoT (tcp://), and conditional DNS routing.
type DNSServerObject struct {
	Address   string   `json:"address"`
	Domains   []string `json:"domains,omitempty"`
	ExpectIPs []string `json:"expectIPs,omitempty"`
}

type Inbound struct {
	Tag      string          `json:"tag"`
	Port     int             `json:"port"`
	Listen   string          `json:"listen"`
	Protocol string          `json:"protocol"`
	Settings json.RawMessage `json:"settings"`
	Sniffing *Sniffing       `json:"sniffing,omitempty"`
}

type Sniffing struct {
	Enabled      bool     `json:"enabled"`
	DestOverride []string `json:"destOverride"`
}

type Outbound struct {
	Tag            string          `json:"tag"`
	Protocol       string          `json:"protocol"`
	Settings       json.RawMessage `json:"settings,omitempty"`
	StreamSettings *StreamSettings `json:"streamSettings,omitempty"`
	ProxySettings  *ProxySettings  `json:"proxySettings,omitempty"`
}

// ProxySettings specifies that traffic should be forwarded through another outbound.
type ProxySettings struct {
	Tag string `json:"tag"`
}

type StreamSettings struct {
	Network             string               `json:"network"`
	XHTTPSettings       *XHTTPSettings       `json:"xhttpSettings,omitempty"`
	WSSettings          *WSSettings          `json:"wsSettings,omitempty"`
	GRPCSettings        *GRPCSettings        `json:"grpcSettings,omitempty"`
	TCPSettings         *TCPSettings         `json:"tcpSettings,omitempty"`
	H2Settings          *H2Settings          `json:"httpSettings,omitempty"`
	HTTPUpgradeSettings *HTTPUpgradeSettings `json:"httpupgradeSettings,omitempty"`
	SplitHTTPSettings   *SplitHTTPSettings   `json:"splithttpSettings,omitempty"`
	Security            string               `json:"security"`
	TLSSettings         *TLSSettings         `json:"tlsSettings,omitempty"`
	RealitySettings     *RealitySettings     `json:"realitySettings,omitempty"`
	HysteriaSettings    *HysteriaSettings    `json:"hysteriaSettings,omitempty"`
	FinalMask           *FinalMask           `json:"finalmask,omitempty"`
}

// WSSettings holds WebSocket transport settings.
type WSSettings struct {
	Path    string            `json:"path"`
	Headers map[string]string `json:"headers,omitempty"`
}

// GRPCSettings holds gRPC transport settings.
type GRPCSettings struct {
	ServiceName string `json:"serviceName"`
}

// TCPSettings holds TCP transport settings (for HTTP header camouflage).
type TCPSettings struct {
	Header *TCPHeader `json:"header,omitempty"`
}

// TCPHeader configures TCP header camouflage.
type TCPHeader struct {
	Type string `json:"type"`
}

// TLSSettings holds TLS configuration.
type TLSSettings struct {
	ServerName           string   `json:"serverName,omitempty"`
	Fingerprint          string   `json:"fingerprint,omitempty"`
	ALPN                 []string `json:"alpn,omitempty"`
	AllowInsecure        bool     `json:"allowInsecure,omitempty"`
	PinnedPeerCertSha256 string   `json:"pinnedPeerCertSha256,omitempty"`
}

type XHTTPSettings struct {
	Path string `json:"path"`
	Mode string `json:"mode,omitempty"`
	Host string `json:"host,omitempty"`
}

// H2Settings holds HTTP/2 transport settings.
type H2Settings struct {
	Host []string `json:"host,omitempty"`
	Path string   `json:"path"`
}

// HTTPUpgradeSettings holds HTTPUpgrade transport settings.
type HTTPUpgradeSettings struct {
	Host string `json:"host,omitempty"`
	Path string `json:"path"`
}

// SplitHTTPSettings holds SplitHTTP transport settings.
type SplitHTTPSettings struct {
	Host                 string `json:"host,omitempty"`
	Path                 string `json:"path"`
	MaxConcurrentUploads int    `json:"maxConcurrentUploads,omitempty"`
	MaxUploadSize        int    `json:"maxUploadSize,omitempty"`
}

type RealitySettings struct {
	ServerName  string `json:"serverName"`
	Fingerprint string `json:"fingerprint"`
	PublicKey   string `json:"publicKey"`
	ShortID     string `json:"shortId"`
	SpiderX     string `json:"spiderX,omitempty"`
}

// HysteriaSettings holds Hysteria2 transport settings (streamSettings.hysteriaSettings).
type HysteriaSettings struct {
	Version        int    `json:"version"`
	Auth           string `json:"auth"`
	UDPIdleTimeout int    `json:"udpIdleTimeout,omitempty"`
}

// FinalMask holds Hysteria2 obfuscation masks and QUIC params (streamSettings.finalmask).
type FinalMask struct {
	UDP        []Mask      `json:"udp,omitempty"`
	QuicParams *QuicParams `json:"quicParams,omitempty"`
}

// QuicParams holds the finalmask QUIC tuning block (port-hopping lives here).
type QuicParams struct {
	UdpHop *UdpHop `json:"udpHop,omitempty"`
}

// UdpHop configures Hysteria2 port hopping.
type UdpHop struct {
	Ports    string `json:"ports,omitempty"`    // e.g. "443,5000-6000"
	Interval string `json:"interval,omitempty"` // e.g. "30" or "10-30"
}

// Mask is a single obfuscation mask entry.
type Mask struct {
	Type     string      `json:"type"`
	Settings MaskSetting `json:"settings"`
}

// MaskSetting holds the password for a salamander mask.
type MaskSetting struct {
	Password string `json:"password"`
}

type RoutingConfig struct {
	DomainStrategy string        `json:"domainStrategy"`
	Rules          []RoutingRule `json:"rules"`
}

type RoutingRule struct {
	Type        string   `json:"type"`
	Domain      []string `json:"domain,omitempty"`
	IP          []string `json:"ip,omitempty"`
	InboundTag  []string `json:"inboundTag,omitempty"`
	OutboundTag string   `json:"outboundTag"`
}

// VNExtUser represents a VLESS user.
type VNExtUser struct {
	ID         string `json:"id"`
	Encryption string `json:"encryption"`
	Flow       string `json:"flow"`
}

// VNExt represents a VLESS server endpoint.
type VNExt struct {
	Address string      `json:"address"`
	Port    int         `json:"port"`
	Users   []VNExtUser `json:"users"`
}

// buildStreamSettings creates StreamSettings from a ServerConfig, shared across protocols.
func buildStreamSettings(server config.ServerConfig) *StreamSettings {
	stream := &StreamSettings{
		Network:  server.Transport.Network,
		Security: server.Security.Type,
	}

	switch server.Transport.Network {
	case "xhttp":
		stream.XHTTPSettings = &XHTTPSettings{
			Path: server.Transport.Path,
			Mode: server.Transport.Mode,
		}
		if server.Transport.Host != "" {
			stream.XHTTPSettings.Host = server.Transport.Host
		}
	case "ws":
		ws := &WSSettings{Path: server.Transport.Path}
		if server.Transport.Host != "" {
			ws.Headers = map[string]string{"Host": server.Transport.Host}
		}
		stream.WSSettings = ws
	case "grpc":
		stream.GRPCSettings = &GRPCSettings{ServiceName: server.Transport.Path}
	case "h2":
		h2 := &H2Settings{Path: server.Transport.Path}
		if server.Transport.Host != "" {
			h2.Host = strings.Split(server.Transport.Host, ",")
		}
		stream.H2Settings = h2
	case "httpupgrade":
		hu := &HTTPUpgradeSettings{Path: server.Transport.Path}
		if server.Transport.Host != "" {
			hu.Host = server.Transport.Host
		}
		stream.HTTPUpgradeSettings = hu
	case "splithttp":
		sh := &SplitHTTPSettings{
			Path:                 server.Transport.Path,
			MaxConcurrentUploads: server.Transport.MaxConcurrentUploads,
			MaxUploadSize:        server.Transport.MaxUploadSize,
		}
		if server.Transport.Host != "" {
			sh.Host = server.Transport.Host
		}
		stream.SplitHTTPSettings = sh
	case "hysteria":
		stream.HysteriaSettings = &HysteriaSettings{
			Version:        2,
			Auth:           server.UUID,
			UDPIdleTimeout: 60,
		}
		var fm *FinalMask
		if server.Obfs == "salamander" {
			fm = &FinalMask{UDP: []Mask{{
				Type:     "salamander",
				Settings: MaskSetting{Password: server.ObfsPassword},
			}}}
		}
		if server.PortHopping != "" {
			if fm == nil {
				fm = &FinalMask{}
			}
			hop := &UdpHop{Ports: server.PortHopping}
			if server.PortHopInterval != "" {
				hop.Interval = server.PortHopInterval
			}
			fm.QuicParams = &QuicParams{UdpHop: hop}
		}
		stream.FinalMask = fm
	}

	switch server.Security.Type {
	case "reality":
		stream.RealitySettings = &RealitySettings{
			ServerName:  server.Security.SNI,
			Fingerprint: server.Security.Fingerprint,
			PublicKey:   server.Security.PublicKey,
			ShortID:     server.Security.ShortID,
			SpiderX:     server.Security.SpiderX,
		}
	case "tls":
		tls := &TLSSettings{
			ServerName:  server.Security.SNI,
			Fingerprint: server.Security.Fingerprint,
		}
		if server.Security.ALPN != "" {
			tls.ALPN = strings.Split(server.Security.ALPN, ",")
		}
		// Certificate pinning is stronger than allowInsecure and survives
		// self-signed certs, so when a pin is present we emit it and do NOT set
		// allowInsecure (which is also a removed feature on xray-core main).
		if server.Security.PinSHA256 != "" {
			tls.PinnedPeerCertSha256 = server.Security.PinSHA256
		} else {
			tls.AllowInsecure = server.Security.AllowInsecure
		}
		stream.TLSSettings = tls
	}

	return stream
}

// newOutbound assembles the common Outbound envelope (tag, protocol, stream
// settings, optional chain proxyTag) around a protocol-specific settings blob.
func newOutbound(tag, protocol string, settings json.RawMessage, server config.ServerConfig, proxyTag string) Outbound {
	ob := Outbound{
		Tag:            tag,
		Protocol:       protocol,
		Settings:       settings,
		StreamSettings: buildStreamSettings(server),
	}
	if proxyTag != "" {
		ob.ProxySettings = &ProxySettings{Tag: proxyTag}
	}
	return ob
}

// buildVMessOutbound creates a VMess outbound for a single server.
func buildVMessOutbound(tag string, server config.ServerConfig, proxyTag string) Outbound {
	security := server.Encryption
	if security == "" {
		security = "auto"
	}

	type vmessUser struct {
		ID       string `json:"id"`
		AlterID  int    `json:"alterId"`
		Security string `json:"security"`
	}
	type vmessVnext struct {
		Address string      `json:"address"`
		Port    int         `json:"port"`
		Users   []vmessUser `json:"users"`
	}

	vnext := []vmessVnext{{
		Address: server.Address,
		Port:    server.Port,
		Users: []vmessUser{{
			ID:       server.UUID,
			AlterID:  server.AlterID,
			Security: security,
		}},
	}}
	settings, _ := json.Marshal(map[string]interface{}{"vnext": vnext})
	return newOutbound(tag, "vmess", settings, server, proxyTag)
}

// buildTrojanOutbound creates a Trojan outbound for a single server.
func buildTrojanOutbound(tag string, server config.ServerConfig, proxyTag string) Outbound {
	type trojanServer struct {
		Address  string `json:"address"`
		Port     int    `json:"port"`
		Password string `json:"password"`
	}

	servers := []trojanServer{{
		Address:  server.Address,
		Port:     server.Port,
		Password: server.UUID, // Trojan stores password in UUID field
	}}
	settings, _ := json.Marshal(map[string]interface{}{"servers": servers})
	return newOutbound(tag, "trojan", settings, server, proxyTag)
}

// buildShadowsocksOutbound creates a Shadowsocks outbound for a single server.
func buildShadowsocksOutbound(tag string, server config.ServerConfig, proxyTag string) Outbound {
	type ssServer struct {
		Address  string `json:"address"`
		Port     int    `json:"port"`
		Method   string `json:"method"`
		Password string `json:"password"`
	}

	servers := []ssServer{{
		Address:  server.Address,
		Port:     server.Port,
		Method:   server.Encryption,
		Password: server.UUID, // Shadowsocks stores password in UUID field
	}}
	settings, _ := json.Marshal(map[string]interface{}{"servers": servers})
	return newOutbound(tag, "shadowsocks", settings, server, proxyTag)
}

// buildHysteria2Outbound creates a Hysteria2 outbound for a single server.
// Xray models Hysteria2 as protocol "hysteria" + the hysteria stream transport.
func buildHysteria2Outbound(tag string, server config.ServerConfig, proxyTag string) Outbound {
	settings, _ := json.Marshal(map[string]interface{}{
		"version": 2,
		"address": server.Address,
		"port":    server.Port,
	})
	return newOutbound(tag, "hysteria", settings, server, proxyTag)
}

// buildOutbound creates a protocol-appropriate outbound based on the server's protocol.
func buildOutbound(tag string, server config.ServerConfig, proxyTag string) Outbound {
	if spec, ok := protocolFor(server.GetProtocol()); ok {
		return spec.BuildOutbound(tag, server, proxyTag)
	}
	return buildVLESSOutbound(tag, server, proxyTag) // default
}

// ValidateProfile checks a profile for common configuration errors.
func ValidateProfile(profile *config.Profile) error {
	var errs []string

	proto := profile.Server.GetProtocol()

	validate := validateGeneric
	if spec, ok := protocolFor(proto); ok {
		validate = spec.Validate
	}
	errs = append(errs, validate(profile.Server)...)

	if profile.Server.Address == "" {
		errs = append(errs, "server address is empty")
	}
	if profile.Server.Port <= 0 || profile.Server.Port > 65535 {
		errs = append(errs, fmt.Sprintf("invalid port %d (must be 1-65535)", profile.Server.Port))
	}
	if profile.Server.Transport.Network == "" {
		errs = append(errs, "transport network is empty")
	}

	if profile.Server.Security.Type == "reality" {
		if profile.Server.Security.PublicKey == "" {
			errs = append(errs, "reality publicKey is required")
		}
		if profile.Server.Security.SNI == "" {
			errs = append(errs, "reality SNI is required")
		}
		if profile.Server.Security.Fingerprint == "" {
			errs = append(errs, "reality fingerprint is required")
		}
	}

	// Generic TLS branch: a cert pin on any non-datagram protocol must satisfy
	// the same 64-hex / 32-byte rule the hysteria2 branch enforces.
	// The hysteria2 case validates its pin in the per-protocol validate above, so it
	// is excluded here to avoid a duplicate error.
	if !isDatagramTransport(proto) && profile.Server.Security.Type == "tls" {
		if pin := profile.Server.Security.PinSHA256; pin != "" {
			if err := validatePinSHA256(pin); err != nil {
				errs = append(errs, err.Error())
			}
		}
	}

	for i := range profile.Chain {
		node := profile.Chain[i]
		if node.Address == "" {
			errs = append(errs, fmt.Sprintf("chain[%d] server address is empty", i))
		}
		if node.Port <= 0 || node.Port > 65535 {
			errs = append(errs, fmt.Sprintf("chain[%d] invalid port %d (must be 1-65535)", i, node.Port))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("profile %q: %s", profile.Name, strings.Join(errs, "; "))
	}
	return nil
}

// buildVLESSOutbound creates a VLESS outbound for a single server.
func buildVLESSOutbound(tag string, server config.ServerConfig, proxyTag string) Outbound {
	vnext := []VNExt{{
		Address: server.Address,
		Port:    server.Port,
		Users: []VNExtUser{{
			ID:         server.UUID,
			Encryption: server.Encryption,
			Flow:       server.Flow,
		}},
	}}
	settings, _ := json.Marshal(map[string]interface{}{"vnext": vnext})
	return newOutbound(tag, "vless", settings, server, proxyTag)
}

// accessLogTarget returns the xray access-log destination: the log path only
// when explicitly enabled, otherwise "none" (privacy default — the access log
// records browsing destinations).
func accessLogTarget(settings *config.Settings) string {
	if settings.Xray.AccessLog == "file" {
		return config.AccessLogPath()
	}
	return "none"
}

// GenerateXrayConfig creates a full xray config from a profile and settings.
func GenerateXrayConfig(profile *config.Profile, settings *config.Settings) (*XrayConfig, error) {
	socksSettings, _ := json.Marshal(map[string]interface{}{
		"auth": "noauth",
		"udp":  true,
	})

	httpSettings, _ := json.Marshal(map[string]interface{}{})

	directSettings, _ := json.Marshal(map[string]interface{}{})
	blockSettings, _ := json.Marshal(map[string]interface{}{})

	// Stats API inbound (dokodemo-door for gRPC)
	apiSettings, _ := json.Marshal(map[string]interface{}{
		"address": "127.0.0.1",
	})

	inbounds := []Inbound{
		{
			Tag:      "socks-in",
			Port:     settings.Local.SocksPort,
			Listen:   settings.Local.Listen,
			Protocol: "socks",
			Settings: socksSettings,
			Sniffing: &Sniffing{
				Enabled:      true,
				DestOverride: []string{"http", "tls"},
			},
		},
		{
			Tag:      "http-in",
			Port:     settings.Local.HTTPPort,
			Listen:   settings.Local.Listen,
			Protocol: "http",
			Settings: httpSettings,
		},
		{
			Tag:      "api-in",
			Port:     StatsAPIPort,
			Listen:   "127.0.0.1",
			Protocol: "dokodemo-door",
			Settings: apiSettings,
		},
	}

	// Build proxy outbounds (single or chained)
	var proxyOutbounds []Outbound
	servers := profile.ChainServers()

	if len(servers) == 1 {
		// Single-hop: one proxy outbound
		proxyOutbounds = append(proxyOutbounds, buildOutbound("proxy", servers[0], ""))
	} else {
		// Multi-hop chain: entry→hop1→...→exit
		// In xray-core, each hop (except entry) references the previous via proxySettings.
		// Entry node (servers[0]) connects directly.
		// Each subsequent node routes through the previous hop.
		for i, server := range servers {
			tag := fmt.Sprintf("hop-%d", i)
			if i == len(servers)-1 {
				tag = "proxy" // Final exit node gets the "proxy" tag for routing
			}

			proxyTag := ""
			if i > 0 {
				proxyTag = fmt.Sprintf("hop-%d", i-1)
			}

			proxyOutbounds = append(proxyOutbounds, buildOutbound(tag, server, proxyTag))
		}
	}

	outbounds := append(proxyOutbounds,
		Outbound{
			Tag:      "direct",
			Protocol: "freedom",
			Settings: directSettings,
		},
		Outbound{
			Tag:      "block",
			Protocol: "blackhole",
			Settings: blockSettings,
		},
	)

	cfg := &XrayConfig{
		Log: XrayLog{
			LogLevel: settings.Xray.LogLevel,
			Access:   accessLogTarget(settings),
			Error:    config.ErrorLogPath(),
		},
		API: &XrayAPI{
			Tag:      "api",
			Services: []string{"StatsService"},
		},
		Stats: &XrayStats{},
		Policy: &XrayPolicy{
			System: XrayPolicySystem{
				StatsInboundUplink:    true,
				StatsInboundDownlink:  true,
				StatsOutboundUplink:   true,
				StatsOutboundDownlink: true,
			},
		},
		DNS:       buildDNSConfig(settings, profile),
		Inbounds:  inbounds,
		Outbounds: outbounds,
		Routing: RoutingConfig{
			DomainStrategy: "AsIs",
			Rules: []RoutingRule{
				{
					Type:        "field",
					InboundTag:  []string{"api-in"},
					OutboundTag: "api",
				},
				{
					Type:        "field",
					IP:          []string{"geoip:private"},
					OutboundTag: "direct",
				},
			},
		},
	}

	// Append profile-defined routing rules
	cfg.Routing.Rules = append(cfg.Routing.Rules, buildProfileRoutingRules(profile)...)

	return cfg, nil
}

// WriteXrayConfig generates and writes the xray config.json file.
func WriteXrayConfig(profile *config.Profile, settings *config.Settings) error {
	if err := ValidateProfile(profile); err != nil {
		return fmt.Errorf("invalid profile: %w", err)
	}

	if err := config.EnsureDirs(); err != nil {
		return err
	}

	cfg, err := GenerateXrayConfig(profile, settings)
	if err != nil {
		return fmt.Errorf("generating xray config: %w", err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling xray config: %w", err)
	}

	if err := fsutil.WriteFile(config.XrayConfigPath(), data, 0o600); err != nil {
		return fmt.Errorf("writing xray config: %w", err)
	}

	return nil
}

// buildProfileRoutingRules converts profile bypass/block entries into xray routing rules.
// Entries starting with "geoip:" or raw IPs go into the IP field;
// entries starting with "domain:", "geosite:", or "regexp:" go into the Domain field.
func buildProfileRoutingRules(profile *config.Profile) []RoutingRule {
	var rules []RoutingRule

	if bypassRule := buildRoutingRule(profile.Routing.Bypass, "direct"); bypassRule != nil {
		rules = append(rules, *bypassRule)
	}
	if blockRule := buildRoutingRule(profile.Routing.Block, "block"); blockRule != nil {
		rules = append(rules, *blockRule)
	}

	return rules
}

func buildRoutingRule(entries []string, outboundTag string) *RoutingRule {
	if len(entries) == 0 {
		return nil
	}

	var ips, domains []string
	for _, entry := range entries {
		if strings.HasPrefix(entry, "geoip:") || isIPEntry(entry) {
			ips = append(ips, entry)
		} else {
			domains = append(domains, entry)
		}
	}

	// Generate separate rules for IPs and domains since xray requires them split
	if len(ips) > 0 && len(domains) > 0 {
		// Return both as a single function doesn't support returning two,
		// so merge: xray actually supports IP + domain in the same rule as OR
		return &RoutingRule{
			Type:        "field",
			IP:          ips,
			Domain:      domains,
			OutboundTag: outboundTag,
		}
	}
	if len(ips) > 0 {
		return &RoutingRule{
			Type:        "field",
			IP:          ips,
			OutboundTag: outboundTag,
		}
	}
	return &RoutingRule{
		Type:        "field",
		Domain:      domains,
		OutboundTag: outboundTag,
	}
}

func isIPEntry(s string) bool {
	// Known domain-type prefixes are not IP entries
	for _, prefix := range []string{"domain:", "geosite:", "regexp:", "full:", "keyword:"} {
		if strings.HasPrefix(s, prefix) {
			return false
		}
	}
	// CIDR notation (10.0.0.0/8) or IPv6 (::1)
	if strings.Contains(s, "/") || strings.Contains(s, ":") {
		return true
	}
	// Plain IPv4 (starts with digit, contains dots)
	if len(s) > 0 && s[0] >= '0' && s[0] <= '9' && strings.Contains(s, ".") {
		return true
	}
	return false
}

// buildDNSConfig creates the xray DNS configuration from settings and profile.
// Plain IP addresses become string entries. DoH (https://) and DoT (tcp://)
// URLs become DNSServerObject entries. Profile DNS routing rules add domain
// mappings to specific DNS servers.
func buildDNSConfig(settings *config.Settings, profile *config.Profile) XrayDNS {
	dns := XrayDNS{}

	for _, server := range settings.Local.DNS {
		if isAdvancedDNS(server) {
			dns.Servers = append(dns.Servers, DNSServerObject{Address: server})
		} else {
			dns.Servers = append(dns.Servers, server)
		}
	}

	// Apply profile DNS routing rules if present
	if profile != nil {
		for _, rule := range profile.Routing.DNSRules {
			obj := DNSServerObject{Address: rule.Server}
			if len(rule.Domains) > 0 {
				obj.Domains = rule.Domains
			}
			if len(rule.ExpectIPs) > 0 {
				obj.ExpectIPs = rule.ExpectIPs
			}
			dns.Servers = append(dns.Servers, obj)
		}
	}

	if len(dns.Servers) == 0 {
		dns.Servers = []interface{}{"1.1.1.1", "8.8.8.8"}
	}

	return dns
}

// isAdvancedDNS returns true if the DNS server string is a DoH or DoT URL.
func isAdvancedDNS(server string) bool {
	return strings.HasPrefix(server, "https://") ||
		strings.HasPrefix(server, "https+local://") ||
		strings.HasPrefix(server, "tcp://")
}

// ReadXrayConfig reads the existing xray config.json.
func ReadXrayConfig() (*XrayConfig, error) {
	data, err := os.ReadFile(config.XrayConfigPath())
	if err != nil {
		return nil, fmt.Errorf("reading xray config: %w", err)
	}

	var cfg XrayConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing xray config: %w", err)
	}
	return &cfg, nil
}
