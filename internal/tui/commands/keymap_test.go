package commands

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/rtxnik/lazyray/internal/config"
)

// withKeysYAML redirects the config dir to a temp dir (HOME for linux/darwin,
// APPDATA for windows — matching config.ConfigDir) and writes keys.yaml at the
// exact path the loader reads, so a following DefaultKeyMap() picks it up.
// t.Setenv restores the env after the test (and forbids t.Parallel, unused here).
func withKeysYAML(t *testing.T, body string) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("APPDATA", filepath.Join(home, "AppData"))
	path := config.KeysPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// E2f-7: a legacy `health:` entry in keys.yaml must still bind the doctor
// command after the Health->Doctor rename (back-compat alias).
func TestKeysYAMLHealthAliasBindsDoctor(t *testing.T) {
	withKeysYAML(t, "health: H\n")
	km := DefaultKeyMap()
	if !slices.Contains(km.Doctor.Keys(), "H") {
		t.Errorf("legacy `health: H` should bind Doctor; got keys %v", km.Doctor.Keys())
	}
}

// The canonical `doctor:` entry binds the doctor command.
func TestKeysYAMLDoctorBinds(t *testing.T) {
	withKeysYAML(t, "doctor: H\n")
	km := DefaultKeyMap()
	if !slices.Contains(km.Doctor.Keys(), "H") {
		t.Errorf("`doctor: H` should bind Doctor; got keys %v", km.Doctor.Keys())
	}
}

// When both are present, the canonical `doctor:` wins over the `health:` alias.
func TestKeysYAMLDoctorBeatsHealthAlias(t *testing.T) {
	withKeysYAML(t, "doctor: x\nhealth: y\n")
	km := DefaultKeyMap()
	keys := km.Doctor.Keys()
	if !slices.Contains(keys, "x") || slices.Contains(keys, "y") {
		t.Errorf("`doctor:` should win over `health:`; got keys %v", keys)
	}
}
