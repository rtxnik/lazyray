package config

import (
	"os"
	"testing"
)

func TestServersConfig_DefaultProfile(t *testing.T) {
	tests := []struct {
		name     string
		profiles []Profile
		wantName string
		wantNil  bool
	}{
		{
			name:    "empty profiles",
			wantNil: true,
		},
		{
			name: "returns default profile",
			profiles: []Profile{
				{Name: "first"},
				{Name: "second", Default: true},
			},
			wantName: "second",
		},
		{
			name: "returns first if no default",
			profiles: []Profile{
				{Name: "first"},
				{Name: "second"},
			},
			wantName: "first",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &ServersConfig{Profiles: tc.profiles}
			p := cfg.DefaultProfile()
			if tc.wantNil {
				if p != nil {
					t.Errorf("expected nil, got %v", p)
				}
				return
			}
			if p == nil {
				t.Fatal("expected non-nil profile")
			}
			if p.Name != tc.wantName {
				t.Errorf("Name = %q, want %q", p.Name, tc.wantName)
			}
		})
	}
}

func TestServersConfig_SetDefault(t *testing.T) {
	cfg := &ServersConfig{
		Profiles: []Profile{
			{Name: "a", Default: true},
			{Name: "b"},
			{Name: "c"},
		},
	}

	if err := cfg.SetDefault(2); err != nil {
		t.Fatalf("SetDefault() error = %v", err)
	}

	for i, p := range cfg.Profiles {
		if i == 2 && !p.Default {
			t.Errorf("Profile[2] should be default")
		}
		if i != 2 && p.Default {
			t.Errorf("Profile[%d] should not be default", i)
		}
	}
}

func TestServersConfig_SetDefault_OutOfRange(t *testing.T) {
	cfg := &ServersConfig{
		Profiles: []Profile{{Name: "a"}},
	}

	if err := cfg.SetDefault(-1); err == nil {
		t.Error("SetDefault(-1) should return error")
	}
	if err := cfg.SetDefault(5); err == nil {
		t.Error("SetDefault(5) should return error")
	}
}

func TestServersConfig_HasUUID(t *testing.T) {
	cfg := &ServersConfig{
		Profiles: []Profile{
			{Name: "first", Server: ServerConfig{UUID: "uuid-1"}},
			{Name: "second", Server: ServerConfig{UUID: "uuid-2"}},
		},
	}

	name, exists := cfg.HasUUID("uuid-2")
	if !exists {
		t.Error("expected uuid-2 to exist")
	}
	if name != "second" {
		t.Errorf("Name = %q, want %q", name, "second")
	}

	_, exists = cfg.HasUUID("uuid-3")
	if exists {
		t.Error("expected uuid-3 to not exist")
	}
}

func TestServersConfig_HasUUID_Empty(t *testing.T) {
	cfg := &ServersConfig{}
	_, exists := cfg.HasUUID("any-uuid")
	if exists {
		t.Error("expected no match in empty config")
	}
}

func TestCurrentConfigVersion(t *testing.T) {
	if CurrentConfigVersion != 2 {
		t.Errorf("CurrentConfigVersion = %d, want 2", CurrentConfigVersion)
	}
}

func TestMigrateServerConfig_VLESSDefaults(t *testing.T) {
	s := &ServerConfig{
		Address: "1.2.3.4",
		Port:    443,
		UUID:    "test-uuid",
	}
	migrateServerConfig(s)

	if s.Protocol != "vless" {
		t.Errorf("Protocol = %q, want %q", s.Protocol, "vless")
	}
	if s.Encryption != "none" {
		t.Errorf("Encryption = %q, want %q", s.Encryption, "none")
	}
	if s.Transport.Network != "tcp" {
		t.Errorf("Transport.Network = %q, want %q", s.Transport.Network, "tcp")
	}
}

func TestMigrateServerConfig_VMESSDefaults(t *testing.T) {
	s := &ServerConfig{
		Address:  "1.2.3.4",
		Port:     443,
		UUID:     "test-uuid",
		Protocol: "vmess",
	}
	migrateServerConfig(s)

	if s.Protocol != "vmess" {
		t.Errorf("Protocol = %q, want %q", s.Protocol, "vmess")
	}
	if s.Encryption != "auto" {
		t.Errorf("Encryption = %q, want %q", s.Encryption, "auto")
	}
}

func TestMigrateServerConfig_PreservesExisting(t *testing.T) {
	s := &ServerConfig{
		Address:    "1.2.3.4",
		Port:       443,
		UUID:       "test-uuid",
		Protocol:   "trojan",
		Encryption: "custom",
		Transport:  TransportConfig{Network: "xhttp"},
	}
	migrateServerConfig(s)

	if s.Protocol != "trojan" {
		t.Errorf("Protocol should be preserved, got %q", s.Protocol)
	}
	if s.Encryption != "custom" {
		t.Errorf("Encryption should be preserved, got %q", s.Encryption)
	}
	if s.Transport.Network != "xhttp" {
		t.Errorf("Transport.Network should be preserved, got %q", s.Transport.Network)
	}
}

func TestMigrateProfiles(t *testing.T) {
	profiles := []Profile{
		{
			Name:   "test1",
			Server: ServerConfig{Address: "1.2.3.4", Port: 443, UUID: "uuid1"},
			Chain: []ServerConfig{
				{Address: "5.6.7.8", Port: 443, UUID: "uuid2"},
			},
		},
	}
	migrateProfiles(profiles)

	if profiles[0].Server.Protocol != "vless" {
		t.Error("server protocol should be migrated")
	}
	if profiles[0].Chain[0].Protocol != "vless" {
		t.Error("chain server protocol should be migrated")
	}
}

// A second migration attempt (e.g. after an earlier interrupted save) must NEVER
// re-clobber the original pre-migration backup — that backup is the only copy of
// the user's true original data.
func TestLoadServers_MigrationBackupWriteOnce(t *testing.T) {
	cleanup := setupTestHome(t)
	defer cleanup()

	if err := EnsureDirs(); err != nil {
		t.Fatalf("EnsureDirs: %v", err)
	}
	// A version-1 servers.yaml with one profile triggers migration to v2.
	v1 := []byte("configVersion: 1\nprofiles:\n  - name: legacy\n    server:\n      address: 1.2.3.4\n      port: 443\n")
	if err := os.WriteFile(ServersPath(), v1, 0o600); err != nil {
		t.Fatalf("seed servers.yaml: %v", err)
	}
	// Pre-existing backup that must be preserved byte-for-byte.
	backupPath := ServersPath() + ".v1.bak"
	sentinel := []byte("ORIGINAL-PRESERVED-DO-NOT-CLOBBER")
	if err := os.WriteFile(backupPath, sentinel, 0o600); err != nil {
		t.Fatalf("seed backup: %v", err)
	}

	if _, err := LoadServers(); err != nil {
		t.Fatalf("LoadServers: %v", err)
	}

	got, err := os.ReadFile(backupPath)
	if err != nil {
		t.Fatalf("read backup: %v", err)
	}
	if string(got) != string(sentinel) {
		t.Errorf("backup was clobbered: got %q, want %q", got, sentinel)
	}
}

// When no backup exists, migration creates one containing the original bytes.
func TestLoadServers_MigrationCreatesBackup(t *testing.T) {
	cleanup := setupTestHome(t)
	defer cleanup()

	if err := EnsureDirs(); err != nil {
		t.Fatalf("EnsureDirs: %v", err)
	}
	v1 := []byte("configVersion: 1\nprofiles:\n  - name: legacy\n    server:\n      address: 1.2.3.4\n      port: 443\n")
	if err := os.WriteFile(ServersPath(), v1, 0o600); err != nil {
		t.Fatalf("seed servers.yaml: %v", err)
	}

	if _, err := LoadServers(); err != nil {
		t.Fatalf("LoadServers: %v", err)
	}

	got, err := os.ReadFile(ServersPath() + ".v1.bak")
	if err != nil {
		t.Fatalf("backup not created: %v", err)
	}
	if string(got) != string(v1) {
		t.Errorf("backup content = %q, want the original %q", got, v1)
	}
}

func TestMaskUUID(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"short", "****"},
		{"", "****"},
		{"12345678", "****"},
		{"123e4567-e89b-42d3-a456-426614174000", "123e4567-****-****-****-************"},
	}

	for _, tc := range tests {
		got := MaskUUID(tc.input)
		if got != tc.want {
			t.Errorf("MaskUUID(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}
