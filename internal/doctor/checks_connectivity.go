package doctor

import (
	"context"
	"fmt"
	"strings"

	"github.com/rtxnik/lazyray/internal/core"
)

// connectivityChecks returns the "connectivity" group check. It wraps core's
// full health check into a single doctor Result: OK when every sub-check
// passed, WARN when any failed, INFO when the session is stopped.
func connectivityChecks() []Check {
	return []Check{checkConnectivity}
}

func checkConnectivity(_ context.Context, env *Env) Result {
	r := Result{Group: "connectivity", Name: "tunnel health"}

	st, _ := env.ReadState()
	if st == nil {
		r.Severity = SeverityInfo
		r.Detail = "not running; connectivity checks skipped"
		return r
	}

	report := env.RunHealthCheck()
	if report == nil {
		r.Severity = SeverityWarn
		r.Detail = "health check could not run (config or xray process unavailable)"
		r.Hint = "verify 'lzr status' shows a running session, then 'lzr health'"
		return r
	}

	if report.AllPassed {
		r.Severity = SeverityOK
		r.Detail = connectivitySummary(report.Checks)
		return r
	}

	var failed []string
	for _, c := range report.Checks {
		if !c.OK {
			failed = append(failed, fmt.Sprintf("%s (%s)", c.Name, c.Detail))
		}
	}
	r.Severity = SeverityWarn
	r.Detail = "failing checks: " + strings.Join(failed, "; ")
	r.Hint = "run 'lzr health' for the full per-check breakdown"
	return r
}

// connectivitySummary renders a one-line digest of the passing checks (exit IP
// and latency are the headline values when present).
func connectivitySummary(checks []core.CheckResult) string {
	var exitIP, latency string
	for _, c := range checks {
		switch c.Name {
		case "Exit IP":
			exitIP = c.Detail
		case "Latency":
			latency = c.Detail
		}
	}
	switch {
	case exitIP != "" && latency != "":
		return fmt.Sprintf("all checks passed (exit IP %s, latency %s)", exitIP, latency)
	case exitIP != "":
		return fmt.Sprintf("all checks passed (exit IP %s)", exitIP)
	default:
		return "all connectivity checks passed"
	}
}
