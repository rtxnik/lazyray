package config

import (
	"fmt"
	"os"

	"github.com/rtxnik/lazyray/internal/fsutil"
	"gopkg.in/yaml.v3"
)

// SubscriptionEntry represents a subscription URL with metadata.
type SubscriptionEntry struct {
	Name     string `yaml:"name"`
	URL      string `yaml:"url"`
	Interval int    `yaml:"interval,omitempty"`
}

// Settings represents the lazyray.yaml application settings.
type Settings struct {
	Local           LocalSettings        `yaml:"local"`
	Xray            XraySettings         `yaml:"xray"`
	Health          HealthSettings       `yaml:"health"`
	Update          UpdateSettings       `yaml:"update"`
	UI              UISettings           `yaml:"ui"`
	Notifications   NotificationSettings `yaml:"notifications"`
	Backup          BackupSettings       `yaml:"backup"`
	AutoSystemProxy bool                 `yaml:"autoSystemProxy"`
	Subscriptions   []SubscriptionEntry  `yaml:"subscriptions,omitempty"`
}

// BackupSettings holds backup rotation preferences.
type BackupSettings struct {
	MaxFiles int `yaml:"maxFiles,omitempty"`
}

// NotificationSettings holds notification preferences.
type NotificationSettings struct {
	Enabled bool `yaml:"enabled"`
}

// LocalSettings holds local proxy settings.
type LocalSettings struct {
	SocksPort int      `yaml:"socksPort"`
	HTTPPort  int      `yaml:"httpPort"`
	Listen    string   `yaml:"listen"`
	DNS       []string `yaml:"dns"`
}

// XraySettings holds xray behavior settings.
type XraySettings struct {
	AutoRestart bool   `yaml:"autoRestart"`
	LogLevel    string `yaml:"logLevel"`
	MaxLogSize  int    `yaml:"maxLogSize,omitempty"`
}

// HealthSettings holds health check settings.
type HealthSettings struct {
	Timeout        int    `yaml:"timeout"`
	AlertOnFailure bool   `yaml:"alertOnFailure"`
	IPCheckURL     string `yaml:"ipCheckURL,omitempty"`
	LatencyHost    string `yaml:"latencyHost,omitempty"`
	DNSCheckHost   string `yaml:"dnsCheckHost,omitempty"`
}

// UpdateSettings holds update settings.
type UpdateSettings struct {
	Channel      string `yaml:"channel"`
	AutoCheck    bool   `yaml:"autoCheck"`
	BackupBefore bool   `yaml:"backupBefore"`
	XrayVersion  string `yaml:"xrayVersion"`
}

// UISettings holds TUI settings.
type UISettings struct {
	Theme           string `yaml:"theme"`
	RefreshInterval int    `yaml:"refreshInterval"`
	LogLines        int    `yaml:"logLines"`
}

// DefaultSettings returns settings with default values.
func DefaultSettings() *Settings {
	return &Settings{
		Local: LocalSettings{
			SocksPort: 10808,
			HTTPPort:  10809,
			Listen:    "127.0.0.1",
			DNS:       []string{"1.1.1.1", "8.8.8.8"},
		},
		Xray: XraySettings{
			AutoRestart: true,
			LogLevel:    "warning",
			MaxLogSize:  10,
		},
		Health: HealthSettings{
			Timeout:        5,
			AlertOnFailure: true,
			IPCheckURL:     "https://ifconfig.me/ip",
			LatencyHost:    "1.1.1.1:443",
			DNSCheckHost:   "dns.google:443",
		},
		Update: UpdateSettings{
			Channel:      "stable",
			AutoCheck:    true,
			BackupBefore: true,
			XrayVersion:  "v26.3.27",
		},
		UI: UISettings{
			Theme:           "dark",
			RefreshInterval: 5,
			LogLines:        100,
		},
		Notifications: NotificationSettings{
			Enabled: true,
		},
		Backup: BackupSettings{
			MaxFiles: 5,
		},
		AutoSystemProxy: true,
	}
}

// LoadSettings loads application settings from lazyray.yaml.
func LoadSettings() (*Settings, error) {
	path := SettingsPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultSettings(), nil
		}
		return nil, fmt.Errorf("reading settings: %w", err)
	}

	settings := DefaultSettings()
	if err := yaml.Unmarshal(data, settings); err != nil {
		return nil, fmt.Errorf("parsing settings: %w", err)
	}
	return settings, nil
}

// SaveSettings writes application settings to lazyray.yaml.
func SaveSettings(settings *Settings) error {
	if err := EnsureDirs(); err != nil {
		return err
	}

	data, err := yaml.Marshal(settings)
	if err != nil {
		return fmt.Errorf("marshaling settings: %w", err)
	}

	path := SettingsPath()
	if err := fsutil.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("writing settings: %w", err)
	}
	return nil
}

// UpsertSubscription adds a subscription entry, or updates the name of the
// existing entry matched by URL. New entries default to a 24h refresh interval;
// an existing entry's interval is preserved.
func (s *Settings) UpsertSubscription(subURL, subName string) {
	for i := range s.Subscriptions {
		if s.Subscriptions[i].URL == subURL {
			s.Subscriptions[i].Name = subName
			return
		}
	}
	s.Subscriptions = append(s.Subscriptions, SubscriptionEntry{
		Name:     subName,
		URL:      subURL,
		Interval: 24,
	})
}
