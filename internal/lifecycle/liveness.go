package lifecycle

import (
	"net"
	"strconv"
	"time"

	"github.com/rtxnik/lazyray/internal/config"
	"github.com/rtxnik/lazyray/internal/core"
)

// ProbeContextFor builds a core.ProbeContext for probing profileName, marking it
// Active when the running proxy (per state.json) is currently routing it. core
// cannot import lifecycle (import cycle), so this State→ProbeContext bridge lives
// here. Liveness is not re-checked: the functional probe is itself the liveness
// test, so a stale/broken active proxy yields an honest Fail downstream.
func ProbeContextFor(profileName string, settings *config.Settings) core.ProbeContext {
	pc := core.ProbeContext{
		SocksAddr:   net.JoinHostPort(settings.Local.Listen, strconv.Itoa(settings.Local.SocksPort)),
		LatencyHost: settings.Health.LatencyHost,
		Timeout:     3 * time.Second,
	}
	if st, _ := ReadState(); st != nil && st.ActiveProfile == profileName {
		pc.Active = true
		if st.SocksPort > 0 {
			pc.SocksAddr = net.JoinHostPort(settings.Local.Listen, strconv.Itoa(st.SocksPort))
		}
	}
	return pc
}
