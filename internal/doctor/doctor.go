// Package doctor aggregates installation, configuration, runtime, routing, and
// connectivity diagnostics into a single severity-graded report. The package
// never prints and never calls os.Exit; rendering and exit-code derivation live
// in cmd/doctor.go. Every check is a pure function over an injected Env, so the
// whole surface is unit-testable without touching the real host.
package doctor

import (
	"context"
	"os"
	"sort"
	"time"

	"github.com/rtxnik/lazyray/internal/config"
	"github.com/rtxnik/lazyray/internal/core"
	"github.com/rtxnik/lazyray/internal/lifecycle"
	"github.com/rtxnik/lazyray/internal/platform"
	"github.com/rtxnik/lazyray/internal/status"
)

// Severity grades a single check result.
type Severity int

const (
	// SeverityOK is a passing check.
	SeverityOK Severity = iota
	// SeverityInfo is a neutral, non-failing observation (e.g. "not running").
	SeverityInfo
	// SeverityWarn is a degraded-but-tolerable condition.
	SeverityWarn
	// SeverityFail is a blocking problem.
	SeverityFail
)

// String renders a severity as a fixed-width-friendly label. Out-of-range
// values render as the most severe label so an unexpected value never hides
// behind an OK.
func (s Severity) String() string {
	switch s {
	case SeverityOK:
		return "OK"
	case SeverityInfo:
		return "INFO"
	case SeverityWarn:
		return "WARN"
	default:
		return "FAIL"
	}
}

// Result is the outcome of one check.
type Result struct {
	Group    string   `json:"group"`
	Name     string   `json:"name"`
	Severity Severity `json:"severity"`
	Detail   string   `json:"detail"`
	Hint     string   `json:"hint,omitempty"`
}

// Summary tallies results by severity.
type Summary struct {
	OK   int `json:"ok"`
	Info int `json:"info"`
	Warn int `json:"warn"`
	Fail int `json:"fail"`
}

// Report is the full doctor output.
type Report struct {
	Checks  []Result `json:"checks"`
	Summary Summary  `json:"summary"`
}

// Check is a single diagnostic. It must be pure with respect to env: all host
// access goes through the injected seams.
type Check func(ctx context.Context, env *Env) Result

// Env carries already-resolved paths plus pluggable function-value seams so
// every check is unit-testable with a fake. DefaultEnv wires the real
// implementations.
type Env struct {
	// Resolved path strings.
	XrayBinaryPath string
	DataDir        string
	XrayConfigPath string
	StatePath      string
	ServersPath    string

	// Install / version seams.
	GetXrayVersion           func() string
	CheckXrayVersionCompat   func() string
	CheckProtocolXraySupport func(protocol string) error

	// Process / identity seams.
	ScanXrayPID    func() int
	IsProcessAlive func(pid int) bool
	IsOurXray      func(pid int) bool

	// Lifecycle / state seams.
	ReadState        func() (*lifecycle.State, error)
	SupervisorAlive  func() bool
	ReadStartupError func() (*lifecycle.StartupError, error)

	// Session snapshot.
	StatusSnapshot func() (*status.Snapshot, error)

	// Routing seams.
	ProxyStatus func() (*platform.ProxyStatus, error)
	DesktopEnv  func() string

	// Config seams.
	LoadServers  func() (*config.ServersConfig, error)
	LoadSettings func() (*config.Settings, error)

	// Connectivity seam.
	RunHealthCheck func() *core.HealthReport

	// Generic host seams.
	Stat func(path string) (os.FileInfo, error)
	Now  func() time.Time

	// GOOS is the target OS (runtime.GOOS); injectable so the linux-only
	// headless heuristic is unit-testable on any runner.
	GOOS string
}

// groupOrder is the deterministic section order for RunAll.
var groupOrder = []string{
	"install",
	"config",
	"session",
	"routing",
	"foreign",
	"startup",
	"connectivity",
}

// groupRank returns the sort key for a group; unknown groups sort last but
// stay mutually stable.
func groupRank(group string) int {
	for i, g := range groupOrder {
		if g == group {
			return i
		}
	}
	return len(groupOrder)
}

// Registry returns all v1 checks. Order within a group follows this slice;
// across groups RunAll re-sorts into groupOrder.
func Registry() []Check {
	return registryChecks()
}

// RunAll executes the default registry in deterministic group order and fills
// the Summary.
func RunAll(ctx context.Context, env *Env) Report {
	return runAll(ctx, env, Registry())
}

// runAll is the testable core: run every check, then stable-sort the results by
// group rank (preserving registration order within a group) and tally Summary.
func runAll(ctx context.Context, env *Env, checks []Check) Report {
	results := make([]Result, 0, len(checks))
	for _, c := range checks {
		results = append(results, c(ctx, env))
	}
	sort.SliceStable(results, func(i, j int) bool {
		return groupRank(results[i].Group) < groupRank(results[j].Group)
	})

	rep := Report{Checks: results}
	for _, r := range results {
		switch r.Severity {
		case SeverityOK:
			rep.Summary.OK++
		case SeverityInfo:
			rep.Summary.Info++
		case SeverityWarn:
			rep.Summary.Warn++
		default:
			rep.Summary.Fail++
		}
	}
	return rep
}

// registryChecks aggregates all check constructors. Tasks 7-9 replace each stub.
func registryChecks() []Check {
	var checks []Check
	checks = append(checks, installChecks()...)
	checks = append(checks, configChecks()...)
	checks = append(checks, sessionChecks()...)
	checks = append(checks, routingChecks()...)
	checks = append(checks, foreignChecks()...)
	checks = append(checks, startupChecks()...)
	checks = append(checks, connectivityChecks()...)
	return checks
}
