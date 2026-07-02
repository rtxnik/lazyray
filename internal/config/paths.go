package config

import (
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

// TunnelPIDPath returns the path to a tunnel PID file for a given profile name.
// The name is sanitized to produce a safe filename.
func TunnelPIDPath(name string) string {
	safe := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			return r
		}
		return '_'
	}, name)
	return filepath.Join(DataDir(), "tunnel-"+safe+".pid")
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
		if err := os.MkdirAll(d, 0755); err != nil {
			return err
		}
	}
	return nil
}
