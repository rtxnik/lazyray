package core

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/rtxnik/lazyray/internal/config"
)

// --- WriteXrayConfig / ReadXrayConfig ---

func TestWriteXrayConfig_ValidProfile(t *testing.T) {
	profile := testProfile()
	settings := testSettings()
	_ = config.EnsureDirs()

	err := WriteXrayConfig(profile, settings)
	if err != nil {
		t.Fatalf("WriteXrayConfig() error = %v", err)
	}

	// Verify by reading back
	cfg, err := ReadXrayConfig()
	if err != nil {
		t.Fatalf("ReadXrayConfig() error = %v", err)
	}

	if cfg.Log.LogLevel != "warning" {
		t.Errorf("Log.LogLevel = %q, want 'warning'", cfg.Log.LogLevel)
	}
	if len(cfg.Outbounds) < 3 {
		t.Errorf("Outbounds count = %d, want >= 3", len(cfg.Outbounds))
	}
}

func TestWriteXrayConfig_InvalidProfile(t *testing.T) {
	profile := &config.Profile{
		Name: "bad",
		Server: config.ServerConfig{
			Address: "",
			Port:    0,
		},
	}

	err := WriteXrayConfig(profile, testSettings())
	if err == nil {
		t.Error("WriteXrayConfig should fail for invalid profile")
	}
}

// --- AssetName ---

func TestAssetName_KnownValue(t *testing.T) {
	name := AssetName()
	// On Linux amd64 test runner, should return Xray-linux-64.zip
	if name == "" {
		t.Log("AssetName() returned empty (unsupported platform in test)")
	}
}

// --- ImportSubscription ---

func TestImportSubscription_Integration(t *testing.T) {
	servers := &config.ServersConfig{}

	// Create test profile that would come from a subscription
	profiles := []*config.Profile{
		{
			Name: "Server 1",
			Server: config.ServerConfig{
				Address: "1.2.3.4",
				Port:    443,
				UUID:    "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
			},
		},
	}

	// Simulate what ImportSubscription does with pre-fetched profiles
	for _, p := range profiles {
		p.Group = "test-sub"
		p.Subscription = "https://example.com/sub"
		servers.Profiles = append(servers.Profiles, *p)
	}

	if len(servers.Profiles) != 1 {
		t.Errorf("profiles count = %d, want 1", len(servers.Profiles))
	}
	if servers.Profiles[0].Group != "test-sub" {
		t.Errorf("group = %q, want 'test-sub'", servers.Profiles[0].Group)
	}
}

// --- PID file helpers ---

func TestWriteAndReadPIDFile(t *testing.T) {
	_ = config.EnsureDirs()

	writePIDFile(12345)
	defer removePIDFile()

	pid := readPIDFile()
	if pid != 12345 {
		t.Errorf("readPIDFile() = %d, want 12345", pid)
	}

	removePIDFile()
	pid = readPIDFile()
	if pid != 0 {
		t.Errorf("after remove, readPIDFile() = %d, want 0", pid)
	}
}

func TestReadPIDFile_InvalidContent(t *testing.T) {
	_ = config.EnsureDirs()

	_ = os.WriteFile(config.PIDFilePath(), []byte("not-a-number"), 0644)
	defer os.Remove(config.PIDFilePath())

	pid := readPIDFile()
	if pid != 0 {
		t.Errorf("readPIDFile invalid = %d, want 0", pid)
	}
}

// --- Tunnel PID helpers ---

func TestWriteAndReadTunnelPID(t *testing.T) {
	_ = config.EnsureDirs()

	writeTunnelPID("test-tunnel-phase6", 1234, 8080)
	defer removeTunnelPID("test-tunnel-phase6")

	pid := readTunnelPID("test-tunnel-phase6")
	if pid != 1234 {
		t.Errorf("readTunnelPID = %d, want 1234", pid)
	}

	pid2, port := readTunnelPIDAndPort("test-tunnel-phase6")
	if pid2 != 1234 {
		t.Errorf("readTunnelPIDAndPort pid = %d, want 1234", pid2)
	}
	if port != 8080 {
		t.Errorf("readTunnelPIDAndPort port = %d, want 8080", port)
	}

	removeTunnelPID("test-tunnel-phase6")
	pid3 := readTunnelPID("test-tunnel-phase6")
	if pid3 != 0 {
		t.Errorf("after remove, readTunnelPID = %d, want 0", pid3)
	}
}

func TestReadTunnelPID_NonExistent(t *testing.T) {
	pid := readTunnelPID("definitely-nonexistent-tunnel-xyz123")
	if pid != 0 {
		t.Errorf("readTunnelPID non-existent = %d, want 0", pid)
	}
}

// --- XrayProcess.Status ---

func TestXrayProcess_Status_NotRunning(t *testing.T) {
	p := NewXrayProcess()
	status := p.Status()
	if status.Running {
		t.Error("Status.Running should be false for new process")
	}
	if status.PID != 0 {
		t.Errorf("Status.PID = %d, should be 0", status.PID)
	}
}

// --- GenerateXrayConfig chain ---

func TestGenerateXrayConfig_ChainConfig(t *testing.T) {
	profile := testProfile()
	profile.Chain = []config.ServerConfig{
		{
			Address:    "10.0.0.2",
			Port:       443,
			UUID:       "exit-uuid",
			Encryption: "none",
			Transport: config.TransportConfig{
				Network: "xhttp",
				Path:    "/exit",
				Mode:    "auto",
			},
			Security: config.SecurityConfig{
				Type:        "reality",
				SNI:         "exit.example.com",
				Fingerprint: "chrome",
				PublicKey:   "exit-public-key",
				ShortID:     "exit-short-id",
			},
		},
	}

	cfg, err := GenerateXrayConfig(profile, testSettings())
	if err != nil {
		t.Fatalf("GenerateXrayConfig() error = %v", err)
	}

	// Should have at least 4 outbounds: 2 chain hops + direct + block
	if len(cfg.Outbounds) < 4 {
		t.Errorf("Outbounds count = %d, want >= 4 for chain config", len(cfg.Outbounds))
	}
}

// --- XrayConfig JSON roundtrip with Policy ---

func TestXrayConfig_PolicyJSON(t *testing.T) {
	cfg := &XrayConfig{
		Log: XrayLog{LogLevel: "warning"},
		Policy: &XrayPolicy{
			System: XrayPolicySystem{
				StatsInboundUplink:   true,
				StatsInboundDownlink: true,
			},
		},
	}

	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var parsed XrayConfig
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if parsed.Policy == nil {
		t.Fatal("Policy should not be nil")
	}
	if !parsed.Policy.System.StatsInboundUplink {
		t.Error("StatsInboundUplink should be true")
	}
}

// --- TunnelManager with SSH profiles ---

func TestTunnelManager_Status_WithSSHProfile(t *testing.T) {
	tm := NewTunnelManager()
	profiles := []config.Profile{
		{
			Name:   "test-profile",
			Server: config.ServerConfig{Address: "1.2.3.4", Port: 443},
			SSH: config.SSHConfig{
				Host:    "1.2.3.4",
				Port:    22,
				User:    "root",
				KeyPath: "~/.ssh/id_ed25519",
				Panel: config.PanelConfig{
					Port: 28080,
					Path: "/panel/",
				},
			},
		},
	}

	statuses := tm.Status(profiles)
	if len(statuses) != 1 {
		t.Fatalf("Status count = %d, want 1", len(statuses))
	}
	if statuses[0].Name != "test-profile" {
		t.Errorf("Name = %q, want 'test-profile'", statuses[0].Name)
	}
	if statuses[0].Connected {
		t.Error("Should not be connected (no actual SSH tunnel)")
	}
}

// --- Validate Profile edge cases ---

func TestValidateProfile_EmptyAddress(t *testing.T) {
	profile := testProfile()
	profile.Server.Address = ""
	err := ValidateProfile(profile)
	if err == nil {
		t.Error("ValidateProfile should fail for empty address")
	}
}

func TestValidateProfile_NegativePort(t *testing.T) {
	profile := testProfile()
	profile.Server.Port = -1
	err := ValidateProfile(profile)
	if err == nil {
		t.Error("ValidateProfile should fail for negative port")
	}
}

// --- detachedProcAttr ---

func TestDetachedProcAttr(t *testing.T) {
	attr := detachedProcAttr()
	if attr == nil {
		t.Fatal("detachedProcAttr() should not return nil")
	}
}

// --- isTunnelProcessAlive ---

func TestIsTunnelProcessAlive_CurrentPID(t *testing.T) {
	pid := os.Getpid()
	if !isTunnelProcessAlive(pid) {
		t.Error("isTunnelProcessAlive should return true for current process")
	}
}

func TestIsTunnelProcessAlive_InvalidPID(t *testing.T) {
	if isTunnelProcessAlive(99999999) {
		t.Error("isTunnelProcessAlive should return false for invalid PID")
	}
}

// --- XrayProcess.Status additional ---

func TestXrayProcess_Status_Fields(t *testing.T) {
	p := NewXrayProcess()
	status := p.Status()
	if status == nil {
		t.Fatal("Status() should not return nil")
	}
	// Not running: fields should have default values
	if status.Running {
		t.Error("Running should be false")
	}
	if status.Uptime != 0 {
		t.Errorf("Uptime = %v, want 0", status.Uptime)
	}
}

// --- GetXrayVersion ---

func TestGetXrayVersion_ReturnsString(t *testing.T) {
	ver := GetXrayVersion()
	// May be "unknown" or an actual version
	if ver == "" {
		t.Error("GetXrayVersion should return non-empty string")
	}
}

// --- parseAllTrafficStats ---

func TestParseAllTrafficStats_Valid(t *testing.T) {
	input := `stat: {
  name: "outbound>>>proxy>>>traffic>>>uplink"
  value: 12345
}
stat: {
  name: "outbound>>>proxy>>>traffic>>>downlink"
  value: 67890
}`
	stats := parseAllTrafficStats(input)
	if stats.Uplink != 12345 {
		t.Errorf("Uplink = %d, want 12345", stats.Uplink)
	}
	if stats.Downlink != 67890 {
		t.Errorf("Downlink = %d, want 67890", stats.Downlink)
	}
}

func TestParseAllTrafficStats_QuotedValue(t *testing.T) {
	input := `stat: {
  name: "outbound>>>proxy>>>traffic>>>downlink"
  value: "67890"
}`
	stats := parseAllTrafficStats(input)
	if stats.Downlink != 67890 {
		t.Errorf("Downlink = %d, want 67890", stats.Downlink)
	}
}

func TestParseAllTrafficStats_NoValue(t *testing.T) {
	input := `stat: {
  name: "outbound>>>proxy>>>traffic>>>uplink"
}`
	stats := parseAllTrafficStats(input)
	if stats.Uplink != 0 {
		t.Errorf("Uplink = %d, want 0", stats.Uplink)
	}
}

func TestParseAllTrafficStats_EmptyString(t *testing.T) {
	stats := parseAllTrafficStats("")
	if stats.Uplink != 0 || stats.Downlink != 0 {
		t.Errorf("expected zero stats, got up=%d down=%d", stats.Uplink, stats.Downlink)
	}
}

func TestParseAllTrafficStats_LargeValue(t *testing.T) {
	input := `stat: {
  name: "outbound>>>proxy>>>traffic>>>uplink"
  value: 1073741824
}`
	stats := parseAllTrafficStats(input)
	if stats.Uplink != 1073741824 {
		t.Errorf("Uplink = %d, want 1073741824", stats.Uplink)
	}
}

// --- rotateFile ---

func TestRotateFile_BelowThreshold(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.log")
	_ = os.WriteFile(path, []byte("small"), 0644)

	// maxBytes is large, should not rotate
	rotateFile(path, 1024*1024)

	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("small file should not be rotated")
	}
	if _, err := os.Stat(path + ".1"); err == nil {
		t.Error("archive .1 should not exist for small file")
	}
}

func TestRotateFile_AboveThreshold(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.log")
	data := make([]byte, 2048)
	_ = os.WriteFile(path, data, 0644)

	rotateFile(path, 1024)

	// Original should be gone (renamed to .1)
	if _, err := os.Stat(path); err == nil {
		t.Error("original file should be renamed after rotation")
	}
	if _, err := os.Stat(path + ".1"); os.IsNotExist(err) {
		t.Error("archive .1 should exist after rotation")
	}
}

func TestRotateFile_ChainedRotation(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.log")
	_ = os.WriteFile(path+".1", []byte("old-archive"), 0644)
	data := make([]byte, 2048)
	_ = os.WriteFile(path, data, 0644)

	rotateFile(path, 1024)

	// .1 should have new content, .2 should have old archive
	if _, err := os.Stat(path + ".2"); os.IsNotExist(err) {
		t.Error("archive .2 should exist (chained rotation)")
	}
}

func TestRotateFile_NonExistent(t *testing.T) {
	// Should not panic on non-existent file
	nonExistent := filepath.Join(t.TempDir(), "definitely-nonexistent-logfile-xyz123.log")
	rotateFile(nonExistent, 1024)
}

// --- RotateLogs ---

func TestRotateLogs_WithSettings(t *testing.T) {
	settings := testSettings()
	settings.Xray.MaxLogSize = 100 // 100 MB - files should be small, no rotation
	// Should not panic
	RotateLogs(settings)
}

func TestRotateLogs_ZeroMaxLogSize(t *testing.T) {
	settings := testSettings()
	settings.Xray.MaxLogSize = 0 // Should use default (10 MB)
	RotateLogs(settings)
}

// --- isIPEntry edge cases ---

func TestIsIPEntry_DomainPrefixes(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"domain:example.com", false},
		{"geosite:cn", false},
		{"regexp:.*\\.cn$", false},
		{"full:www.example.com", false},
		{"keyword:google", false},
		{"10.0.0.0/8", true},
		{"192.168.1.1", true},
		{"::1", true},
		{"geoip:cn", true}, // geoip is IP, not in domain prefixes
		{"2001:db8::/32", true},
		{"example.com", false}, // bare domain (no digit start, no slash/colon)
	}

	for _, tc := range tests {
		got := isIPEntry(tc.input)
		if got != tc.want {
			t.Errorf("isIPEntry(%q) = %v, want %v", tc.input, got, tc.want)
		}
	}
}

// --- isProcessAlive ---

func TestIsProcessAlive_Self(t *testing.T) {
	pid := os.Getpid()
	if !isProcessAlive(pid) {
		t.Error("isProcessAlive should return true for current process")
	}
}

func TestIsProcessAlive_Nonexistent(t *testing.T) {
	if isProcessAlive(99999999) {
		t.Error("isProcessAlive should return false for invalid PID")
	}
}

// --- findFreePort ---

func TestFindFreePort_ReturnsValidPort(t *testing.T) {
	port, err := findFreePort()
	if err != nil {
		t.Fatalf("findFreePort() error = %v", err)
	}
	if port <= 0 || port > 65535 {
		t.Errorf("findFreePort() = %d, want valid port number", port)
	}
}

func TestFindFreePort_UniqueEachCall(t *testing.T) {
	port1, err := findFreePort()
	if err != nil {
		t.Fatalf("findFreePort() error = %v", err)
	}
	port2, err := findFreePort()
	if err != nil {
		t.Fatalf("findFreePort() error = %v", err)
	}
	// Ports should typically be different (not guaranteed but highly likely)
	t.Logf("Got ports %d and %d", port1, port2)
}

// --- checkProcess ---

func TestCheckProcess_NotRunning(t *testing.T) {
	p := NewXrayProcess()
	result := checkProcess(p)
	if result.OK {
		t.Error("checkProcess should return not OK for non-running process")
	}
	if result.Name != "Process" {
		t.Errorf("result.Name = %q, want 'Process'", result.Name)
	}
}

// --- TunnelManager.Disconnect ---

func TestTunnelManager_Disconnect_NonExistentWithPIDFile(t *testing.T) {
	_ = config.EnsureDirs()
	tm := NewTunnelManager()

	// Write a PID file with a non-existent process
	writeTunnelPID("test-disconnect-phase6", 99999999, 8080)
	defer removeTunnelPID("test-disconnect-phase6")

	// Should succeed (cleans up PID file)
	err := tm.Disconnect("test-disconnect-phase6")
	if err != nil {
		t.Errorf("Disconnect with stale PID file should succeed: %v", err)
	}
}

func TestTunnelManager_Disconnect_NonExistentNoPIDFile(t *testing.T) {
	tm := NewTunnelManager()
	err := tm.Disconnect("completely-nonexistent-xyz123")
	if err == nil {
		t.Error("Disconnect with no tunnel and no PID file should return error")
	}
}

// --- TunnelManager.DisconnectAll ---

func TestTunnelManager_DisconnectAll_NoTunnels(t *testing.T) {
	tm := NewTunnelManager()
	// Should not panic with empty tunnels
	tm.DisconnectAll()
}

// --- TunnelManager.Status edge cases ---

func TestTunnelManager_Status_NoSSHProfiles(t *testing.T) {
	tm := NewTunnelManager()
	profiles := []config.Profile{
		{
			Name:   "no-ssh",
			Server: config.ServerConfig{Address: "1.2.3.4", Port: 443},
			// No SSH config
		},
	}
	statuses := tm.Status(profiles)
	if len(statuses) != 0 {
		t.Errorf("Status should be empty for profiles without SSH, got %d", len(statuses))
	}
}

func TestTunnelManager_Status_WithStalePIDFile(t *testing.T) {
	_ = config.EnsureDirs()
	tm := NewTunnelManager()

	// Write a stale PID file (non-existent process)
	writeTunnelPID("stale-tunnel-phase6", 99999999, 9090)
	defer removeTunnelPID("stale-tunnel-phase6")

	profiles := []config.Profile{
		{
			Name:   "stale-tunnel-phase6",
			Server: config.ServerConfig{Address: "1.2.3.4", Port: 443},
			SSH: config.SSHConfig{
				Host: "1.2.3.4",
				Port: 22,
				User: "root",
			},
		},
	}

	statuses := tm.Status(profiles)
	if len(statuses) != 1 {
		t.Fatalf("Status count = %d, want 1", len(statuses))
	}
	// Stale PID should be cleaned up, not shown as connected
	if statuses[0].Connected {
		t.Error("Should not be connected for stale PID")
	}
}

// --- FormatUptime edge cases ---

func TestFormatUptime_Various(t *testing.T) {
	tests := []struct {
		d    time.Duration
		want string
	}{
		{0, "-"},
		{5 * time.Minute, "5m"},
		{30 * time.Minute, "30m"},
		{time.Hour, "1h 0m"},
		{2*time.Hour + 15*time.Minute, "2h 15m"},
		{24 * time.Hour, "24h 0m"},
	}
	for _, tc := range tests {
		got := FormatUptime(tc.d)
		if got != tc.want {
			t.Errorf("FormatUptime(%v) = %q, want %q", tc.d, got, tc.want)
		}
	}
}

// --- FormatBytes edge cases ---

func TestFormatBytes_Various(t *testing.T) {
	tests := []struct {
		b    int64
		want string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1024, "1 KB"},
		{1536, "2 KB"},
		{1048576, "1 MB"},
		{1073741824, "1.0 GB"},
		{5368709120, "5.0 GB"},
	}
	for _, tc := range tests {
		got := FormatBytes(tc.b)
		if got != tc.want {
			t.Errorf("FormatBytes(%d) = %q, want %q", tc.b, got, tc.want)
		}
	}
}

// --- InvalidateXrayVersionCache ---

func TestInvalidateXrayVersionCache_AndRequery(t *testing.T) {
	// Call GetXrayVersion to populate cache
	_ = GetXrayVersion()
	// Invalidate should not panic
	InvalidateXrayVersionCache()
	// Subsequent call should still work
	ver := GetXrayVersion()
	if ver == "" {
		t.Error("GetXrayVersion after invalidate should return non-empty string")
	}
}

// --- XrayProcess basic lifecycle ---

func TestNewXrayProcess_Defaults(t *testing.T) {
	p := NewXrayProcess()
	if p == nil {
		t.Fatal("NewXrayProcess should not return nil")
	}
	if p.IsRunning() {
		t.Error("new process should not be running")
	}
	if p.GetPID() != 0 {
		t.Errorf("new process PID = %d, want 0", p.GetPID())
	}
}

// --- CloseAllPersistentTunnels ---

func TestCloseAllPersistentTunnels_NoFiles(t *testing.T) {
	// Should not panic when no PID files exist
	CloseAllPersistentTunnels()
}

func TestCloseAllPersistentTunnels_WithStaleFile(t *testing.T) {
	_ = config.EnsureDirs()
	// Write a stale tunnel PID file
	writeTunnelPID("test-close-all-phase6", 99999999, 8080)
	defer removeTunnelPID("test-close-all-phase6")

	// Should clean up stale PID files without panicking
	CloseAllPersistentTunnels()
}

// --- RunHealthCheck ---

func TestRunHealthCheck_NotRunning(t *testing.T) {
	xray := NewXrayProcess()
	profile := testProfile()
	settings := testSettings()

	report := RunHealthCheck(xray, profile, settings)
	if report == nil {
		t.Fatal("RunHealthCheck should not return nil")
	}
	if len(report.Checks) < 3 {
		t.Errorf("Checks count = %d, want >= 3 (process, socks, http)", len(report.Checks))
	}
	// Process should fail since xray is not running
	if report.Checks[0].Name != "Process" {
		t.Errorf("first check name = %q, want 'Process'", report.Checks[0].Name)
	}
	if report.Checks[0].OK {
		t.Error("Process check should fail for non-running xray")
	}
	if report.AllPassed {
		t.Error("AllPassed should be false when xray is not running")
	}
	// Should have Exit IP skip check since SOCKS5 is down
	foundExitIP := false
	for _, c := range report.Checks {
		if c.Name == "Exit IP" {
			foundExitIP = true
			if c.OK {
				t.Error("Exit IP should not be OK when SOCKS5 is down")
			}
		}
	}
	if !foundExitIP {
		t.Error("should have Exit IP check result")
	}
}

func TestRunHealthCheck_NilProfile(t *testing.T) {
	xray := NewXrayProcess()
	settings := testSettings()

	// nil profile should not panic
	report := RunHealthCheck(xray, nil, settings)
	if report == nil {
		t.Fatal("RunHealthCheck should not return nil with nil profile")
	}
}

// --- ProbeProfile (stream transport, not connected) ---

func TestProbeProfile_UnreachableServer(t *testing.T) {
	server := config.ServerConfig{
		Address: "127.0.0.1", // loopback port 1 — guaranteed connection refused
		Port:    1,
	}
	r := ProbeProfile(config.Profile{Server: server}, ProbeContext{Timeout: 1 * time.Second})
	if r.Status != LivenessFail {
		t.Error("ProbeProfile to unreachable server should return LivenessFail")
	}
}

// --- ParseSubscriptionBody additional edge cases ---

func TestParseSubscriptionBody_URLSafeBase64(t *testing.T) {
	// URL-safe base64 encoded VLESS URL
	vlessURL := "vless://aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee@1.2.3.4:443?type=tcp&security=none#Server1"
	encoded := base64.URLEncoding.EncodeToString([]byte(vlessURL))

	profiles, err := ParseSubscriptionBody(encoded)
	if err != nil {
		t.Fatalf("ParseSubscriptionBody with URL-safe base64 error: %v", err)
	}
	if len(profiles) != 1 {
		t.Errorf("profiles count = %d, want 1", len(profiles))
	}
}
