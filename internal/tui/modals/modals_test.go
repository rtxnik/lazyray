package modals

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/rtxnik/lazyray/internal/config"
	"github.com/rtxnik/lazyray/internal/tui/commands"
)

// --- Import Modal Tests ---

func TestImportModal_NewImportModal(t *testing.T) {
	m := NewImportModal(120, 40)
	if m == nil {
		t.Fatal("NewImportModal returned nil")
	}
	if m.Done {
		t.Error("Done should be false initially")
	}
	if m.Profile != nil {
		t.Error("Profile should be nil initially")
	}
	if !m.focusURL {
		t.Error("focusURL should be true initially")
	}
}

func TestImportModal_View(t *testing.T) {
	m := NewImportModal(120, 40)
	view := m.View()

	if !strings.Contains(view, "Import Configuration") {
		t.Error("View should contain title")
	}
	if !strings.Contains(view, "vless://") {
		t.Error("View should contain placeholder")
	}
}

func TestImportModal_TabSwitch(t *testing.T) {
	m := NewImportModal(120, 40)
	if !m.focusURL {
		t.Fatal("should start focused on URL")
	}

	// Press Tab to switch to name input
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}}
	_ = msg // just to use it
	tabMsg := tea.KeyMsg{Type: tea.KeyTab}
	m.Update(tabMsg)

	if m.focusURL {
		t.Error("after Tab, focusURL should be false")
	}

	// Tab back
	m.Update(tabMsg)
	if !m.focusURL {
		t.Error("after second Tab, focusURL should be true")
	}
}

func TestImportModal_EscClosesModal(t *testing.T) {
	m := NewImportModal(120, 40)
	escMsg := tea.KeyMsg{Type: tea.KeyEsc}
	m.Update(escMsg)

	if !m.Done {
		t.Error("Esc should set Done to true")
	}
}

func TestImportModal_SmallTerminal(t *testing.T) {
	m := NewImportModal(30, 15)
	view := m.View()
	if view == "" {
		t.Error("View should render even on small terminals")
	}
}

// --- Help Modal Tests ---

func TestHelpModal_NewHelpModal(t *testing.T) {
	m := NewHelpModal(commands.New(commands.DefaultKeyMap()), 120, 40)
	if m == nil {
		t.Fatal("NewHelpModal returned nil")
	}
	if m.Done {
		t.Error("Done should be false initially")
	}
}

func TestHelpModal_View(t *testing.T) {
	m := NewHelpModal(commands.New(commands.DefaultKeyMap()), 120, 40)
	view := m.View()

	if !strings.Contains(view, "Keyboard Shortcuts") {
		t.Error("View should contain title")
	}
	if !strings.Contains(view, "start / stop") {
		t.Error("View should contain keybinding descriptions")
	}
}

func TestHelpModal_CloseKeys(t *testing.T) {
	closeKeys := []tea.KeyMsg{
		{Type: tea.KeyEnter},
		{Type: tea.KeyEsc},
	}

	for _, key := range closeKeys {
		m := NewHelpModal(commands.New(commands.DefaultKeyMap()), 120, 40)
		m.Update(key)
		if !m.Done {
			t.Errorf("key %v should close help modal", key)
		}
	}
}

// --- Confirm Modal Tests ---

func TestConfirmModal_NewConfirmModal(t *testing.T) {
	m := NewConfirmModal("Test Title", "Are you sure?", "test-action", 120, 40)
	if m == nil {
		t.Fatal("NewConfirmModal returned nil")
	}
	if m.Done {
		t.Error("Done should be false initially")
	}
	if m.Confirmed {
		t.Error("Confirmed should be false initially")
	}
	if m.Action != "test-action" {
		t.Errorf("Action = %q, want %q", m.Action, "test-action")
	}
}

func TestConfirmModal_View(t *testing.T) {
	m := NewConfirmModal("Delete Profile", "Delete this profile?", "delete", 120, 40)
	view := m.View()

	if !strings.Contains(view, "Delete Profile") {
		t.Error("View should contain title")
	}
	if !strings.Contains(view, "Delete this profile?") {
		t.Error("View should contain message")
	}
}

func TestConfirmModal_ConfirmWithY(t *testing.T) {
	m := NewConfirmModal("Title", "Message", "action", 120, 40)
	yMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")}
	m.Update(yMsg)

	if !m.Done {
		t.Error("y should set Done to true")
	}
	if !m.Confirmed {
		t.Error("y should set Confirmed to true")
	}
}

func TestConfirmModal_DenyWithN(t *testing.T) {
	m := NewConfirmModal("Title", "Message", "action", 120, 40)
	nMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")}
	m.Update(nMsg)

	if !m.Done {
		t.Error("n should set Done to true")
	}
	if m.Confirmed {
		t.Error("n should keep Confirmed as false")
	}
}

func TestConfirmModal_EscCancels(t *testing.T) {
	m := NewConfirmModal("Title", "Message", "action", 120, 40)
	escMsg := tea.KeyMsg{Type: tea.KeyEsc}
	m.Update(escMsg)

	if !m.Done {
		t.Error("Esc should set Done to true")
	}
	if m.Confirmed {
		t.Error("Esc should keep Confirmed as false")
	}
}

// --- Edit Modal Tests ---

func TestEditModal_NewEditModal(t *testing.T) {
	profile := &config.Profile{
		Name: "Test Profile",
		Server: config.ServerConfig{
			Address:    "1.2.3.4",
			Port:       8443,
			UUID:       "test-uuid",
			Encryption: "none",
			Transport: config.TransportConfig{
				Network: "xhttp",
				Path:    "/test",
			},
			Security: config.SecurityConfig{
				Type:        "reality",
				SNI:         "example.com",
				Fingerprint: "chrome",
				PublicKey:   "testkey",
				ShortID:     "1234",
			},
		},
	}

	m := NewEditModal(profile, 120, 40)
	if m == nil {
		t.Fatal("NewEditModal returned nil")
	}
	if m.Done {
		t.Error("Done should be false initially")
	}
	if m.focusIndex != 0 {
		t.Errorf("focusIndex should start at 0, got %d", m.focusIndex)
	}
}

func TestEditModal_View(t *testing.T) {
	profile := &config.Profile{
		Name: "Test",
		Server: config.ServerConfig{
			Address: "1.2.3.4",
			Port:    443,
			UUID:    "uuid",
			Transport: config.TransportConfig{
				Network: "tcp",
			},
			Security: config.SecurityConfig{
				Type: "none",
			},
		},
	}

	m := NewEditModal(profile, 120, 40)
	view := m.View()

	if !strings.Contains(view, "Edit Profile") {
		t.Error("View should contain title")
	}
	if !strings.Contains(view, "Name") {
		t.Error("View should contain field labels")
	}
}

func TestEditModal_TabNavigation(t *testing.T) {
	profile := &config.Profile{
		Name: "Test",
		Server: config.ServerConfig{
			Address: "1.2.3.4",
			Port:    443,
			UUID:    "uuid",
			Transport: config.TransportConfig{
				Network: "tcp",
			},
			Security: config.SecurityConfig{
				Type: "none",
			},
		},
	}

	m := NewEditModal(profile, 120, 40)
	if m.focusIndex != 0 {
		t.Fatal("should start at field 0")
	}

	tabMsg := tea.KeyMsg{Type: tea.KeyTab}
	m.Update(tabMsg)
	if m.focusIndex != 1 {
		t.Errorf("after Tab, focusIndex = %d, want 1", m.focusIndex)
	}

	// Shift+Tab should go back
	shiftTabMsg := tea.KeyMsg{Type: tea.KeyShiftTab}
	m.Update(shiftTabMsg)
	if m.focusIndex != 0 {
		t.Errorf("after Shift+Tab, focusIndex = %d, want 0", m.focusIndex)
	}
}

func TestEditModal_EscCloses(t *testing.T) {
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

	m := NewEditModal(profile, 120, 40)
	escMsg := tea.KeyMsg{Type: tea.KeyEsc}
	m.Update(escMsg)

	if !m.Done {
		t.Error("Esc should close the edit modal")
	}
}

// --- QR Modal Tests ---

func TestQRModal_NewQRModal(t *testing.T) {
	m := NewQRModal("Test Profile", "vless://test@1.2.3.4:443#test", 120, 40)
	if m == nil {
		t.Fatal("NewQRModal returned nil")
	}
	if m.Done {
		t.Error("Done should be false initially")
	}
}

func TestQRModal_View(t *testing.T) {
	m := NewQRModal("My Profile", "vless://uuid@host:443#name", 120, 40)
	view := m.View()

	if view == "" {
		t.Error("QR modal View should not be empty")
	}
}

func TestQRModal_CloseKeys(t *testing.T) {
	m := NewQRModal("Test", "vless://test", 120, 40)
	escMsg := tea.KeyMsg{Type: tea.KeyEsc}
	m.Update(escMsg)

	if !m.Done {
		t.Error("Esc should close QR modal")
	}
}

// --- Diff Modal Tests ---

func TestDiffModal_NewDiffModal(t *testing.T) {
	old := []string{"line1", "line2", "same"}
	new := []string{"line1", "changed", "same"}

	m := NewDiffModal("Config Diff", old, new, 120, 40)
	if m == nil {
		t.Fatal("NewDiffModal returned nil")
	}
	if m.Done {
		t.Error("Done should be false initially")
	}
}

func TestDiffModal_View(t *testing.T) {
	old := []string{`"port": 10808`, `"listen": "127.0.0.1"`}
	new := []string{`"port": 1080`, `"listen": "127.0.0.1"`}

	m := NewDiffModal("Config Diff", old, new, 120, 40)
	view := m.View()

	if view == "" {
		t.Error("Diff modal View should not be empty")
	}
	if !strings.Contains(view, "Config Diff") {
		t.Error("View should contain title")
	}
}

func TestDiffModal_Close(t *testing.T) {
	m := NewDiffModal("Test", []string{"a"}, []string{"b"}, 120, 40)
	escMsg := tea.KeyMsg{Type: tea.KeyEsc}
	m.Update(escMsg)

	if !m.Done {
		t.Error("Esc should close diff modal")
	}
}

// --- Subscription Modal Tests ---

func TestSubscriptionModal_NewEmpty(t *testing.T) {
	m := NewSubscriptionModal(nil, 120, 40)
	if m == nil {
		t.Fatal("NewSubscriptionModal returned nil")
	}
	if m.Done {
		t.Error("Done should be false initially")
	}
	if m.adding {
		t.Error("adding should be false initially")
	}
}

func TestSubscriptionModal_ViewEmpty(t *testing.T) {
	m := NewSubscriptionModal(nil, 120, 40)
	view := m.View()

	if !strings.Contains(view, "Subscriptions") {
		t.Error("View should contain title")
	}
	if !strings.Contains(view, "No subscriptions") {
		t.Error("View should show 'no subscriptions' message")
	}
}

func TestSubscriptionModal_ViewWithSubs(t *testing.T) {
	subs := []config.SubscriptionEntry{
		{Name: "Sub 1", URL: "https://example.com/sub1"},
		{Name: "Sub 2", URL: "https://example.com/sub2"},
	}

	m := NewSubscriptionModal(subs, 120, 40)
	view := m.View()

	if !strings.Contains(view, "Sub 1") {
		t.Error("View should contain subscription name")
	}
	if !strings.Contains(view, "Sub 2") {
		t.Error("View should contain second subscription name")
	}
}

func TestSubscriptionModal_NavigateList(t *testing.T) {
	subs := []config.SubscriptionEntry{
		{Name: "Sub 1", URL: "https://example.com/sub1"},
		{Name: "Sub 2", URL: "https://example.com/sub2"},
	}

	m := NewSubscriptionModal(subs, 120, 40)
	if m.selected != 0 {
		t.Fatal("should start at 0")
	}

	downMsg := tea.KeyMsg{Type: tea.KeyDown}
	m.Update(downMsg)
	if m.selected != 1 {
		t.Errorf("after down, selected = %d, want 1", m.selected)
	}

	upMsg := tea.KeyMsg{Type: tea.KeyUp}
	m.Update(upMsg)
	if m.selected != 0 {
		t.Errorf("after up, selected = %d, want 0", m.selected)
	}
}

func TestSubscriptionModal_AddMode(t *testing.T) {
	m := NewSubscriptionModal(nil, 120, 40)

	aMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")}
	m.Update(aMsg)

	if !m.adding {
		t.Error("pressing 'a' should enter add mode")
	}

	// Esc in add mode should go back to list mode
	escMsg := tea.KeyMsg{Type: tea.KeyEsc}
	m.Update(escMsg)
	if m.adding {
		t.Error("Esc in add mode should return to list mode")
	}
}

func TestSubscriptionModal_DeleteAction(t *testing.T) {
	subs := []config.SubscriptionEntry{
		{Name: "Sub 1", URL: "https://example.com/sub1"},
	}

	m := NewSubscriptionModal(subs, 120, 40)
	dMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")}
	m.Update(dMsg)

	if !m.Done {
		t.Error("d should trigger delete and close modal")
	}
	if m.Action != SubActionDelete {
		t.Errorf("Action = %v, want SubActionDelete", m.Action)
	}
	if m.DeleteIndex != 0 {
		t.Errorf("DeleteIndex = %d, want 0", m.DeleteIndex)
	}
}

func TestSubscriptionModal_UpdateAction(t *testing.T) {
	subs := []config.SubscriptionEntry{
		{Name: "Sub 1", URL: "https://example.com/sub1"},
	}

	m := NewSubscriptionModal(subs, 120, 40)
	uMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("u")}
	m.Update(uMsg)

	if !m.Done {
		t.Error("u should trigger update and close modal")
	}
	if m.Action != SubActionUpdate {
		t.Errorf("Action = %v, want SubActionUpdate", m.Action)
	}
	if m.SubURL != "https://example.com/sub1" {
		t.Errorf("SubURL = %q, want subscription URL", m.SubURL)
	}
}

func TestSubscriptionModal_EscClosesList(t *testing.T) {
	m := NewSubscriptionModal(nil, 120, 40)
	escMsg := tea.KeyMsg{Type: tea.KeyEsc}
	m.Update(escMsg)

	if !m.Done {
		t.Error("Esc should close the modal")
	}
	if m.Action != SubActionNone {
		t.Errorf("Action = %v, want SubActionNone", m.Action)
	}
}

// --- DNS Rule Parsing Tests ---

func TestParseDNSRules_Empty(t *testing.T) {
	rules, err := parseDNSRules("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rules) != 0 {
		t.Errorf("expected 0 rules, got %d", len(rules))
	}
}

func TestParseDNSRules_SingleRule(t *testing.T) {
	rules, err := parseDNSRules("domain:example.com>https://dns.google/dns-query")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(rules))
	}
	if rules[0].Server != "https://dns.google/dns-query" {
		t.Errorf("server = %q, want https://dns.google/dns-query", rules[0].Server)
	}
	if len(rules[0].Domains) != 1 || rules[0].Domains[0] != "domain:example.com" {
		t.Errorf("domains = %v, want [domain:example.com]", rules[0].Domains)
	}
}

func TestParseDNSRules_MultipleDomains(t *testing.T) {
	rules, err := parseDNSRules("domain:google.com;domain:youtube.com>https://dns.google/dns-query")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(rules))
	}
	if len(rules[0].Domains) != 2 {
		t.Errorf("domains count = %d, want 2", len(rules[0].Domains))
	}
}

func TestParseDNSRules_MultipleRules(t *testing.T) {
	input := "domain:google.com>https://dns.google/dns-query, geosite:private>1.1.1.1"
	rules, err := parseDNSRules(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rules) != 2 {
		t.Fatalf("expected 2 rules, got %d", len(rules))
	}
	if rules[0].Server != "https://dns.google/dns-query" {
		t.Errorf("rule[0] server = %q", rules[0].Server)
	}
	if rules[1].Server != "1.1.1.1" {
		t.Errorf("rule[1] server = %q", rules[1].Server)
	}
}

func TestParseDNSRules_InvalidNoSeparator(t *testing.T) {
	_, err := parseDNSRules("domain:example.com")
	if err == nil {
		t.Error("expected error for rule without > separator")
	}
}

func TestParseDNSRules_InvalidEmptyServer(t *testing.T) {
	_, err := parseDNSRules("domain:example.com>")
	if err == nil {
		t.Error("expected error for empty server")
	}
}

func TestFormatDNSRules_Empty(t *testing.T) {
	result := formatDNSRules(nil)
	if result != "" {
		t.Errorf("expected empty string, got %q", result)
	}
}

func TestFormatDNSRules_RoundTrip(t *testing.T) {
	rules := []config.DNSRule{
		{
			Server:  "https://dns.google/dns-query",
			Domains: []string{"domain:google.com", "domain:youtube.com"},
		},
		{
			Server:  "1.1.1.1",
			Domains: []string{"geosite:private"},
		},
	}

	formatted := formatDNSRules(rules)
	parsed, err := parseDNSRules(formatted)
	if err != nil {
		t.Fatalf("round-trip parse error: %v", err)
	}
	if len(parsed) != len(rules) {
		t.Fatalf("round-trip: got %d rules, want %d", len(parsed), len(rules))
	}
	for i, r := range parsed {
		if r.Server != rules[i].Server {
			t.Errorf("rule[%d] server = %q, want %q", i, r.Server, rules[i].Server)
		}
		if len(r.Domains) != len(rules[i].Domains) {
			t.Errorf("rule[%d] domains count = %d, want %d", i, len(r.Domains), len(rules[i].Domains))
		}
	}
}

// --- Routing Modal DNS Integration Tests ---

func TestRoutingModal_NewRoutingModal_WithDNSRules(t *testing.T) {
	profile := &config.Profile{
		Name: "Test",
		Server: config.ServerConfig{
			Address:   "1.2.3.4",
			Port:      443,
			UUID:      "uuid",
			Transport: config.TransportConfig{Network: "tcp"},
			Security:  config.SecurityConfig{Type: "none"},
		},
		Routing: config.ProfileRouting{
			Bypass: []string{"geoip:private"},
			Block:  []string{"geosite:category-ads"},
			DNSRules: []config.DNSRule{
				{Server: "https://dns.google/dns-query", Domains: []string{"domain:google.com"}},
			},
		},
	}

	m := NewRoutingModal(profile, 120, 40)
	view := m.View()

	if !strings.Contains(view, "DNS Rules") {
		t.Error("View should contain DNS Rules section")
	}
	if !strings.Contains(view, "Routing Rules") {
		t.Error("View should contain title")
	}
}

func TestRoutingModal_ThreeFieldNavigation(t *testing.T) {
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
	if m.focusIndex != 0 {
		t.Fatal("should start at bypass field")
	}

	tabMsg := tea.KeyMsg{Type: tea.KeyTab}
	m.Update(tabMsg)
	if m.focusIndex != 1 {
		t.Errorf("after Tab, focusIndex = %d, want 1 (block)", m.focusIndex)
	}

	m.Update(tabMsg)
	if m.focusIndex != 2 {
		t.Errorf("after second Tab, focusIndex = %d, want 2 (dns)", m.focusIndex)
	}

	m.Update(tabMsg)
	if m.focusIndex != 0 {
		t.Errorf("after third Tab, focusIndex = %d, want 0 (bypass, wrap)", m.focusIndex)
	}
}
