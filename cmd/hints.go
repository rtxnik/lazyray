package cmd

import "github.com/rtxnik/lazyray/internal/clihint"

// errNoProfilesConfigured is returned when an operation needs a proxy profile
// but the profile store is empty.
func errNoProfilesConfigured() error {
	return clihint.Errorf("import a profile with 'lzr import <url>'", "no profiles configured")
}

// errProfileNotFound is returned when a named proxy profile is not in the store.
func errProfileNotFound(name string) error {
	return clihint.Errorf("list profiles with 'lzr config list'", "profile %q not found", name)
}

// errNoDefaultProfile is returned when no proxy profile is marked default.
func errNoDefaultProfile() error {
	return clihint.Errorf("pick one with 'lzr config switch <name>'", "no default profile")
}
