// Package status assembles the structured proxy status snapshot consumed by
// the `lzr status` command. It is a top-level consumer of config/core/
// lifecycle/platform and is imported only by cmd (no import cycle).
package status

import (
	"fmt"
	"strings"
	"time"

	"github.com/rtxnik/lazyray/internal/config"
	"github.com/rtxnik/lazyray/internal/core"
	"github.com/rtxnik/lazyray/internal/lifecycle"
	"github.com/rtxnik/lazyray/internal/platform"
)

// Snapshot is the structured status of the resident proxy. Its fields and JSON
// tags are identical to the former cmd.StatusOutput so `lzr status --json`
// keeps the exact same wire shape.
type Snapshot struct {
	Running       bool   `json:"running"`
	PID           int    `json:"pid"`
	Owner         string `json:"owner,omitempty"`
	SupervisorPID int    `json:"supervisorPID,omitempty"`
	Uptime        string `json:"uptime"`
	UptimeSeconds int    `json:"uptimeSeconds"`
	SocksOK       bool   `json:"socksOK"`
	HTTPOK        bool   `json:"httpOK"`
	SocksAddr     string `json:"socksAddr"`
	HTTPAddr      string `json:"httpAddr"`
	Profile       string `json:"profile"`
	XrayVersion   string `json:"xrayVersion"`
	ExitIP        string `json:"exitIP,omitempty"`
	PIDFile       string `json:"pidFile"`
	ConfigPath    string `json:"configPath"`
}

// fromState derives the state-dependent fields of a Snapshot from lifecycle
// state. Equivalent to the former cmd.buildStatusOutput.
func fromState(st *lifecycle.State, listen string) Snapshot {
	out := Snapshot{}
	if st != nil {
		out.Running = true
		out.Owner = string(st.Owner)
		out.SupervisorPID = st.SupervisorPID
		out.PID = st.XrayPID
		uptime := time.Since(st.StartedAt)
		out.Uptime = core.FormatUptime(uptime)
		out.UptimeSeconds = int(uptime.Seconds())
		out.SocksAddr = fmt.Sprintf("%s:%d", listen, st.SocksPort)
		out.HTTPAddr = fmt.Sprintf("%s:%d", listen, st.HTTPPort)
	}
	return out
}

// Get self-heals any dangling proxy/state and returns the current status
// snapshot. The Reconcile call is a deliberate side effect preserved from the
// original `lzr status` behavior (not a pure read).
func Get() (*Snapshot, error) {
	// Self-heal any dangling proxy/state from a previous crash.
	_ = lifecycle.Reconcile(platform.CurrentSystemProxy())

	st, _ := lifecycle.ReadState()

	settings, _ := config.LoadSettings()
	if settings == nil {
		settings = config.DefaultSettings()
	}

	servers, _ := config.LoadServers()
	profileName := ""
	if servers != nil {
		if p := servers.DefaultProfile(); p != nil {
			profileName = p.Name
		}
	}

	// Use the xray process for port-open checks (SocksOK/HTTPOK).
	xray := core.NewXrayProcess()
	xrayStatus := xray.Status()

	out := fromState(st, settings.Local.Listen)
	// Preserve prior behavior: always show configured host:port even when stopped.
	if !out.Running {
		out.SocksAddr = fmt.Sprintf("%s:%d", settings.Local.Listen, settings.Local.SocksPort)
		out.HTTPAddr = fmt.Sprintf("%s:%d", settings.Local.Listen, settings.Local.HTTPPort)
	}
	out.SocksOK = xrayStatus.SocksOK
	out.HTTPOK = xrayStatus.HTTPOK
	out.Profile = profileName
	out.XrayVersion = core.GetXrayVersion()
	out.PIDFile = config.PIDFilePath()
	out.ConfigPath = config.XrayConfigPath()

	// Try to get exit IP if proxy is up.
	if out.SocksOK {
		if ip, err := core.GetExitIP(settings); err == nil {
			out.ExitIP = strings.TrimSpace(ip)
		}
	}

	return &out, nil
}
