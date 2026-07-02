package doctor

import (
	"context"
	"fmt"
)

// routingChecks returns the "routing" group checks.
func routingChecks() []Check {
	return []Check{
		checkProxyDesync,
		checkHeadless,
	}
}

// foreignChecks returns the "foreign" group checks.
func foreignChecks() []Check {
	return []Check{checkForeignXray}
}

// startupChecks returns the "startup" group checks.
func startupChecks() []Check {
	return []Check{checkStartupError}
}

func checkProxyDesync(_ context.Context, env *Env) Result {
	r := Result{Group: "routing", Name: "os proxy vs state"}
	st, _ := env.ReadState()
	if st == nil {
		r.Severity = SeverityInfo
		r.Detail = "not running; OS proxy desync not applicable"
		return r
	}
	osProxy, err := env.ProxyStatus()
	if err != nil || osProxy == nil {
		r.Severity = SeverityWarn
		r.Detail = "could not read OS proxy state"
		if err != nil {
			r.Detail += ": " + err.Error()
		}
		return r
	}
	stateWants := st.Routing.SystemProxy
	osHas := osProxy.HTTPEnabled || osProxy.SOCKSEnabled || osProxy.PACEnabled
	if stateWants != osHas {
		r.Severity = SeverityWarn
		r.Detail = fmt.Sprintf("desync: state expects systemProxy=%v but OS proxy is %v", stateWants, osHas)
		r.Hint = "run 'lzr stop' then 'lzr start' to re-apply routing, or fix the OS proxy manually"
		return r
	}
	r.Severity = SeverityOK
	r.Detail = fmt.Sprintf("OS proxy (%v) matches state.Routing.SystemProxy (%v)", osHas, stateWants)
	return r
}

func checkHeadless(_ context.Context, env *Env) Result {
	r := Result{Group: "routing", Name: "desktop environment"}
	st, _ := env.ReadState()
	if st == nil {
		r.Severity = SeverityInfo
		r.Detail = "not running; desktop-environment check not applicable"
		return r
	}
	if env.GOOS != "linux" {
		r.Severity = SeverityOK
		r.Detail = "OS proxy is configured via native APIs on " + env.GOOS
		return r
	}
	if env.DesktopEnv() != "" {
		r.Severity = SeverityOK
		r.Detail = "supported desktop environment detected for OS proxy"
		return r
	}
	// Headless host.
	if st.Routing.SystemProxy {
		r.Severity = SeverityWarn
		r.Detail = "headless host (no GNOME/KDE) but state records an active OS proxy"
		r.Hint = "OS proxy cannot be applied here; export http_proxy/https_proxy manually or run with --no-proxy"
		return r
	}
	r.Severity = SeverityInfo
	r.Detail = "headless host; running without OS proxy (SOCKS/HTTP still available)"
	r.Hint = "set http_proxy=http://<listen>:<httpPort> to route shell tools through the tunnel"
	return r
}

func checkForeignXray(_ context.Context, env *Env) Result {
	r := Result{Group: "foreign", Name: "foreign xray"}
	scanned := env.ScanXrayPID()
	if scanned <= 0 {
		r.Severity = SeverityOK
		r.Detail = "no stray xray process detected"
		return r
	}
	st, _ := env.ReadState()
	ourPID := 0
	if st != nil {
		ourPID = st.XrayPID
	}
	if scanned == ourPID && env.IsOurXray(scanned) {
		r.Severity = SeverityOK
		r.Detail = fmt.Sprintf("the only xray running is ours (PID %d)", scanned)
		return r
	}
	if env.IsOurXray(scanned) {
		// Alive, ours, but not the PID state expects — surface as a foreign-ish drift.
		r.Severity = SeverityWarn
		r.Detail = fmt.Sprintf("an xray (PID %d) matches our binary but is not the PID in state.json (%d)", scanned, ourPID)
		r.Hint = "diagnostics only — never auto-killed; reconcile with 'lzr stop' then 'lzr start'"
		return r
	}
	r.Severity = SeverityWarn
	r.Detail = fmt.Sprintf("a foreign xray (PID %d) is running and may hold proxy ports", scanned)
	r.Hint = "diagnostics only — lazyray never kills foreign processes; stop it manually if it conflicts"
	return r
}

func checkStartupError(_ context.Context, env *Env) Result {
	r := Result{Group: "startup", Name: "last supervisor start"}
	se, err := env.ReadStartupError()
	if err != nil {
		r.Severity = SeverityWarn
		r.Detail = "could not read startup-error record: " + err.Error()
		return r
	}
	if se == nil {
		r.Severity = SeverityOK
		r.Detail = "no recorded supervisor startup failure"
		return r
	}
	r.Severity = SeverityFail
	r.Detail = fmt.Sprintf("last supervisor start failed at stage %q: %s (%s)",
		se.Stage, se.Message, se.Time.Format("2006-01-02 15:04:05"))
	r.Hint = "inspect supervisor.log for full output; fix the cause then 'lzr start'"
	return r
}
