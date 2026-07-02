package modals

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/rtxnik/lazyray/internal/config"
	"github.com/rtxnik/lazyray/internal/tui/commands"
)

// --- QR Modal: all protocol URLs ---

func TestQRModal_VLESSUrl(t *testing.T) {
	url := "vless://uuid@1.2.3.4:443?type=xhttp&security=reality&sni=example.com&fp=chrome&pbk=key&sid=1234#VLESS+Profile"
	m := NewQRModal("VLESS Profile", url, 120, 40)

	view := m.View()
	if view == "" {
		t.Error("QR modal should render for VLESS URL")
	}
	if m.Done {
		t.Error("modal should not be done initially")
	}
}

func TestQRModal_VMessUrl(t *testing.T) {
	url := "vmess://eyJ2IjoiMiIsInBzIjoiVk1lc3MiLCJhZGQiOiIxLjIuMy40IiwicG9ydCI6NDQzLCJpZCI6InV1aWQiLCJhaWQiOjAsIm5ldCI6IndzIiwidGxzIjoidGxzIn0="
	m := NewQRModal("VMess Profile", url, 120, 40)

	view := m.View()
	if view == "" {
		t.Error("QR modal should render for VMess URL")
	}
}

func TestQRModal_TrojanUrl(t *testing.T) {
	url := "trojan://password@1.2.3.4:443?sni=example.com#Trojan+Profile"
	m := NewQRModal("Trojan Profile", url, 120, 40)

	view := m.View()
	if view == "" {
		t.Error("QR modal should render for Trojan URL")
	}
}

func TestQRModal_ShadowsocksUrl(t *testing.T) {
	url := "ss://YWVzLTI1Ni1nY206cGFzc3dvcmQ@1.2.3.4:8388#SS+Profile"
	m := NewQRModal("SS Profile", url, 120, 40)

	view := m.View()
	if view == "" {
		t.Error("QR modal should render for Shadowsocks URL")
	}
}

func TestQRModal_LongUrl(t *testing.T) {
	// Test with a very long URL to ensure no panics
	url := "vless://aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee@very-long-hostname.example.com:8443?type=xhttp&security=reality&sni=very-long-sni.example.com&fp=chrome&pbk=DXLqqc2ZxtxKHm_ab5GnF59s4d0SLpWz8VOwlsW3wyY&sid=abc123&spx=%2Fsome%2Flong%2Fpath%2Fhere#Very+Long+Profile+Name+For+Testing"
	m := NewQRModal("Very Long Profile", url, 120, 40)

	view := m.View()
	if view == "" {
		t.Error("QR modal should render for long URL")
	}
}

func TestQRModal_SmallTerminal(t *testing.T) {
	m := NewQRModal("Test", "vless://uuid@1.2.3.4:443#test", 40, 20)
	view := m.View()
	if view == "" {
		t.Error("QR modal should render on small terminal")
	}
}

func TestQRModal_CloseWithQ(t *testing.T) {
	m := NewQRModal("Test", "vless://test", 120, 40)
	qMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")}
	m.Update(qMsg)

	if !m.Done {
		t.Error("q should close QR modal")
	}
}

func TestQRModal_CloseWithEnter(t *testing.T) {
	m := NewQRModal("Test", "vless://test", 120, 40)
	enterMsg := tea.KeyMsg{Type: tea.KeyEnter}
	m.Update(enterMsg)

	if !m.Done {
		t.Error("Enter should close QR modal")
	}
}

// --- Routing Modal: additional tests ---

func TestRoutingModal_NewRoutingModal_EmptyProfile(t *testing.T) {
	profile := &config.Profile{
		Name: "Empty",
		Server: config.ServerConfig{
			Address:   "1.2.3.4",
			Port:      443,
			UUID:      "uuid",
			Transport: config.TransportConfig{Network: "tcp"},
			Security:  config.SecurityConfig{Type: "none"},
		},
	}

	m := NewRoutingModal(profile, 120, 40)
	if m == nil {
		t.Fatal("NewRoutingModal returned nil")
	}
	view := m.View()
	if !strings.Contains(view, "Routing Rules") {
		t.Error("View should contain Routing Rules title")
	}
}

func TestRoutingModal_EscCloses(t *testing.T) {
	profile := &config.Profile{
		Name: "Test",
		Server: config.ServerConfig{
			Address:   "1.2.3.4",
			Port:      443,
			UUID:      "uuid",
			Transport: config.TransportConfig{Network: "tcp"},
			Security:  config.SecurityConfig{Type: "none"},
		},
	}

	m := NewRoutingModal(profile, 120, 40)
	escMsg := tea.KeyMsg{Type: tea.KeyEsc}
	m.Update(escMsg)

	if !m.Done {
		t.Error("Esc should close routing modal")
	}
}

func TestRoutingModal_WithBypassAndBlock(t *testing.T) {
	profile := &config.Profile{
		Name: "Full",
		Server: config.ServerConfig{
			Address:   "1.2.3.4",
			Port:      443,
			UUID:      "uuid",
			Transport: config.TransportConfig{Network: "tcp"},
			Security:  config.SecurityConfig{Type: "none"},
		},
		Routing: config.ProfileRouting{
			Bypass: []string{"geoip:private", "geosite:private"},
			Block:  []string{"geosite:category-ads-all"},
		},
	}

	m := NewRoutingModal(profile, 120, 40)
	view := m.View()

	if !strings.Contains(view, "Routing Rules") {
		t.Error("View should contain title")
	}
}

func TestRoutingModal_ShiftTabNavigation(t *testing.T) {
	profile := &config.Profile{
		Name: "Test",
		Server: config.ServerConfig{
			Address:   "1.2.3.4",
			Port:      443,
			UUID:      "uuid",
			Transport: config.TransportConfig{Network: "tcp"},
			Security:  config.SecurityConfig{Type: "none"},
		},
	}

	m := NewRoutingModal(profile, 120, 40)
	// Start at 0 (bypass)

	// Shift+Tab wraps to last field
	stMsg := tea.KeyMsg{Type: tea.KeyShiftTab}
	m.Update(stMsg)
	if m.focusIndex != 2 {
		t.Errorf("after Shift+Tab from 0, focusIndex = %d, want 2 (wrap)", m.focusIndex)
	}
}

// --- DNS Rule parsing: additional edge cases ---

func TestParseDNSRules_WhitespaceHandling(t *testing.T) {
	rules, err := parseDNSRules("  domain:example.com > https://dns.google/dns-query  ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(rules))
	}
	if rules[0].Server != "https://dns.google/dns-query" {
		t.Errorf("server = %q", rules[0].Server)
	}
}

func TestParseDNSRules_GeositePrefix(t *testing.T) {
	rules, err := parseDNSRules("geosite:cn>1.1.1.1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(rules))
	}
	if rules[0].Domains[0] != "geosite:cn" {
		t.Errorf("domain = %q, want geosite:cn", rules[0].Domains[0])
	}
}

// --- Edit Modal: additional protocol fields ---

func TestEditModal_ShadowsocksProfile(t *testing.T) {
	profile := &config.Profile{
		Name: "SS Profile",
		Server: config.ServerConfig{
			Address:    "1.2.3.4",
			Port:       8388,
			UUID:       "password",
			Encryption: "aes-256-gcm",
			Protocol:   "shadowsocks",
			Transport:  config.TransportConfig{Network: "tcp"},
			Security:   config.SecurityConfig{Type: "none"},
		},
	}

	m := NewEditModal(profile, 120, 40)
	if m == nil {
		t.Fatal("NewEditModal returned nil for SS profile")
	}
	view := m.View()
	if !strings.Contains(view, "Edit Profile") {
		t.Error("View should contain title")
	}
}

func TestEditModal_VMessProfile(t *testing.T) {
	profile := &config.Profile{
		Name: "VMess Profile",
		Server: config.ServerConfig{
			Address:    "1.2.3.4",
			Port:       443,
			UUID:       "uuid",
			Encryption: "auto",
			Protocol:   "vmess",
			Transport:  config.TransportConfig{Network: "ws", Path: "/ws"},
			Security:   config.SecurityConfig{Type: "tls", SNI: "example.com"},
		},
	}

	m := NewEditModal(profile, 120, 40)
	if m == nil {
		t.Fatal("NewEditModal returned nil for VMess profile")
	}
	view := m.View()
	if !strings.Contains(view, "Edit Profile") {
		t.Error("View should contain title")
	}
}

func TestEditModal_TrojanProfile(t *testing.T) {
	profile := &config.Profile{
		Name: "Trojan Profile",
		Server: config.ServerConfig{
			Address:   "1.2.3.4",
			Port:      443,
			UUID:      "trojanpass",
			Protocol:  "trojan",
			Transport: config.TransportConfig{Network: "tcp"},
			Security:  config.SecurityConfig{Type: "tls", SNI: "example.com"},
		},
	}

	m := NewEditModal(profile, 120, 40)
	if m == nil {
		t.Fatal("NewEditModal returned nil for Trojan profile")
	}
	view := m.View()
	if !strings.Contains(view, "Edit Profile") {
		t.Error("View should contain title")
	}
}

// --- Wizard Modal tests ---

func TestWizardModal_NewWizardModal(t *testing.T) {
	m := NewWizardModal(120, 40)
	if m == nil {
		t.Fatal("NewWizardModal returned nil")
	}
	if m.Done {
		t.Error("Done should be false initially")
	}
}

func TestWizardModal_View(t *testing.T) {
	m := NewWizardModal(120, 40)
	view := m.View()
	if view == "" {
		t.Error("Wizard modal View should not be empty")
	}
}

func TestWizardModal_EscCloses(t *testing.T) {
	m := NewWizardModal(120, 40)
	escMsg := tea.KeyMsg{Type: tea.KeyEsc}
	m.Update(escMsg)

	if !m.Done {
		t.Error("Esc should close wizard modal")
	}
}

// --- Diff Modal: additional tests ---

func TestDiffModal_LargeContent(t *testing.T) {
	old := make([]string, 100)
	new := make([]string, 100)
	for i := range old {
		old[i] = "line " + string(rune('A'+i%26))
		new[i] = "line " + string(rune('A'+(i+1)%26))
	}

	m := NewDiffModal("Big Diff", old, new, 120, 40)
	if m == nil {
		t.Fatal("NewDiffModal returned nil for large content")
	}
	view := m.View()
	if view == "" {
		t.Error("View should not be empty")
	}
}

func TestDiffModal_IdenticalContent(t *testing.T) {
	lines := []string{"same1", "same2", "same3"}

	m := NewDiffModal("No Changes", lines, lines, 120, 40)
	view := m.View()
	if view == "" {
		t.Error("View should render even for identical content")
	}
}

func TestDiffModal_EmptyContent(t *testing.T) {
	m := NewDiffModal("Empty", []string{}, []string{}, 120, 40)
	view := m.View()
	if view == "" {
		t.Error("View should render for empty content")
	}
}

// --- Subscription Modal extra tests ---

func TestSubscriptionModal_Empty(t *testing.T) {
	m := NewSubscriptionModal([]config.SubscriptionEntry{}, 120, 40)
	if m == nil {
		t.Fatal("NewSubscriptionModal returned nil")
	}
	view := m.View()
	if view == "" {
		t.Error("View should not be empty")
	}
}

func TestSubscriptionModal_WithEntries(t *testing.T) {
	entries := []config.SubscriptionEntry{
		{Name: "Sub 1", URL: "https://example.com/sub1"},
		{Name: "Sub 2", URL: "https://example.com/sub2"},
	}
	m := NewSubscriptionModal(entries, 120, 40)
	view := m.View()
	if view == "" {
		t.Error("View should not be empty")
	}
}

// --- Import Modal extra tests ---

func TestImportModal_New(t *testing.T) {
	m := NewImportModal(120, 40)
	if m == nil {
		t.Fatal("NewImportModal returned nil")
	}
	if m.Done {
		t.Error("Done should be false initially")
	}
}

func TestImportModal_View_8C(t *testing.T) {
	m := NewImportModal(120, 40)
	view := m.View()
	if view == "" {
		t.Error("View should not be empty")
	}
}

// --- Confirm Modal extended tests ---

func TestConfirmModal_WithLongMessage(t *testing.T) {
	m := NewConfirmModal("Delete", "Are you sure you want to delete this very long profile name that spans multiple lines?", "delete", 120, 40)
	view := m.View()
	if view == "" {
		t.Error("View should not be empty")
	}
}

// --- Help Modal extra tests ---

func TestHelpModal_View_Extra(t *testing.T) {
	m := NewHelpModal(commands.New(commands.DefaultKeyMap()), 120, 40)
	view := m.View()
	if view == "" {
		t.Error("Help modal view should not be empty")
	}
}
