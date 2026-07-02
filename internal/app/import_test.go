package app

import (
	"errors"
	"testing"

	"github.com/rtxnik/lazyray/internal/config"
)

func profile(name, uuid string) *config.Profile {
	return &config.Profile{
		Name: name,
		Server: config.ServerConfig{
			Address:   "1.2.3.4",
			Port:      443,
			UUID:      uuid,
			Transport: config.TransportConfig{Network: "tcp"},
			Security:  config.SecurityConfig{Type: "none"},
		},
	}
}

func TestImportProfile_FirstBecomesDefaultAndSaves(t *testing.T) {
	var saved *config.ServersConfig
	svc := &Service{saveServers: func(c *config.ServersConfig) error { saved = c; return nil }}

	servers := &config.ServersConfig{}
	got, err := svc.ImportProfile(servers, profile("p1", "uuid-1"), false)
	if err != nil {
		t.Fatalf("ImportProfile() error = %v", err)
	}
	if !got.Default {
		t.Error("first imported profile should be Default")
	}
	if len(servers.Profiles) != 1 {
		t.Fatalf("len(Profiles) = %d, want 1", len(servers.Profiles))
	}
	if saved != servers {
		t.Error("saveServers should be called with the mutated servers")
	}
}

func TestImportProfile_NonFirstNotDefault(t *testing.T) {
	svc := &Service{saveServers: func(*config.ServersConfig) error { return nil }}
	servers := &config.ServersConfig{Profiles: []config.Profile{*profile("p0", "uuid-0")}}

	got, err := svc.ImportProfile(servers, profile("p1", "uuid-1"), false)
	if err != nil {
		t.Fatalf("ImportProfile() error = %v", err)
	}
	if got.Default {
		t.Error("non-first imported profile must not be Default")
	}
	if len(servers.Profiles) != 2 {
		t.Fatalf("len(Profiles) = %d, want 2", len(servers.Profiles))
	}
}

func TestImportProfile_DuplicateUUID_NoForce(t *testing.T) {
	called := false
	svc := &Service{saveServers: func(*config.ServersConfig) error { called = true; return nil }}
	servers := &config.ServersConfig{Profiles: []config.Profile{*profile("orig", "dup")}}

	_, err := svc.ImportProfile(servers, profile("new", "dup"), false)
	var dup *DuplicateUUIDError
	if !errors.As(err, &dup) {
		t.Fatalf("err = %v, want *DuplicateUUIDError", err)
	}
	if dup.ExistingName != "orig" {
		t.Errorf("ExistingName = %q, want %q", dup.ExistingName, "orig")
	}
	if dup.Error() != `UUID already used by profile "orig"` {
		t.Errorf("Error() = %q", dup.Error())
	}
	if called {
		t.Error("saveServers must not run on a rejected duplicate")
	}
	if len(servers.Profiles) != 1 {
		t.Errorf("duplicate must not be appended")
	}
}

func TestImportProfile_DuplicateUUID_Force(t *testing.T) {
	svc := &Service{saveServers: func(*config.ServersConfig) error { return nil }}
	servers := &config.ServersConfig{Profiles: []config.Profile{*profile("orig", "dup")}}

	if _, err := svc.ImportProfile(servers, profile("new", "dup"), true); err != nil {
		t.Fatalf("force import error = %v", err)
	}
	if len(servers.Profiles) != 2 {
		t.Errorf("force import should append despite duplicate")
	}
}

func TestImportProfile_SaveErrorPropagatesRaw(t *testing.T) {
	sentinel := errors.New("disk full")
	svc := &Service{saveServers: func(*config.ServersConfig) error { return sentinel }}
	_, err := svc.ImportProfile(&config.ServersConfig{}, profile("p1", "u1"), false)
	if !errors.Is(err, sentinel) {
		t.Fatalf("err = %v, want raw sentinel (shell wraps)", err)
	}
}

func TestImportSubscription_MergeCounts_DefaultGuardIsNoop(t *testing.T) {
	// fake merge mimics core.ImportSubscription: appends profiles (Default=false), returns counts.
	svc := &Service{importSubscription: func(_, _ string, servers *config.ServersConfig) (int, int, error) {
		servers.Profiles = append(servers.Profiles, *profile("s1", "su1"), *profile("s2", "su2"))
		return 2, 0, nil
	}}
	servers := &config.ServersConfig{}
	added, updated, err := svc.ImportSubscription(servers, "https://x/sub", "sub")
	if err != nil {
		t.Fatalf("ImportSubscription() error = %v", err)
	}
	if added != 2 || updated != 0 {
		t.Fatalf("added,updated = %d,%d want 2,0", added, updated)
	}
	// DefaultProfile()==nil only when empty, so the guard never flags a Default: preserve that.
	for _, p := range servers.Profiles {
		if p.Default {
			t.Error("ImportSubscription must not set Default (guard is a preserved no-op)")
		}
	}
}

func TestImportSubscription_FetchErrorPropagates(t *testing.T) {
	sentinel := errors.New("http 500")
	svc := &Service{importSubscription: func(_, _ string, _ *config.ServersConfig) (int, int, error) {
		return 0, 0, sentinel
	}}
	added, updated, err := svc.ImportSubscription(&config.ServersConfig{}, "u", "n")
	if !errors.Is(err, sentinel) {
		t.Fatalf("err = %v, want raw sentinel", err)
	}
	if added != 0 || updated != 0 {
		t.Errorf("added,updated = %d,%d want 0,0", added, updated)
	}
}
