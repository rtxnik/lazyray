package config

import (
	"errors"
	"fmt"
	"os"

	"github.com/rtxnik/lazyray/internal/fsutil"
	"gopkg.in/yaml.v3"
)

// ServerConfig represents a proxy server profile.
type ServerConfig struct {
	Address         string          `yaml:"address"`
	Port            int             `yaml:"port"`
	UUID            string          `yaml:"uuid"`
	Encryption      string          `yaml:"encryption"`
	Flow            string          `yaml:"flow"`
	Protocol        string          `yaml:"protocol,omitempty"`        // "vless" (default), "vmess", "trojan", "shadowsocks"
	AlterID         int             `yaml:"alterId,omitempty"`         // VMess only
	Obfs            string          `yaml:"obfs,omitempty"`            // Hysteria2 obfs type (e.g. "salamander")
	ObfsPassword    string          `yaml:"obfsPassword,omitempty"`    // Hysteria2 obfs password
	PortHopping     string          `yaml:"portHopping,omitempty"`     // Hysteria2 multi-port spec, e.g. "443,5000-6000"
	PortHopInterval string          `yaml:"portHopInterval,omitempty"` // Hysteria2 port-hop interval, e.g. "30" or "10-30"
	Transport       TransportConfig `yaml:"transport"`
	Security        SecurityConfig  `yaml:"security"`
}

// TransportConfig holds transport settings.
type TransportConfig struct {
	Network              string `yaml:"network"`
	Path                 string `yaml:"path"`
	Mode                 string `yaml:"mode"`
	Host                 string `yaml:"host,omitempty"`
	MaxConcurrentUploads int    `yaml:"maxConcurrentUploads,omitempty"` // SplitHTTP only
	MaxUploadSize        int    `yaml:"maxUploadSize,omitempty"`        // SplitHTTP only
}

// SecurityConfig holds TLS/Reality settings.
type SecurityConfig struct {
	Type          string `yaml:"type"`
	SNI           string `yaml:"sni"`
	Fingerprint   string `yaml:"fingerprint"`
	PublicKey     string `yaml:"publicKey"`
	ShortID       string `yaml:"shortId"`
	SpiderX       string `yaml:"spiderX"`
	ALPN          string `yaml:"alpn,omitempty"`
	AllowInsecure bool   `yaml:"allowInsecure,omitempty"`
	PinSHA256     string `yaml:"pinSHA256,omitempty"` // Hysteria2 cert pin: comma-separated hex SHA-256 of cert DER
}

// GetProtocol returns the effective protocol, defaulting to "vless".
func (s *ServerConfig) GetProtocol() string {
	if s.Protocol == "" {
		return "vless"
	}
	return s.Protocol
}

// SSHConfig holds SSH tunnel settings for a server.
type SSHConfig struct {
	Host    string      `yaml:"host"`
	Port    int         `yaml:"port"`
	User    string      `yaml:"user"`
	KeyPath string      `yaml:"keyPath"`
	Panel   PanelConfig `yaml:"panel"`
}

// PanelConfig holds panel access settings.
type PanelConfig struct {
	Port int    `yaml:"port"`
	Path string `yaml:"path"`
}

// DNSRule maps domains to a specific DNS server for conditional DNS routing.
type DNSRule struct {
	Server    string   `yaml:"server"`
	Domains   []string `yaml:"domains,omitempty"`
	ExpectIPs []string `yaml:"expectIPs,omitempty"`
}

// ProfileRouting holds per-profile routing rules.
type ProfileRouting struct {
	Bypass   []string  `yaml:"bypass,omitempty"`
	Block    []string  `yaml:"block,omitempty"`
	DNSRules []DNSRule `yaml:"dnsRules,omitempty"`
}

// Profile represents a single proxy profile.
type Profile struct {
	Name           string         `yaml:"name"`
	Default        bool           `yaml:"default,omitempty"`
	Server         ServerConfig   `yaml:"server"`
	Chain          []ServerConfig `yaml:"chain,omitempty"`
	SSH            SSHConfig      `yaml:"ssh,omitempty"`
	ExpectedExitIP string         `yaml:"expectedExitIP,omitempty"`
	Routing        ProfileRouting `yaml:"routing,omitempty"`
	Group          string         `yaml:"group,omitempty"`
	Tags           []string       `yaml:"tags,omitempty"`
	Latency        int64          `yaml:"latency,omitempty"`
	Subscription   string         `yaml:"subscription,omitempty"`
}

// Clone returns a deep copy of p whose slice-backed fields share no storage
// with the original, so mutating one profile cannot corrupt another. Nil-ness
// is preserved: a nil input slice stays nil in the clone (so the YAML
// round-trip via omitempty is unchanged).
func (p Profile) Clone() Profile {
	clone := p
	clone.Chain = cloneSlice(p.Chain)
	clone.Tags = cloneSlice(p.Tags)
	clone.Routing.Bypass = cloneSlice(p.Routing.Bypass)
	clone.Routing.Block = cloneSlice(p.Routing.Block)
	if p.Routing.DNSRules != nil {
		rules := make([]DNSRule, len(p.Routing.DNSRules))
		for i, r := range p.Routing.DNSRules {
			r.Domains = cloneSlice(r.Domains)
			r.ExpectIPs = cloneSlice(r.ExpectIPs)
			rules[i] = r
		}
		clone.Routing.DNSRules = rules
	}
	return clone
}

// cloneSlice returns an independent copy of s, preserving nil-ness: a nil input
// yields a nil output rather than an empty non-nil slice.
func cloneSlice[T any](s []T) []T {
	return append([]T(nil), s...)
}

// IsChained returns true if the profile uses multi-hop chaining.
func (p *Profile) IsChained() bool {
	return len(p.Chain) > 0
}

// ChainServers returns the full ordered list of servers.
// Entry node first, exit node last. For single-hop, returns just Server.
func (p *Profile) ChainServers() []ServerConfig {
	if len(p.Chain) == 0 {
		return []ServerConfig{p.Server}
	}
	return append([]ServerConfig{p.Server}, p.Chain...)
}

// CurrentConfigVersion is the latest servers.yaml schema version.
const CurrentConfigVersion = 2

// ServersConfig is the top-level structure of servers.yaml.
type ServersConfig struct {
	ConfigVersion int       `yaml:"configVersion,omitempty"`
	Profiles      []Profile `yaml:"profiles"`
}

// LoadServers loads server profiles from servers.yaml.
// If the config is an older version, it migrates profiles and saves a backup.
func LoadServers() (*ServersConfig, error) {
	path := ServersPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &ServersConfig{ConfigVersion: CurrentConfigVersion}, nil
		}
		return nil, fmt.Errorf("reading servers config: %w", err)
	}

	var cfg ServersConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing servers config: %w", err)
	}

	if cfg.ConfigVersion < CurrentConfigVersion && len(cfg.Profiles) > 0 {
		// Back up the pre-migration file exactly once. O_EXCL means a repeat
		// migration (e.g. after an interrupted save) never re-clobbers the true
		// original — that backup is the only copy of the user's original data.
		backupPath := fmt.Sprintf("%s.v%d.bak", path, cfg.ConfigVersion)
		if err := writeBackupOnce(backupPath, data); err != nil {
			return nil, fmt.Errorf("backing up servers config before migration: %w", err)
		}

		migrateProfiles(cfg.Profiles)
		cfg.ConfigVersion = CurrentConfigVersion

		// Persist the migrated config and fail loudly: a swallowed error would
		// re-migrate (and could re-clobber) on every subsequent load.
		if err := SaveServers(&cfg); err != nil {
			return nil, fmt.Errorf("saving migrated servers config: %w", err)
		}
	} else if cfg.ConfigVersion == 0 && len(cfg.Profiles) == 0 {
		cfg.ConfigVersion = CurrentConfigVersion
	}

	return &cfg, nil
}

// migrateProfiles applies default values for missing fields in profiles.
func migrateProfiles(profiles []Profile) {
	for i := range profiles {
		p := &profiles[i]
		migrateServerConfig(&p.Server)
		for j := range p.Chain {
			migrateServerConfig(&p.Chain[j])
		}
	}
}

// migrateServerConfig fills in missing fields with protocol-appropriate defaults.
func migrateServerConfig(s *ServerConfig) {
	if s.Protocol == "" {
		s.Protocol = "vless"
	}
	if s.Encryption == "" {
		switch s.Protocol {
		case "vless":
			s.Encryption = "none"
		case "vmess":
			s.Encryption = "auto"
		}
	}
	if s.Transport.Network == "" {
		s.Transport.Network = "tcp"
	}
}

// SaveServers writes server profiles to servers.yaml.
func SaveServers(cfg *ServersConfig) error {
	if err := EnsureDirs(); err != nil {
		return err
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshaling servers config: %w", err)
	}

	path := ServersPath()
	if err := fsutil.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("writing servers config: %w", err)
	}
	return nil
}

// DefaultProfile returns the default profile or the first one.
func (c *ServersConfig) DefaultProfile() *Profile {
	for i := range c.Profiles {
		if c.Profiles[i].Default {
			return &c.Profiles[i]
		}
	}
	if len(c.Profiles) > 0 {
		return &c.Profiles[0]
	}
	return nil
}

// SetDefault sets the profile at index as default, unsetting others.
func (c *ServersConfig) SetDefault(index int) error {
	if index < 0 || index >= len(c.Profiles) {
		return fmt.Errorf("profile index %d out of range (0-%d)", index, len(c.Profiles)-1)
	}
	for i := range c.Profiles {
		c.Profiles[i].Default = (i == index)
	}
	return nil
}

// HasUUID checks if any profile already uses the given UUID.
func (c *ServersConfig) HasUUID(uuid string) (string, bool) {
	for _, p := range c.Profiles {
		if p.Server.UUID == uuid {
			return p.Name, true
		}
	}
	return "", false
}

// MaskUUID returns a masked UUID for display.
func MaskUUID(uuid string) string {
	if len(uuid) < 9 {
		return "****"
	}
	return uuid[:8] + "-****-****-****-************"
}

// writeBackupOnce writes data to path only if path does not already exist. An
// existing backup (a prior migration already captured the original) is left
// untouched and is not an error. Any other failure is returned.
func writeBackupOnce(path string, data []byte) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			return nil // already backed up; never clobber the true original
		}
		return err
	}
	defer f.Close()
	if _, err := f.Write(data); err != nil {
		return err
	}
	return f.Sync()
}
