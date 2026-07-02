package doctor

import (
	"context"
	"fmt"
	"runtime"

	"github.com/rtxnik/lazyray/internal/config"
)

// configChecks returns the "config" group checks.
func configChecks() []Check {
	return []Check{
		checkServersParse,
		checkPorts,
		checkFilePerms,
	}
}

func checkServersParse(_ context.Context, env *Env) Result {
	r := Result{Group: "config", Name: "servers config"}
	servers, err := env.LoadServers()
	if err != nil {
		r.Severity = SeverityFail
		r.Detail = "servers.yaml failed to parse: " + err.Error()
		r.Hint = "fix or recreate servers.yaml (see 'lzr import')"
		return r
	}
	profile := servers.DefaultProfile()
	if profile == nil {
		r.Severity = SeverityFail
		r.Detail = "no default profile configured"
		r.Hint = "add a profile with 'lzr import <vless://...>'"
		return r
	}
	r.Severity = SeverityOK
	r.Detail = fmt.Sprintf("default profile %q is valid", profile.Name)
	return r
}

func checkPorts(_ context.Context, env *Env) Result {
	r := Result{Group: "config", Name: "local ports"}
	settings, err := env.LoadSettings()
	if err != nil || settings == nil {
		settings = config.DefaultSettings()
	}
	socks := settings.Local.SocksPort
	http := settings.Local.HTTPPort

	inRange := func(p int) bool { return p >= 1 && p <= 65535 }
	if !inRange(socks) || !inRange(http) {
		r.Severity = SeverityFail
		r.Detail = fmt.Sprintf("port out of range: socks=%d http=%d (must be 1-65535)", socks, http)
		r.Hint = "set local.socksPort/local.httpPort in lazyray.yaml"
		return r
	}
	if socks == http {
		r.Severity = SeverityFail
		r.Detail = fmt.Sprintf("SOCKS and HTTP ports are identical (%d)", socks)
		r.Hint = "give local.socksPort and local.httpPort distinct values"
		return r
	}
	if socks < 1024 || http < 1024 {
		r.Severity = SeverityWarn
		r.Detail = fmt.Sprintf("privileged port in use: socks=%d http=%d (<1024 may need elevated rights)", socks, http)
		r.Hint = "prefer ports >= 1024 for an unprivileged proxy"
		return r
	}
	r.Severity = SeverityOK
	r.Detail = fmt.Sprintf("socks=%d http=%d in range and distinct", socks, http)
	return r
}

func checkFilePerms(_ context.Context, env *Env) Result {
	r := Result{Group: "config", Name: "file permissions"}
	if runtime.GOOS == "windows" {
		r.Severity = SeverityInfo
		r.Detail = "POSIX permission checks not applicable on Windows"
		return r
	}
	var loose []string
	for _, path := range []string{env.StatePath, env.XrayConfigPath, env.ServersPath} {
		fi, err := env.Stat(path)
		if err != nil {
			continue // absent file is fine; presence checks live in other groups
		}
		if fi.Mode().Perm()&0o077 != 0 {
			loose = append(loose, fmt.Sprintf("%s (%o)", path, fi.Mode().Perm()))
		}
	}
	if len(loose) > 0 {
		r.Severity = SeverityWarn
		r.Detail = "world/group-readable sensitive files: " + join(loose)
		r.Hint = "chmod 600 the listed files"
		return r
	}
	r.Severity = SeverityOK
	r.Detail = "state.json/config.json/servers.yaml are 0600 (or absent)"
	return r
}
