package config

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// ConfigDir returns the lazyray configuration directory.
func ConfigDir() string {
	if runtime.GOOS == "windows" {
		return filepath.Join(os.Getenv("APPDATA"), "lazyray")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "lazyray")
}

// DataDir returns the lazyray data directory.
func DataDir() string {
	if runtime.GOOS == "windows" {
		return filepath.Join(os.Getenv("LOCALAPPDATA"), "lazyray")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share", "lazyray")
}

// ServersPath returns the path to servers.yaml.
func ServersPath() string {
	return filepath.Join(ConfigDir(), "servers.yaml")
}

// SettingsPath returns the path to lazyray.yaml.
func SettingsPath() string {
	return filepath.Join(ConfigDir(), "lazyray.yaml")
}

// XrayBinaryPath returns the path to the xray binary.
func XrayBinaryPath() string {
	name := "xray"
	if runtime.GOOS == "windows" {
		name = "xray.exe"
	}
	return filepath.Join(DataDir(), name)
}

// XrayConfigPath returns the path to xray config.json.
func XrayConfigPath() string {
	return filepath.Join(DataDir(), "config.json")
}

// LogDir returns the log directory.
func LogDir() string {
	return filepath.Join(DataDir(), "logs")
}

// AccessLogPath returns the path to xray access log.
func AccessLogPath() string {
	return filepath.Join(LogDir(), "access.log")
}

// ErrorLogPath returns the path to xray error log.
func ErrorLogPath() string {
	return filepath.Join(LogDir(), "error.log")
}

// SupervisorLogPath returns the path to the background supervisor's stderr log.
func SupervisorLogPath() string {
	return filepath.Join(LogDir(), "supervisor.log")
}

// LastErrorPath returns the path to the persisted last-startup-failure record.
func LastErrorPath() string {
	return filepath.Join(DataDir(), "last-error.json")
}

// BackupDir returns the backup directory.
func BackupDir() string {
	return filepath.Join(DataDir(), "backup")
}

// PIDFilePath returns the path to the xray PID file.
func PIDFilePath() string {
	return filepath.Join(DataDir(), "xray.pid")
}

// sanitizeProfileName maps a profile name onto a safe filename fragment.
// Lossy: distinct names can collide ("prod/eu" and "prod_eu" both become
// "prod_eu"), which is acceptable for PID files only.
func sanitizeProfileName(name string) string {
	return strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			return r
		}
		return '_'
	}, name)
}

// TunnelPIDPath returns the path to a tunnel PID file for a given profile name.
// The name is sanitized to produce a safe filename.
func TunnelPIDPath(name string) string {
	return filepath.Join(DataDir(), "tunnel-"+sanitizeProfileName(name)+".pid")
}

// TunnelKnownHostsPath returns the path to the derived per-profile known_hosts
// file. Unlike PID files, this is a trust boundary, so the sanitized name is
// suffixed with a short hash of the exact profile name: distinct profiles must
// never share host-key trust even when their sanitized names collide.
func TunnelKnownHostsPath(name string) string {
	sum := sha256.Sum256([]byte(name))
	return filepath.Join(DataDir(),
		"tunnel-"+sanitizeProfileName(name)+"-"+hex.EncodeToString(sum[:4])+".known_hosts")
}

// TunnelPIDGlob returns the glob pattern matching all tunnel PID files.
func TunnelPIDGlob() string {
	return filepath.Join(DataDir(), "tunnel-*.pid")
}

// StatsPath returns the path to the traffic stats history file.
func StatsPath() string {
	return filepath.Join(DataDir(), "stats.json")
}

// KeysPath returns the path to the custom keybindings config.
func KeysPath() string {
	return filepath.Join(ConfigDir(), "keys.yaml")
}

// EnsureDirs creates all required directories.
func EnsureDirs() error {
	dirs := []string{
		ConfigDir(),
		DataDir(),
		LogDir(),
		BackupDir(),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0700); err != nil {
			return err
		}
	}
	return nil
}
