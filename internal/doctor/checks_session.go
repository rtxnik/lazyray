package doctor

import (
	"context"
	"fmt"
)

// sessionChecks returns the "session" group checks.
func sessionChecks() []Check {
	return []Check{
		checkSupervisor,
		checkXrayProcess,
		checkPortsOpen,
	}
}

func checkSupervisor(_ context.Context, env *Env) Result {
	r := Result{Group: "session", Name: "supervisor"}
	alive := env.SupervisorAlive()
	st, _ := env.ReadState()

	switch {
	case !alive && st == nil:
		r.Severity = SeverityInfo
		r.Detail = "not running (no supervisor, no state) — this is a normal stopped state"
		return r
	case alive && st != nil:
		r.Severity = SeverityOK
		r.Detail = fmt.Sprintf("supervisor live (PID %d), state.json present", st.SupervisorPID)
		return r
	case alive && st == nil:
		r.Severity = SeverityWarn
		r.Detail = "supervisor lock held but no state.json — start may be mid-flight or state was lost"
		r.Hint = "re-run 'lzr status'; if it persists, 'lzr stop' then 'lzr start'"
		return r
	default: // !alive && st != nil
		r.Severity = SeverityWarn
		r.Detail = "state.json present but no live supervisor — stale state from a crash"
		r.Hint = "run 'lzr stop' to clear stale state, then 'lzr start'"
		return r
	}
}

func checkXrayProcess(_ context.Context, env *Env) Result {
	r := Result{Group: "session", Name: "xray process"}
	st, _ := env.ReadState()
	if st == nil {
		r.Severity = SeverityInfo
		r.Detail = "not running; no xray PID to verify"
		return r
	}
	pid := st.XrayPID
	if !env.IsProcessAlive(pid) {
		r.Severity = SeverityFail
		r.Detail = fmt.Sprintf("recorded xray PID %d is not alive", pid)
		r.Hint = "run 'lzr stop' then 'lzr start' to respawn xray"
		return r
	}
	if !env.IsOurXray(pid) {
		r.Severity = SeverityWarn
		r.Detail = fmt.Sprintf("PID %d is alive but does not look like our xray binary", pid)
		r.Hint = "PID may have been recycled by the OS; restart with 'lzr stop && lzr start'"
		return r
	}
	r.Severity = SeverityOK
	r.Detail = fmt.Sprintf("xray PID %d alive and ours", pid)
	return r
}

func checkPortsOpen(_ context.Context, env *Env) Result {
	r := Result{Group: "session", Name: "local ports open"}
	st, _ := env.ReadState()
	if st == nil {
		r.Severity = SeverityInfo
		r.Detail = "not running; ports not expected to be open"
		return r
	}
	snap, err := env.StatusSnapshot()
	if err != nil || snap == nil {
		r.Severity = SeverityWarn
		r.Detail = "could not probe local ports"
		if err != nil {
			r.Detail += ": " + err.Error()
		}
		return r
	}
	if !snap.SocksOK || !snap.HTTPOK {
		r.Severity = SeverityWarn
		r.Detail = fmt.Sprintf("listener down: socksOK=%v httpOK=%v (%s / %s)",
			snap.SocksOK, snap.HTTPOK, snap.SocksAddr, snap.HTTPAddr)
		r.Hint = "xray may still be starting; check 'lzr status' and the error log"
		return r
	}
	r.Severity = SeverityOK
	r.Detail = fmt.Sprintf("SOCKS (%s) and HTTP (%s) both accepting", snap.SocksAddr, snap.HTTPAddr)
	return r
}
