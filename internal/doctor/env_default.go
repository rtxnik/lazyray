package doctor

import (
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/rtxnik/lazyray/internal/config"
	"github.com/rtxnik/lazyray/internal/core"
	"github.com/rtxnik/lazyray/internal/lifecycle"
	"github.com/rtxnik/lazyray/internal/platform"
	"github.com/rtxnik/lazyray/internal/status"
)

// DefaultEnv wires the real host implementations behind every seam.
func DefaultEnv() *Env {
	return &Env{
		XrayBinaryPath: config.XrayBinaryPath(),
		DataDir:        config.DataDir(),
		XrayConfigPath: config.XrayConfigPath(),
		StatePath:      lifecycle.StatePath(),
		ServersPath:    config.ServersPath(),
		SettingsPath:   config.SettingsPath(),
		StatsPath:      config.StatsPath(),
		BackupDir:      config.BackupDir(),
		LogDir:         config.LogDir(),
		ConfigDir:      config.ConfigDir(),

		GetXrayVersion:           core.GetXrayVersion,
		CheckXrayVersionCompat:   core.CheckXrayVersionCompat,
		CheckProtocolXraySupport: core.CheckProtocolXraySupport,

		ScanXrayPID:    core.ScanXrayPID,
		IsProcessAlive: core.IsProcessAlive,
		IsOurXray:      lifecycle.IsOurXray,

		ReadState:        lifecycle.ReadState,
		SupervisorAlive:  lifecycle.SupervisorAlive,
		ReadStartupError: lifecycle.ReadStartupError,

		StatusSnapshot: status.Get,

		ProxyStatus: func() (*platform.ProxyStatus, error) {
			return platform.CurrentSystemProxy().Status()
		},
		DesktopEnv: detectDesktopEnv,

		LoadServers:  config.LoadServers,
		LoadSettings: config.LoadSettings,

		RunHealthCheck: defaultRunHealthCheck,

		Stat: os.Stat,
		Now:  time.Now,

		GOOS: runtime.GOOS,
	}
}

// detectDesktopEnv mirrors platform's linux desktopEnv() classification but is
// build-tag-free so doctor compiles on every OS. On non-linux hosts XDG vars are
// typically unset, yielding "" — which the routing check treats as headless and
// downgrades to INFO rather than asserting a problem.
func detectDesktopEnv() string {
	de := strings.ToLower(os.Getenv("XDG_CURRENT_DESKTOP"))
	switch {
	case strings.Contains(de, "gnome"), strings.Contains(de, "unity"), strings.Contains(de, "cinnamon"):
		return "gnome"
	case strings.Contains(de, "kde"):
		return "kde"
	default:
		return ""
	}
}

// defaultRunHealthCheck builds the real XrayProcess + profile + settings and
// runs core's full health check. On any load error (or no default profile) it
// returns nil so the connectivity check degrades gracefully.
func defaultRunHealthCheck() *core.HealthReport {
	settings, err := config.LoadSettings()
	if err != nil {
		settings = config.DefaultSettings()
	}
	servers, err := config.LoadServers()
	if err != nil {
		return nil
	}
	profile := servers.DefaultProfile()
	if profile == nil {
		return nil
	}
	return core.RunHealthCheck(core.NewXrayProcess(), profile, settings)
}
