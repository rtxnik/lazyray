package doctor

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/rtxnik/lazyray/internal/config"
)

// fakeFileInfo is a minimal os.FileInfo for Stat seams.
type fakeFileInfo struct {
	name string
	mode os.FileMode
}

func (f fakeFileInfo) Name() string       { return f.name }
func (f fakeFileInfo) Size() int64        { return 0 }
func (f fakeFileInfo) Mode() os.FileMode  { return f.mode }
func (f fakeFileInfo) ModTime() time.Time { return time.Time{} }
func (f fakeFileInfo) IsDir() bool        { return f.mode.IsDir() }
func (f fakeFileInfo) Sys() any           { return nil }

// statMap builds a Stat seam from a path->info map; missing paths return ErrNotExist.
func statMap(m map[string]os.FileInfo) func(string) (os.FileInfo, error) {
	return func(path string) (os.FileInfo, error) {
		if fi, ok := m[path]; ok {
			return fi, nil
		}
		return nil, fs.ErrNotExist
	}
}

func findResult(rs []Result, name string) (Result, bool) {
	for _, r := range rs {
		if r.Name == name {
			return r, true
		}
	}
	return Result{}, false
}

func TestCheckXrayBinaryPresent(t *testing.T) {
	bin := "/data/xray"
	tests := []struct {
		name    string
		present bool
		want    Severity
	}{
		{"present", true, SeverityOK},
		{"missing", false, SeverityFail},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			files := map[string]os.FileInfo{}
			if tc.present {
				files[bin] = fakeFileInfo{name: "xray", mode: 0755}
			}
			env := &Env{XrayBinaryPath: bin, Stat: statMap(files)}
			got := checkXrayBinary(context.Background(), env)
			if got.Severity != tc.want {
				t.Errorf("severity = %v, want %v (detail=%q)", got.Severity, tc.want, got.Detail)
			}
			if got.Group != "install" {
				t.Errorf("group = %q, want install", got.Group)
			}
		})
	}
}

func TestCheckXrayVersionFloor(t *testing.T) {
	tests := []struct {
		name    string
		version string // GetXrayVersion output
		warn    string // CheckXrayVersionCompat output ("" == OK)
		want    Severity
	}{
		{"ok", "v26.2.6", "", SeverityOK},
		{"outdated", "v1.7.0", "Xray 1.7.0 is outdated (min 1.8.0), press u to update", SeverityFail},
		{"not-installed", "not installed", "", SeverityInfo},
		{"unknown", "unknown", "", SeverityInfo},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			env := &Env{
				GetXrayVersion:         func() string { return tc.version },
				CheckXrayVersionCompat: func() string { return tc.warn },
			}
			got := checkXrayVersionFloor(context.Background(), env)
			if got.Severity != tc.want {
				t.Errorf("severity = %v, want %v", got.Severity, tc.want)
			}
		})
	}
}

func TestCheckHysteria2Gate(t *testing.T) {
	hysProfile := &config.ServersConfig{Profiles: []config.Profile{
		{Name: "h", Default: true, Server: config.ServerConfig{Protocol: "hysteria2"}},
	}}
	vlessProfile := &config.ServersConfig{Profiles: []config.Profile{
		{Name: "v", Default: true, Server: config.ServerConfig{Protocol: "vless"}},
	}}

	tests := []struct {
		name    string
		servers *config.ServersConfig
		gateErr error
		want    Severity
	}{
		{"no-hysteria-profile", vlessProfile, nil, SeverityInfo},
		{"hysteria-ok", hysProfile, nil, SeverityOK},
		{"hysteria-too-old", hysProfile, errors.New("xray 1.8.0 is too old for hysteria2"), SeverityFail},
		{"no-profiles", &config.ServersConfig{}, nil, SeverityInfo},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			env := &Env{
				LoadServers:              func() (*config.ServersConfig, error) { return tc.servers, nil },
				CheckProtocolXraySupport: func(protocol string) error { return tc.gateErr },
			}
			got := checkHysteria2Gate(context.Background(), env)
			if got.Severity != tc.want {
				t.Errorf("severity = %v, want %v (detail=%q)", got.Severity, tc.want, got.Detail)
			}
		})
	}
}

func TestCheckGeoFiles(t *testing.T) {
	dir := "/data"
	geoip := filepath.Join(dir, "geoip.dat")
	geosite := filepath.Join(dir, "geosite.dat")
	tests := []struct {
		name  string
		files map[string]os.FileInfo
		want  Severity
	}{
		{"both-present", map[string]os.FileInfo{
			geoip:   fakeFileInfo{name: "geoip.dat"},
			geosite: fakeFileInfo{name: "geosite.dat"},
		}, SeverityOK},
		{"one-missing", map[string]os.FileInfo{
			geoip: fakeFileInfo{name: "geoip.dat"},
		}, SeverityWarn},
		{"both-missing", map[string]os.FileInfo{}, SeverityWarn},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			env := &Env{DataDir: dir, Stat: statMap(tc.files)}
			got := checkGeoFiles(context.Background(), env)
			if got.Severity != tc.want {
				t.Errorf("severity = %v, want %v (detail=%q)", got.Severity, tc.want, got.Detail)
			}
			// findResult is a shared helper used by sibling test files (Tasks 8-9);
			// exercise it here so the compiler and linter agree it is reachable.
			if r, ok := findResult([]Result{got}, got.Name); ok && r.Group != "install" {
				t.Errorf("findResult: group = %q, want install", r.Group)
			}
		})
	}
}
