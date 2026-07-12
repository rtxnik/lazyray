package cmd

import (
	"testing"

	"github.com/rtxnik/lazyray/internal/config"
	"github.com/rtxnik/lazyray/internal/core"
	"github.com/spf13/cobra"
)

func validImportBase() config.Profile {
	return config.Profile{
		Name:   "p",
		Server: config.ServerConfig{Address: "h.example", Port: 443, UUID: "11111111-1111-1111-1111-111111111111", Transport: config.TransportConfig{Network: "tcp"}, Security: config.SecurityConfig{Type: "none"}},
	}
}

func encExport(t *testing.T, p config.Profile) string {
	t.Helper()
	blob, err := core.ExportEncrypted([]config.Profile{p}, "pw")
	if err != nil {
		t.Fatal(err)
	}
	return blob
}

func TestImportSingleProfile_RejectsBadPortAndDoesNotPersist(t *testing.T) {
	isolateConfig(t)
	err := importSingleProfile(&cobra.Command{}, "vless://11111111-1111-1111-1111-111111111111@h.example:70000?type=tcp#p")
	if err == nil {
		t.Fatal("expected error for port 70000")
	}
	servers, _ := config.LoadServers()
	if len(servers.Profiles) != 0 {
		t.Fatalf("invalid-port profile must not persist, got %+v", servers.Profiles)
	}
}

func TestImportSingleProfile_RejectsZeroPort(t *testing.T) {
	isolateConfig(t)
	err := importSingleProfile(&cobra.Command{}, "vless://11111111-1111-1111-1111-111111111111@h.example:0?type=tcp#p")
	if err == nil {
		t.Fatal("expected error for port 0")
	}
	servers, _ := config.LoadServers()
	if len(servers.Profiles) != 0 {
		t.Fatalf("zero-port profile must not persist, got %+v", servers.Profiles)
	}
}

func TestImportEncrypted_SkipsBadTopLevelPort(t *testing.T) {
	isolateConfig(t)
	p := validImportBase()
	p.Server.Port = 70000
	importDecrypt = "pw"
	importAllowRouting = false
	t.Cleanup(func() { importDecrypt = ""; importAllowRouting = false })
	if err := importEncrypted(&cobra.Command{}, encExport(t, p)); err != nil {
		t.Fatal(err)
	}
	servers, _ := config.LoadServers()
	if len(servers.Profiles) != 0 {
		t.Fatalf("bad top-level port profile must be skipped, got %+v", servers.Profiles)
	}
}

func TestImportEncrypted_SkipsBadChainPort(t *testing.T) {
	isolateConfig(t)
	p := validImportBase()
	p.Chain = []config.ServerConfig{{Address: "c.example", Port: 70000, Transport: config.TransportConfig{Network: "tcp"}}}
	importDecrypt = "pw"
	importAllowRouting = false
	t.Cleanup(func() { importDecrypt = ""; importAllowRouting = false })
	if err := importEncrypted(&cobra.Command{}, encExport(t, p)); err != nil {
		t.Fatal(err)
	}
	servers, _ := config.LoadServers()
	if len(servers.Profiles) != 0 {
		t.Fatalf("bad chain-port profile must be skipped, got %+v", servers.Profiles)
	}
}
