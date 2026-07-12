package core

import (
	"testing"

	"github.com/rtxnik/lazyray/internal/config"
)

func validBase() *config.Profile {
	return &config.Profile{
		Name:   "p",
		Server: config.ServerConfig{Address: "h.example", Port: 443, UUID: "11111111-1111-1111-1111-111111111111", Transport: config.TransportConfig{Network: "tcp"}, Security: config.SecurityConfig{Type: "none"}},
	}
}

func TestValidateProfile_RejectsBadChainPort(t *testing.T) {
	p := validBase()
	p.Chain = []config.ServerConfig{{Address: "c.example", Port: 70000, Transport: config.TransportConfig{Network: "tcp"}}}
	if err := ValidateProfile(p); err == nil {
		t.Fatal("expected error for chain port 70000")
	}
}

func TestValidateProfile_AcceptsValidChain(t *testing.T) {
	p := validBase()
	p.Chain = []config.ServerConfig{{Address: "c.example", Port: 8443, Transport: config.TransportConfig{Network: "tcp"}}}
	if err := ValidateProfile(p); err != nil {
		t.Fatalf("valid chain rejected: %v", err)
	}
}
