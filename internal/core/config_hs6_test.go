package core

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/rtxnik/lazyray/internal/config"
)

func TestGenerateConfigAccessLogDefaultOff(t *testing.T) {
	s := config.DefaultSettings()
	cfg, err := GenerateXrayConfig(testProfile(), s)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Log.Access != "none" {
		t.Fatalf("default access = %q, want none", cfg.Log.Access)
	}

	s.Xray.AccessLog = "file"
	cfg, err = GenerateXrayConfig(testProfile(), s)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Log.Access == "none" || cfg.Log.Access == "" {
		t.Fatalf("enabled access = %q, want a path", cfg.Log.Access)
	}
}

func TestEnforceLogPolicyReconcilesAndTightens(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := config.EnsureDirs(); err != nil {
		t.Fatal(err)
	}
	// A pre-HS-6 config with access logging on, plus a loose archived log.
	old := &XrayConfig{Log: XrayLog{LogLevel: "warning", Access: config.AccessLogPath(), Error: config.ErrorLogPath()}}
	data, err := json.MarshalIndent(old, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(config.XrayConfigPath(), data, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(config.AccessLogPath()+".1", []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := enforceLogPolicy(config.DefaultSettings()); err != nil {
		t.Fatal(err)
	}

	raw, err := os.ReadFile(config.XrayConfigPath())
	if err != nil {
		t.Fatal(err)
	}
	var got XrayConfig
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	if got.Log.Access != "none" {
		t.Fatalf("on-disk access not reconciled to off: %q", got.Log.Access)
	}
	fi, err := os.Stat(config.AccessLogPath() + ".1")
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode().Perm()&0o077 != 0 {
		t.Fatalf("archived log still loose: %o", fi.Mode().Perm())
	}
}

func TestPrepareLogsForStartTightensAndDefaultsOff(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := config.EnsureDirs(); err != nil {
		t.Fatal(err)
	}
	// A loose error log left by a prior run.
	if err := os.WriteFile(config.ErrorLogPath(), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	PrepareLogsForStart(config.DefaultSettings()) // access log off by default

	fi, err := os.Stat(config.ErrorLogPath())
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode().Perm()&0o077 != 0 {
		t.Fatalf("error log not tightened to 0600: %o", fi.Mode().Perm())
	}
	if _, err := os.Stat(config.AccessLogPath()); err == nil {
		t.Fatal("access log created though it is off by default")
	}
}
