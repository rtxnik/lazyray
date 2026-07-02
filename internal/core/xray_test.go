package core

import (
	"strings"
	"testing"

	"github.com/rtxnik/lazyray/internal/config"
)

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		input int64
		want  string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1023, "1023 B"},
		{1024, "1 KB"},
		{1536, "2 KB"},
		{1048576, "1 MB"},
		{1073741824, "1.0 GB"},
		{2147483648, "2.0 GB"},
	}

	for _, tc := range tests {
		got := FormatBytes(tc.input)
		if got != tc.want {
			t.Errorf("FormatBytes(%d) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestValidateProfile_Valid(t *testing.T) {
	p := &config.Profile{
		Name: "test",
		Server: config.ServerConfig{
			Address: "1.2.3.4",
			Port:    443,
			UUID:    "123e4567-e89b-42d3-a456-426614174000",
			Transport: config.TransportConfig{
				Network: "xhttp",
			},
			Security: config.SecurityConfig{
				Type:        "reality",
				SNI:         "example.com",
				Fingerprint: "chrome",
				PublicKey:   "AAAA",
			},
		},
	}

	if err := ValidateProfile(p); err != nil {
		t.Errorf("ValidateProfile() returned unexpected error: %v", err)
	}
}

func TestValidateProfile_MissingUUID(t *testing.T) {
	p := &config.Profile{
		Name: "test",
		Server: config.ServerConfig{
			Address: "1.2.3.4",
			Port:    443,
			UUID:    "",
			Transport: config.TransportConfig{
				Network: "tcp",
			},
		},
	}

	err := ValidateProfile(p)
	if err == nil {
		t.Fatal("expected error for missing UUID")
	}
	if !strings.Contains(err.Error(), "UUID") {
		t.Errorf("error %q should mention UUID", err.Error())
	}
}

func TestValidateProfile_InvalidPort(t *testing.T) {
	tests := []struct {
		name string
		port int
	}{
		{"zero", 0},
		{"negative", -1},
		{"too high", 70000},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := &config.Profile{
				Name: "test",
				Server: config.ServerConfig{
					Address: "1.2.3.4",
					Port:    tc.port,
					UUID:    "valid-uuid",
					Transport: config.TransportConfig{
						Network: "tcp",
					},
				},
			}

			err := ValidateProfile(p)
			if err == nil {
				t.Fatal("expected error for invalid port")
			}
			if !strings.Contains(err.Error(), "port") {
				t.Errorf("error %q should mention port", err.Error())
			}
		})
	}
}

func TestValidateProfile_MissingAddress(t *testing.T) {
	p := &config.Profile{
		Name: "test",
		Server: config.ServerConfig{
			Address: "",
			Port:    443,
			UUID:    "valid-uuid",
			Transport: config.TransportConfig{
				Network: "tcp",
			},
		},
	}

	err := ValidateProfile(p)
	if err == nil {
		t.Fatal("expected error for missing address")
	}
	if !strings.Contains(err.Error(), "address") {
		t.Errorf("error %q should mention address", err.Error())
	}
}

func TestValidateProfile_MissingNetwork(t *testing.T) {
	p := &config.Profile{
		Name: "test",
		Server: config.ServerConfig{
			Address: "1.2.3.4",
			Port:    443,
			UUID:    "valid-uuid",
			Transport: config.TransportConfig{
				Network: "",
			},
		},
	}

	err := ValidateProfile(p)
	if err == nil {
		t.Fatal("expected error for missing network")
	}
	if !strings.Contains(err.Error(), "network") {
		t.Errorf("error %q should mention network", err.Error())
	}
}

func TestValidateProfile_RealityMissingFields(t *testing.T) {
	p := &config.Profile{
		Name: "test",
		Server: config.ServerConfig{
			Address: "1.2.3.4",
			Port:    443,
			UUID:    "valid-uuid",
			Transport: config.TransportConfig{
				Network: "tcp",
			},
			Security: config.SecurityConfig{
				Type:        "reality",
				SNI:         "",
				Fingerprint: "",
				PublicKey:   "",
			},
		},
	}

	err := ValidateProfile(p)
	if err == nil {
		t.Fatal("expected error for reality missing fields")
	}
	errStr := err.Error()
	if !strings.Contains(errStr, "publicKey") {
		t.Errorf("error %q should mention publicKey", errStr)
	}
	if !strings.Contains(errStr, "SNI") {
		t.Errorf("error %q should mention SNI", errStr)
	}
	if !strings.Contains(errStr, "fingerprint") {
		t.Errorf("error %q should mention fingerprint", errStr)
	}
}

func TestValidateProfile_MultipleErrors(t *testing.T) {
	p := &config.Profile{
		Name: "test",
		Server: config.ServerConfig{
			Address: "",
			Port:    0,
			UUID:    "",
			Transport: config.TransportConfig{
				Network: "",
			},
		},
	}

	err := ValidateProfile(p)
	if err == nil {
		t.Fatal("expected error for multiple issues")
	}
	// Should contain multiple error messages separated by ";"
	parts := strings.Split(err.Error(), ";")
	if len(parts) < 3 {
		t.Errorf("expected at least 3 error parts, got %d: %s", len(parts), err.Error())
	}
}
