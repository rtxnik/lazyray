package config

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func TestServerConfig_Hysteria2FieldsRoundtrip(t *testing.T) {
	in := ServerConfig{
		Address:     "example.com",
		Port:        443,
		UUID:        "auth",
		Protocol:    "hysteria2",
		PortHopping: "443,5000-6000",
		Security:    SecurityConfig{Type: "tls", PinSHA256: "ab12cd"},
	}
	data, err := yaml.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var out ServerConfig
	if err := yaml.Unmarshal(data, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.PortHopping != "443,5000-6000" {
		t.Errorf("PortHopping = %q, want 443,5000-6000", out.PortHopping)
	}
	if out.Security.PinSHA256 != "ab12cd" {
		t.Errorf("PinSHA256 = %q, want ab12cd", out.Security.PinSHA256)
	}
}
