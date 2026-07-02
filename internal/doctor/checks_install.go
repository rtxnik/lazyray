package doctor

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/rtxnik/lazyray/internal/core"
)

// installChecks returns the "install" group checks.
func installChecks() []Check {
	return []Check{
		checkXrayBinary,
		checkXrayVersionFloor,
		checkHysteria2Gate,
		checkGeoFiles,
	}
}

func checkXrayBinary(_ context.Context, env *Env) Result {
	r := Result{Group: "install", Name: "xray binary"}
	if _, err := env.Stat(env.XrayBinaryPath); err != nil {
		r.Severity = SeverityFail
		r.Detail = fmt.Sprintf("xray binary not found at %s", env.XrayBinaryPath)
		r.Hint = "run 'lzr update apply' to install xray-core"
		return r
	}
	r.Severity = SeverityOK
	r.Detail = "present at " + env.XrayBinaryPath
	return r
}

func checkXrayVersionFloor(_ context.Context, env *Env) Result {
	r := Result{Group: "install", Name: "xray version"}
	switch env.GetXrayVersion() {
	case "not installed", "unknown":
		r.Severity = SeverityInfo
		r.Detail = "version not checked (xray binary absent or unreadable)"
		return r
	}
	if warn := env.CheckXrayVersionCompat(); warn != "" {
		r.Severity = SeverityFail
		r.Detail = warn
		r.Hint = "run 'lzr update apply' to upgrade xray-core"
		return r
	}
	r.Severity = SeverityOK
	r.Detail = "meets minimum version " + core.MinXrayVersion
	return r
}

func checkHysteria2Gate(_ context.Context, env *Env) Result {
	r := Result{Group: "install", Name: "hysteria2 support"}
	servers, err := env.LoadServers()
	if err != nil || servers == nil {
		r.Severity = SeverityInfo
		r.Detail = "no profiles loaded; hysteria2 gate not evaluated"
		return r
	}
	profile := servers.DefaultProfile()
	if profile == nil || profile.Server.GetProtocol() != "hysteria2" {
		r.Severity = SeverityInfo
		r.Detail = "no hysteria2 profile configured; gate not applicable"
		return r
	}
	if err := env.CheckProtocolXraySupport("hysteria2"); err != nil {
		r.Severity = SeverityFail
		r.Detail = err.Error()
		r.Hint = "run 'lzr update apply' to upgrade xray-core for hysteria2"
		return r
	}
	r.Severity = SeverityOK
	r.Detail = "xray-core supports the configured hysteria2 profile"
	return r
}

func checkGeoFiles(_ context.Context, env *Env) Result {
	r := Result{Group: "install", Name: "geo data"}
	var missing []string
	for _, name := range []string{"geoip.dat", "geosite.dat"} {
		if _, err := env.Stat(filepath.Join(env.DataDir, name)); err != nil {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		r.Severity = SeverityWarn
		r.Detail = "missing routing data files: " + join(missing)
		r.Hint = "run 'lzr update apply' to (re)extract geoip.dat/geosite.dat"
		return r
	}
	r.Severity = SeverityOK
	r.Detail = "geoip.dat and geosite.dat present in " + env.DataDir
	return r
}

// join concatenates names with ", " without pulling in strings just for this.
func join(parts []string) string {
	out := ""
	for i, p := range parts {
		if i > 0 {
			out += ", "
		}
		out += p
	}
	return out
}
