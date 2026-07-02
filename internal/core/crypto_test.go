package core

import (
	"testing"

	"github.com/rtxnik/lazyray/internal/config"
)

func TestExportImportEncrypted_Roundtrip(t *testing.T) {
	profiles := []config.Profile{
		{
			Name: "Test Server 1",
			Server: config.ServerConfig{
				Address:    "1.2.3.4",
				Port:       443,
				UUID:       "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
				Encryption: "none",
				Protocol:   "vless",
				Transport:  config.TransportConfig{Network: "xhttp", Path: "/test"},
				Security:   config.SecurityConfig{Type: "reality", SNI: "example.com"},
			},
		},
		{
			Name: "Test Server 2",
			Server: config.ServerConfig{
				Address:   "5.6.7.8",
				Port:      8443,
				UUID:      "testpass",
				Protocol:  "trojan",
				Transport: config.TransportConfig{Network: "ws"},
				Security:  config.SecurityConfig{Type: "tls", SNI: "test.com"},
			},
		},
	}

	password := "mysecretpassword123"

	encrypted, err := ExportEncrypted(profiles, password)
	if err != nil {
		t.Fatalf("ExportEncrypted() error = %v", err)
	}

	if !IsEncryptedExport(encrypted) {
		t.Error("IsEncryptedExport should return true")
	}

	decrypted, err := ImportEncrypted(encrypted, password)
	if err != nil {
		t.Fatalf("ImportEncrypted() error = %v", err)
	}

	if len(decrypted) != 2 {
		t.Fatalf("got %d profiles, want 2", len(decrypted))
	}

	if decrypted[0].Name != "Test Server 1" {
		t.Errorf("profiles[0].Name = %q", decrypted[0].Name)
	}
	if decrypted[0].Server.Address != "1.2.3.4" {
		t.Errorf("profiles[0].Address = %q", decrypted[0].Server.Address)
	}
	if decrypted[1].Server.Protocol != "trojan" {
		t.Errorf("profiles[1].Protocol = %q", decrypted[1].Server.Protocol)
	}
}

func TestImportEncrypted_WrongPassword(t *testing.T) {
	profiles := []config.Profile{
		{
			Name:   "Test",
			Server: config.ServerConfig{Address: "1.2.3.4", Port: 443, UUID: "uuid"},
		},
	}

	encrypted, err := ExportEncrypted(profiles, "correctpass")
	if err != nil {
		t.Fatalf("ExportEncrypted() error = %v", err)
	}

	_, err = ImportEncrypted(encrypted, "wrongpass")
	if err == nil {
		t.Fatal("expected error for wrong password")
	}
}

func TestExportEncrypted_EmptyProfiles(t *testing.T) {
	_, err := ExportEncrypted(nil, "pass")
	if err == nil {
		t.Fatal("expected error for empty profiles")
	}
}

func TestExportEncrypted_EmptyPassword(t *testing.T) {
	profiles := []config.Profile{{Name: "Test"}}
	_, err := ExportEncrypted(profiles, "")
	if err == nil {
		t.Fatal("expected error for empty password")
	}
}

func TestIsEncryptedExport(t *testing.T) {
	if IsEncryptedExport("not encrypted") {
		t.Error("should return false for non-encrypted")
	}
	if !IsEncryptedExport("LZRENC1:somedata") {
		t.Error("should return true for encrypted prefix")
	}
}
