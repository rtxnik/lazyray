package core

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCheckProtocolXraySupport_NonHysteria(t *testing.T) {
	if err := CheckProtocolXraySupport("vless"); err != nil {
		t.Errorf("vless should never be gated: %v", err)
	}
}

func TestConfigUsesHysteria(t *testing.T) {
	dir := t.TempDir()
	hy := filepath.Join(dir, "hy.json")
	if err := os.WriteFile(hy, []byte(`{"outbounds":[{"protocol":"hysteria"}]}`), 0644); err != nil {
		t.Fatal(err)
	}
	if !configUsesHysteria(hy) {
		t.Error("configUsesHysteria = false, want true")
	}
	vl := filepath.Join(dir, "vl.json")
	if err := os.WriteFile(vl, []byte(`{"outbounds":[{"protocol":"vless"}]}`), 0644); err != nil {
		t.Fatal(err)
	}
	if configUsesHysteria(vl) {
		t.Error("configUsesHysteria = true for vless, want false")
	}
}

func TestInsecureRemovedError(t *testing.T) {
	xrayLog := `Failed to start: ... > common/errors: The feature "allowInsecure" has been removed and migrated to "pinnedPeerCertSha256". Please update your config.`
	if err := insecureRemovedError(xrayLog); err == nil {
		t.Fatal("expected a remediation error for the removed-allowInsecure log")
	}
	if err := insecureRemovedError("some unrelated startup failure"); err != nil {
		t.Errorf("unexpected error for unrelated log: %v", err)
	}
}
