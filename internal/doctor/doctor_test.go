package doctor

import (
	"context"
	"testing"
)

func TestSeverityString(t *testing.T) {
	cases := []struct {
		sev  Severity
		want string
	}{
		{SeverityOK, "OK"},
		{SeverityInfo, "INFO"},
		{SeverityWarn, "WARN"},
		{SeverityFail, "FAIL"},
		{Severity(99), "FAIL"}, // out-of-range defaults to FAIL (most severe)
	}
	for _, c := range cases {
		if got := c.sev.String(); got != c.want {
			t.Errorf("Severity(%d).String() = %q, want %q", c.sev, got, c.want)
		}
	}
}

func TestRunAllOrdersGroupsAndTalliesSummary(t *testing.T) {
	checks := []Check{
		func(ctx context.Context, env *Env) Result {
			return Result{Group: "connectivity", Name: "c1", Severity: SeverityWarn}
		},
		func(ctx context.Context, env *Env) Result {
			return Result{Group: "install", Name: "i1", Severity: SeverityFail}
		},
		func(ctx context.Context, env *Env) Result {
			return Result{Group: "session", Name: "s1", Severity: SeverityInfo}
		},
		func(ctx context.Context, env *Env) Result {
			return Result{Group: "install", Name: "i2", Severity: SeverityOK}
		},
		func(ctx context.Context, env *Env) Result {
			return Result{Group: "config", Name: "cfg1", Severity: SeverityWarn}
		},
	}

	rep := runAll(context.Background(), &Env{}, checks)

	wantOrder := []string{"i1", "i2", "cfg1", "s1", "c1"}
	if len(rep.Checks) != len(wantOrder) {
		t.Fatalf("got %d checks, want %d", len(rep.Checks), len(wantOrder))
	}
	for i, name := range wantOrder {
		if rep.Checks[i].Name != name {
			t.Errorf("Checks[%d].Name = %q, want %q (order=%v)", i, rep.Checks[i].Name, name, names(rep.Checks))
		}
	}

	want := Summary{OK: 1, Info: 1, Warn: 2, Fail: 1}
	if rep.Summary != want {
		t.Errorf("Summary = %+v, want %+v", rep.Summary, want)
	}
}

func TestRunAllUnknownGroupSortsLast(t *testing.T) {
	checks := []Check{
		func(ctx context.Context, env *Env) Result {
			return Result{Group: "zzz-unknown", Name: "u1", Severity: SeverityOK}
		},
		func(ctx context.Context, env *Env) Result {
			return Result{Group: "install", Name: "i1", Severity: SeverityOK}
		},
	}
	rep := runAll(context.Background(), &Env{}, checks)
	if rep.Checks[0].Name != "i1" || rep.Checks[1].Name != "u1" {
		t.Fatalf("unknown group not sorted last: %v", names(rep.Checks))
	}
}

func TestDefaultEnvWiresSeams(t *testing.T) {
	env := DefaultEnv()
	if env == nil {
		t.Fatal("DefaultEnv() returned nil")
	}
	if env.GetXrayVersion == nil || env.CheckXrayVersionCompat == nil ||
		env.CheckProtocolXraySupport == nil || env.ScanXrayPID == nil || env.IsProcessAlive == nil ||
		env.IsOurXray == nil || env.ReadState == nil || env.SupervisorAlive == nil ||
		env.ReadStartupError == nil || env.StatusSnapshot == nil || env.ProxyStatus == nil ||
		env.DesktopEnv == nil || env.LoadServers == nil || env.LoadSettings == nil ||
		env.RunHealthCheck == nil || env.Stat == nil || env.Now == nil {
		t.Error("DefaultEnv() left a seam nil")
	}
	if env.DataDir == "" || env.XrayConfigPath == "" || env.StatePath == "" || env.XrayBinaryPath == "" {
		t.Error("DefaultEnv() left a resolved path empty")
	}
}

func names(rs []Result) []string {
	out := make([]string, len(rs))
	for i, r := range rs {
		out[i] = r.Name
	}
	return out
}
