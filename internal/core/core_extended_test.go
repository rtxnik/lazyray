package core

import (
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/rtxnik/lazyray/internal/config"
)

// --- XrayProcess tests ---

func TestNewXrayProcess(t *testing.T) {
	p := NewXrayProcess()
	if p == nil {
		t.Fatal("NewXrayProcess() returned nil")
	}
}

func TestNewXrayProcess_IsNotRunning(t *testing.T) {
	p := NewXrayProcess()
	if p.IsRunning() {
		t.Error("new XrayProcess should not be running")
	}
}

func TestNewXrayProcess_GetPID_Zero(t *testing.T) {
	p := NewXrayProcess()
	pid := p.GetPID()
	// Should be 0 or a found xray PID (unlikely in test)
	if pid < 0 {
		t.Errorf("GetPID() = %d, should be >= 0", pid)
	}
}

func TestNewXrayProcess_WatchdogRunning_False(t *testing.T) {
	p := NewXrayProcess()
	if p.WatchdogRunning() {
		t.Error("new XrayProcess should not have watchdog running")
	}
}

func TestNewXrayProcess_StopWatchdog_NoOp(t *testing.T) {
	p := NewXrayProcess()
	// Should not panic on a non-started watchdog
	p.StopWatchdog()
	if p.WatchdogRunning() {
		t.Error("watchdog should still be false after StopWatchdog on new process")
	}
}

// --- Port tests ---

func TestCheckPortAvailable_FreePort(t *testing.T) {
	// Find a free port
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to get free port: %v", err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	ln.Close() // Free the port

	err = checkPortAvailable("127.0.0.1", port)
	if err != nil {
		t.Errorf("checkPortAvailable() should succeed for free port %d: %v", port, err)
	}
}

func TestCheckPortAvailable_BusyPort(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to start listener: %v", err)
	}
	defer ln.Close()

	port := ln.Addr().(*net.TCPAddr).Port
	err = checkPortAvailable("127.0.0.1", port)
	if err == nil {
		t.Errorf("checkPortAvailable() should fail for busy port %d", port)
	}
}

func TestIsPortOpen_Open(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to start listener: %v", err)
	}
	defer ln.Close()

	port := ln.Addr().(*net.TCPAddr).Port
	if !isPortOpen("127.0.0.1", port) {
		t.Errorf("isPortOpen() should return true for open port %d", port)
	}
}

func TestIsPortOpen_Closed(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to get free port: %v", err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	ln.Close()

	if isPortOpen("127.0.0.1", port) {
		t.Errorf("isPortOpen() should return false for closed port %d", port)
	}
}

// --- Health report JSON ---

func TestHealthReportJSON(t *testing.T) {
	report := &HealthReport{
		Timestamp: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		AllPassed: true,
		Checks: []CheckResult{
			{Name: "Process", OK: true, Detail: "running (PID 123)"},
			{Name: "SOCKS5", OK: true, Detail: "127.0.0.1:10808 accepting"},
		},
	}

	jsonStr, err := HealthReportJSON(report)
	if err != nil {
		t.Fatalf("HealthReportJSON() error = %v", err)
	}

	// Verify valid JSON
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		t.Fatalf("HealthReportJSON() produced invalid JSON: %v", err)
	}

	// Verify key fields
	if parsed["AllPassed"] != true {
		t.Error("AllPassed should be true")
	}

	checks, ok := parsed["Checks"].([]interface{})
	if !ok || len(checks) != 2 {
		t.Errorf("Checks count = %d, want 2", len(checks))
	}
}

func TestHealthReportJSON_Failed(t *testing.T) {
	report := &HealthReport{
		Timestamp: time.Now(),
		AllPassed: false,
		Checks: []CheckResult{
			{Name: "Process", OK: false, Detail: "not running"},
		},
	}

	jsonStr, err := HealthReportJSON(report)
	if err != nil {
		t.Fatalf("HealthReportJSON() error = %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		t.Fatalf("HealthReportJSON() produced invalid JSON: %v", err)
	}

	if parsed["AllPassed"] != false {
		t.Error("AllPassed should be false")
	}
}

// --- Tunnel manager tests ---

func TestNewTunnelManager(t *testing.T) {
	tm := NewTunnelManager()
	if tm == nil {
		t.Fatal("NewTunnelManager() returned nil")
	}
}

func TestTunnelManager_Status_Empty(t *testing.T) {
	tm := NewTunnelManager()
	statuses := tm.Status(nil)
	if len(statuses) != 0 {
		t.Errorf("Status with nil profiles should return empty, got %d", len(statuses))
	}
}

func TestTunnelManager_Status_NoSSH(t *testing.T) {
	tm := NewTunnelManager()
	profiles := []config.Profile{
		{Name: "test", Server: config.ServerConfig{Address: "1.2.3.4", Port: 443}},
	}
	statuses := tm.Status(profiles)
	// Profile has no SSH config, so it should be skipped
	if len(statuses) != 0 {
		t.Errorf("Status with no SSH profiles should return empty, got %d", len(statuses))
	}
}

func TestTunnelManager_Disconnect_NotFound(t *testing.T) {
	tm := NewTunnelManager()
	err := tm.Disconnect("nonexistent")
	if err == nil {
		t.Error("Disconnect nonexistent tunnel should return error")
	}
}

func TestTunnelManager_DisconnectAll_Empty(t *testing.T) {
	tm := NewTunnelManager()
	// Should not panic on empty
	tm.DisconnectAll()
}

// --- ANSI regex ---

func TestAnsiRegex(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"\x1b[32mgreen\x1b[0m", "green"},
		{"\x1b[1;31mred\x1b[0m", "red"},
		{"no ansi here", "no ansi here"},
		{"", ""},
	}

	for _, tc := range tests {
		got := ansiRegex.ReplaceAllString(tc.input, "")
		if got != tc.want {
			t.Errorf("ansiRegex cleanup(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

// --- SelfUpdate tests ---
// SelfAssetName / FindSelfAssetURL / ApplySelfUpdate tests now live in
// selfupdate_test.go, which exercises the verified-self-update signatures.

// --- InvalidateXrayVersionCache ---

func TestInvalidateXrayVersionCache(t *testing.T) {
	// Should not panic
	InvalidateXrayVersionCache()
}

// --- RotateFile ---

func TestRotateFile_NoFile(t *testing.T) {
	// Should not panic for nonexistent file
	rotateFile("/nonexistent/path/to/file.log", 1024)
}

func TestRotateFile_SmallFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.log")

	if err := os.WriteFile(path, []byte("small"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// File is smaller than threshold, should not rotate
	rotateFile(path, 1024*1024)

	if _, err := os.Stat(path); err != nil {
		t.Error("original file should still exist after no rotation")
	}
	if _, err := os.Stat(path + ".1"); err == nil {
		t.Error("archive .1 should not exist (file was too small to rotate)")
	}
}

func TestRotateFile_LargeFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.log")

	// Write a file larger than threshold
	data := make([]byte, 2048)
	for i := range data {
		data[i] = 'x'
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	rotateFile(path, 1024) // threshold = 1024 bytes

	// Original should be renamed to .1
	if _, err := os.Stat(path + ".1"); err != nil {
		t.Error("archive .1 should exist after rotation")
	}
	// Original should be gone (renamed)
	if _, err := os.Stat(path); err == nil {
		t.Error("original file should be renamed after rotation")
	}
}

// --- GenerateXrayConfig variations ---

func TestGenerateXrayConfig_NonXHTTPNetwork(t *testing.T) {
	profile := &config.Profile{
		Name: "TCP Profile",
		Server: config.ServerConfig{
			Address:    "10.0.0.1",
			Port:       443,
			UUID:       "test-uuid",
			Encryption: "none",
			Transport: config.TransportConfig{
				Network: "tcp",
			},
			Security: config.SecurityConfig{
				Type: "none",
			},
		},
	}

	cfg, err := GenerateXrayConfig(profile, testSettings())
	if err != nil {
		t.Fatalf("GenerateXrayConfig() error = %v", err)
	}

	proxy := cfg.Outbounds[0]
	if proxy.StreamSettings.Network != "tcp" {
		t.Errorf("Network = %q, want tcp", proxy.StreamSettings.Network)
	}
	if proxy.StreamSettings.XHTTPSettings != nil {
		t.Error("XHTTPSettings should be nil for tcp network")
	}
	if proxy.StreamSettings.RealitySettings != nil {
		t.Error("RealitySettings should be nil for none security")
	}
}

func TestGenerateXrayConfig_XHTTPWithHost(t *testing.T) {
	profile := testProfile()
	profile.Server.Transport.Host = "cdn.example.com"

	cfg, err := GenerateXrayConfig(profile, testSettings())
	if err != nil {
		t.Fatalf("GenerateXrayConfig() error = %v", err)
	}

	proxy := cfg.Outbounds[0]
	if proxy.StreamSettings.XHTTPSettings == nil {
		t.Fatal("XHTTPSettings should not be nil")
	}
	if proxy.StreamSettings.XHTTPSettings.Host != "cdn.example.com" {
		t.Errorf("Host = %q, want cdn.example.com", proxy.StreamSettings.XHTTPSettings.Host)
	}
}

func TestGenerateXrayConfig_CustomDNS(t *testing.T) {
	settings := testSettings()
	settings.Local.DNS = []string{"1.1.1.1", "8.8.8.8", "9.9.9.9"}

	cfg, err := GenerateXrayConfig(testProfile(), settings)
	if err != nil {
		t.Fatalf("GenerateXrayConfig() error = %v", err)
	}

	if len(cfg.DNS.Servers) != 3 {
		t.Errorf("DNS servers count = %d, want 3", len(cfg.DNS.Servers))
	}
}

func TestGenerateXrayConfig_EncryptionAndFlow(t *testing.T) {
	profile := testProfile()
	profile.Server.Encryption = "none"
	profile.Server.Flow = "xtls-rprx-vision"

	cfg, err := GenerateXrayConfig(profile, testSettings())
	if err != nil {
		t.Fatalf("GenerateXrayConfig() error = %v", err)
	}

	proxy := cfg.Outbounds[0]
	var settings map[string]interface{}
	if err := json.Unmarshal(proxy.Settings, &settings); err != nil {
		t.Fatalf("unmarshal proxy settings: %v", err)
	}
	vnext := settings["vnext"].([]interface{})
	server := vnext[0].(map[string]interface{})
	users := server["users"].([]interface{})
	user := users[0].(map[string]interface{})

	if user["flow"] != "xtls-rprx-vision" {
		t.Errorf("user flow = %v, want xtls-rprx-vision", user["flow"])
	}
	if user["encryption"] != "none" {
		t.Errorf("user encryption = %v, want none", user["encryption"])
	}
}

// --- findFreePort ---

func TestFindFreePort(t *testing.T) {
	port, err := findFreePort()
	if err != nil {
		t.Fatalf("findFreePort() error = %v", err)
	}
	if port <= 0 || port > 65535 {
		t.Errorf("findFreePort() = %d, should be valid port", port)
	}
}

// --- isProcessAlive ---

func TestIsProcessAlive_CurrentProcess(t *testing.T) {
	pid := os.Getpid()
	if !isProcessAlive(pid) {
		t.Errorf("isProcessAlive(%d) should be true for current process", pid)
	}
}

func TestIsProcessAlive_InvalidPID(t *testing.T) {
	// PID 99999999 is very unlikely to exist
	if isProcessAlive(99999999) {
		t.Error("isProcessAlive(99999999) should be false")
	}
}

// --- XrayStatus/TrafficStats struct tests ---

func TestXrayStatusFields(t *testing.T) {
	status := &XrayStatus{
		Running: true,
		PID:     1234,
		Uptime:  90 * time.Minute,
		SocksOK: true,
		HTTPOK:  false,
	}

	if !status.Running {
		t.Error("Running should be true")
	}
	if status.PID != 1234 {
		t.Errorf("PID = %d, want 1234", status.PID)
	}
	if !status.SocksOK {
		t.Error("SocksOK should be true")
	}
	if status.HTTPOK {
		t.Error("HTTPOK should be false")
	}
}

func TestTrafficStatsFields(t *testing.T) {
	stats := &TrafficStats{
		Uplink:   1024 * 1024,
		Downlink: 10 * 1024 * 1024,
	}

	if stats.Uplink != 1024*1024 {
		t.Errorf("Uplink = %d, want %d", stats.Uplink, 1024*1024)
	}
	if stats.Downlink != 10*1024*1024 {
		t.Errorf("Downlink = %d, want %d", stats.Downlink, 10*1024*1024)
	}
}

// --- TunnelStatus struct tests ---

func TestTunnelStatusFields(t *testing.T) {
	status := TunnelStatus{
		Name:      "test-tunnel",
		Connected: true,
		PID:       5678,
		LocalPort: 8080,
		PanelURL:  "https://127.0.0.1:8080/panel",
	}

	if status.Name != "test-tunnel" {
		t.Errorf("Name = %q, want test-tunnel", status.Name)
	}
	if !status.Connected {
		t.Error("Connected should be true")
	}
	if status.LocalPort != 8080 {
		t.Errorf("LocalPort = %d, want 8080", status.LocalPort)
	}
}

// --- CheckResult struct tests ---

func TestCheckResultFields(t *testing.T) {
	result := CheckResult{
		Name:    "Latency",
		OK:      true,
		Detail:  "45ms",
		Latency: 45 * time.Millisecond,
	}

	if result.Name != "Latency" {
		t.Errorf("Name = %q, want Latency", result.Name)
	}
	if !result.OK {
		t.Error("OK should be true")
	}
	if result.Latency != 45*time.Millisecond {
		t.Errorf("Latency = %v, want 45ms", result.Latency)
	}
}

// --- ReleaseInfo/Asset struct tests ---

func TestReleaseInfoJSON(t *testing.T) {
	release := ReleaseInfo{
		TagName: "v1.8.24",
		Assets: []Asset{
			{Name: "test.zip", BrowserDownloadURL: "https://example.com/test.zip"},
		},
	}

	data, err := json.Marshal(release)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var parsed ReleaseInfo
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if parsed.TagName != "v1.8.24" {
		t.Errorf("TagName = %q, want v1.8.24", parsed.TagName)
	}
	if len(parsed.Assets) != 1 {
		t.Errorf("Assets count = %d, want 1", len(parsed.Assets))
	}
	if parsed.Assets[0].Name != "test.zip" {
		t.Errorf("Asset name = %q, want test.zip", parsed.Assets[0].Name)
	}
}

// --- StatsAPIPort ---

func TestStatsAPIPort(t *testing.T) {
	if StatsAPIPort != 10813 {
		t.Errorf("StatsAPIPort = %d, want 10813", StatsAPIPort)
	}
}

// --- XrayConfig struct JSON ---

func TestXrayConfigJSON(t *testing.T) {
	cfg := &XrayConfig{
		Log:       XrayLog{LogLevel: "info"},
		API:       &XrayAPI{Tag: "api", Services: []string{"StatsService"}},
		Stats:     &XrayStats{},
		Inbounds:  []Inbound{},
		Outbounds: []Outbound{},
		Routing:   RoutingConfig{DomainStrategy: "AsIs"},
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var parsed XrayConfig
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if parsed.Log.LogLevel != "info" {
		t.Errorf("LogLevel = %q, want info", parsed.Log.LogLevel)
	}
	if parsed.API.Tag != "api" {
		t.Errorf("API.Tag = %q, want api", parsed.API.Tag)
	}
}
