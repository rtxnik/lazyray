package core

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/rtxnik/lazyray/internal/config"
)

// setTestHome overrides the home/data directory for config paths.
// On Windows, DataDir uses LOCALAPPDATA and ConfigDir uses APPDATA;
// on Unix, both use HOME.
func setTestHome(t *testing.T, dir string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Setenv("LOCALAPPDATA", dir)
		t.Setenv("APPDATA", dir)
	} else {
		t.Setenv("HOME", dir)
	}
}

// --- FormatSpeedTestResult tests ---

func TestFormatSpeedTestResult_Success(t *testing.T) {
	result := &SpeedTestResult{
		Downloaded: 10485760, // 10 MB
		Duration:   10 * time.Second,
		SpeedMbps:  8.39,
	}

	output := FormatSpeedTestResult(result)
	if !strings.Contains(output, "Mbps") {
		t.Errorf("output should contain Mbps: %q", output)
	}
	if !strings.Contains(output, "10.0") {
		t.Errorf("output should contain duration: %q", output)
	}
}

func TestFormatSpeedTestResult_Error(t *testing.T) {
	result := &SpeedTestResult{
		Error: fmt.Errorf("connection timeout"),
	}

	output := FormatSpeedTestResult(result)
	if !strings.Contains(output, "failed") {
		t.Errorf("output should contain 'failed': %q", output)
	}
	if !strings.Contains(output, "connection timeout") {
		t.Errorf("output should contain error message: %q", output)
	}
}

// --- hasDNSEncryption tests ---

func TestHasDNSEncryption_DoH(t *testing.T) {
	servers := []string{"https://dns.google/dns-query", "8.8.8.8"}
	if !hasDNSEncryption(servers) {
		t.Error("should detect DoH as encrypted")
	}
}

func TestHasDNSEncryption_DoT(t *testing.T) {
	servers := []string{"tcp://dns.google:853"}
	if !hasDNSEncryption(servers) {
		t.Error("should detect DoT as encrypted")
	}
}

func TestHasDNSEncryption_DoHLocal(t *testing.T) {
	servers := []string{"https+local://dns.google/dns-query"}
	if !hasDNSEncryption(servers) {
		t.Error("should detect DoH local as encrypted")
	}
}

func TestHasDNSEncryption_PlainOnly(t *testing.T) {
	servers := []string{"1.1.1.1", "8.8.8.8"}
	if hasDNSEncryption(servers) {
		t.Error("plain DNS should not be encrypted")
	}
}

func TestHasDNSEncryption_Empty(t *testing.T) {
	if hasDNSEncryption(nil) {
		t.Error("empty servers should not be encrypted")
	}
}

// --- FormatStatsReport tests ---

func TestFormatStatsReport_Empty(t *testing.T) {
	history := StatsHistory{}
	output := FormatStatsReport(history)
	if !strings.Contains(output, "Today:") {
		t.Errorf("should contain Today: %q", output)
	}
}

func TestFormatStatsReport_WithData(t *testing.T) {
	today := time.Now().Format("2006-01-02")
	history := StatsHistory{
		Days: []DailyStats{
			{Date: today, Uplink: 1024 * 1024, Downlink: 10 * 1024 * 1024},
		},
		TotalUplink:   1024 * 1024,
		TotalDownlink: 10 * 1024 * 1024,
	}

	output := FormatStatsReport(history)
	if !strings.Contains(output, "Today:") {
		t.Errorf("should contain Today: %q", output)
	}
	if !strings.Contains(output, "This month:") {
		t.Errorf("should contain This month: %q", output)
	}
	if !strings.Contains(output, "All time:") {
		t.Errorf("should contain All time: %q", output)
	}
}

func TestFormatStatsReport_MultiplesDays(t *testing.T) {
	today := time.Now().Format("2006-01-02")
	history := StatsHistory{
		Days: []DailyStats{
			{Date: "2026-01-01", Uplink: 100, Downlink: 200},
			{Date: "2026-01-02", Uplink: 300, Downlink: 400},
			{Date: today, Uplink: 500, Downlink: 600},
		},
		TotalUplink:   900,
		TotalDownlink: 1200,
	}

	output := FormatStatsReport(history)
	if !strings.Contains(output, "Last 3 days") {
		t.Errorf("should contain day count: %q", output)
	}
}

// --- StatsManager tests ---

func TestStatsManager_RecordTraffic_NilStats(t *testing.T) {
	sm := &StatsManager{
		history: &StatsHistory{},
	}

	// Should not panic
	sm.RecordTraffic(nil)
}

func TestStatsManager_RecordTraffic_FirstCall(t *testing.T) {
	sm := &StatsManager{
		history: &StatsHistory{},
	}

	sm.RecordTraffic(&TrafficStats{Uplink: 1000, Downlink: 2000})

	if !sm.initialized {
		t.Error("should be initialized after first call")
	}
	if sm.lastUplink != 1000 {
		t.Errorf("lastUplink = %d, want 1000", sm.lastUplink)
	}
}

func TestStatsManager_RecordTraffic_Delta(t *testing.T) {
	sm := &StatsManager{
		history: &StatsHistory{},
	}

	// First call — baseline
	sm.RecordTraffic(&TrafficStats{Uplink: 1000, Downlink: 2000})
	// Second call — with delta
	sm.RecordTraffic(&TrafficStats{Uplink: 2000, Downlink: 5000})

	if len(sm.history.Days) != 1 {
		t.Fatalf("Days count = %d, want 1", len(sm.history.Days))
	}

	day := sm.history.Days[0]
	if day.Uplink != 1000 {
		t.Errorf("Uplink = %d, want 1000", day.Uplink)
	}
	if day.Downlink != 3000 {
		t.Errorf("Downlink = %d, want 3000", day.Downlink)
	}
}

func TestStatsManager_RecordTraffic_CounterReset(t *testing.T) {
	sm := &StatsManager{
		history:      &StatsHistory{},
		initialized:  true,
		lastUplink:   5000,
		lastDownlink: 8000,
	}

	// Counter reset (xray restart) — values lower than last seen
	sm.RecordTraffic(&TrafficStats{Uplink: 500, Downlink: 1000})

	if len(sm.history.Days) != 1 {
		t.Fatalf("Days count = %d, want 1", len(sm.history.Days))
	}
	// On counter reset, delta = current value (not negative)
	if sm.history.Days[0].Uplink != 500 {
		t.Errorf("Uplink = %d, want 500 (counter reset)", sm.history.Days[0].Uplink)
	}
}

func TestStatsManager_RecordTraffic_NoDelta(t *testing.T) {
	sm := &StatsManager{
		history:      &StatsHistory{},
		initialized:  true,
		lastUplink:   1000,
		lastDownlink: 2000,
	}

	// Same values — no delta
	sm.RecordTraffic(&TrafficStats{Uplink: 1000, Downlink: 2000})

	if len(sm.history.Days) != 0 {
		t.Errorf("Days should be empty with no delta, got %d", len(sm.history.Days))
	}
}

func TestStatsManager_TodayStats(t *testing.T) {
	today := time.Now().Format("2006-01-02")
	sm := &StatsManager{
		history: &StatsHistory{
			Days: []DailyStats{
				{Date: today, Uplink: 100, Downlink: 200},
			},
		},
	}

	stats := sm.TodayStats()
	if stats.Uplink != 100 {
		t.Errorf("Uplink = %d, want 100", stats.Uplink)
	}
	if stats.Downlink != 200 {
		t.Errorf("Downlink = %d, want 200", stats.Downlink)
	}
}

func TestStatsManager_TodayStats_NoData(t *testing.T) {
	sm := &StatsManager{
		history: &StatsHistory{},
	}

	stats := sm.TodayStats()
	if stats.Uplink != 0 || stats.Downlink != 0 {
		t.Errorf("should be zero with no data")
	}
}

func TestStatsManager_MonthStats(t *testing.T) {
	prefix := time.Now().Format("2006-01")
	sm := &StatsManager{
		history: &StatsHistory{
			Days: []DailyStats{
				{Date: prefix + "-01", Uplink: 100, Downlink: 200},
				{Date: prefix + "-15", Uplink: 300, Downlink: 400},
				{Date: "2020-01-01", Uplink: 999, Downlink: 999},
			},
		},
	}

	stats := sm.MonthStats()
	if stats.Uplink != 400 {
		t.Errorf("Month Uplink = %d, want 400", stats.Uplink)
	}
	if stats.Downlink != 600 {
		t.Errorf("Month Downlink = %d, want 600", stats.Downlink)
	}
}

func TestStatsManager_GetHistory(t *testing.T) {
	sm := &StatsManager{
		history: &StatsHistory{
			Days: []DailyStats{
				{Date: "2026-01-01", Uplink: 100, Downlink: 200},
			},
			TotalUplink:   100,
			TotalDownlink: 200,
		},
	}

	h := sm.GetHistory()
	if len(h.Days) != 1 {
		t.Fatalf("Days count = %d, want 1", len(h.Days))
	}
	if h.TotalUplink != 100 {
		t.Errorf("TotalUplink = %d, want 100", h.TotalUplink)
	}

	// Verify it's a copy (modifying doesn't affect original)
	h.Days[0].Uplink = 999
	if sm.history.Days[0].Uplink == 999 {
		t.Error("GetHistory should return a copy")
	}
}

func TestStatsManager_PruneOldEntries(t *testing.T) {
	days := make([]DailyStats, 100)
	for i := range days {
		days[i] = DailyStats{Date: fmt.Sprintf("2026-01-%02d", i+1)}
	}

	sm := &StatsManager{
		history: &StatsHistory{Days: days},
	}

	sm.pruneOldEntries(10)
	if len(sm.history.Days) != 10 {
		t.Errorf("after prune, Days count = %d, want 10", len(sm.history.Days))
	}
}

// --- parseHostPort (shadowsocks.go) tests ---

func TestParseHostPort_SS_Regular(t *testing.T) {
	host, port, err := parseHostPort("1.2.3.4:8388")
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if host != "1.2.3.4" {
		t.Errorf("host = %q, want 1.2.3.4", host)
	}
	if port != 8388 {
		t.Errorf("port = %d, want 8388", port)
	}
}

func TestParseHostPort_SS_IPv6(t *testing.T) {
	host, port, err := parseHostPort("[::1]:8388")
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if host != "::1" {
		t.Errorf("host = %q, want ::1", host)
	}
	if port != 8388 {
		t.Errorf("port = %d, want 8388", port)
	}
}

func TestParseHostPort_SS_MissingPort(t *testing.T) {
	_, _, err := parseHostPort("1.2.3.4")
	if err == nil {
		t.Error("expected error for missing port")
	}
}

func TestParseHostPort_SS_InvalidPort(t *testing.T) {
	_, _, err := parseHostPort("1.2.3.4:abc")
	if err == nil {
		t.Error("expected error for invalid port")
	}
}

func TestParseHostPort_SS_IPv6_UnclosedBracket(t *testing.T) {
	_, _, err := parseHostPort("[::1:8388")
	if err == nil {
		t.Error("expected error for unclosed bracket")
	}
}

func TestParseHostPort_SS_IPv6_MissingPort(t *testing.T) {
	_, _, err := parseHostPort("[::1]")
	if err == nil {
		t.Error("expected error for IPv6 without port")
	}
}

// --- decodeBase64Any tests ---

func TestTryBase64Decode_URLSafe(t *testing.T) {
	input := "YWVzLTI1Ni1nY206cGFzc3dvcmQ" // aes-256-gcm:password (URL-safe no padding)
	got, err := decodeBase64Any(input)
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if string(got) != "aes-256-gcm:password" {
		t.Errorf("decoded = %q, want aes-256-gcm:password", string(got))
	}
}

func TestTryBase64Decode_Standard(t *testing.T) {
	input := "YWVzLTI1Ni1nY206cGFzc3dvcmQ=" // standard with padding
	got, err := decodeBase64Any(input)
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if string(got) != "aes-256-gcm:password" {
		t.Errorf("decoded = %q, want aes-256-gcm:password", string(got))
	}
}

func TestTryBase64Decode_Invalid(t *testing.T) {
	_, err := decodeBase64Any("!!!not-base64!!!")
	if err == nil {
		t.Error("expected error for invalid base64")
	}
}

// --- FormatBytes extended tests ---

func TestFormatBytes_Extended(t *testing.T) {
	tests := []struct {
		bytes int64
		want  string
	}{
		{0, "0 B"},
		{1023, "1023 B"},
		{1024, "1 KB"},
		{1048576, "1 MB"},
		{1073741824, "1.0 GB"},
		{2147483648, "2.0 GB"},
	}

	for _, tc := range tests {
		t.Run(tc.want, func(t *testing.T) {
			got := FormatBytes(tc.bytes)
			if got != tc.want {
				t.Errorf("FormatBytes(%d) = %q, want %q", tc.bytes, got, tc.want)
			}
		})
	}
}

// --- DailyStats / StatsHistory JSON ---

func TestDailyStats_Fields(t *testing.T) {
	d := DailyStats{
		Date:     "2026-02-27",
		Uplink:   1024,
		Downlink: 2048,
	}
	if d.Date != "2026-02-27" || d.Uplink != 1024 || d.Downlink != 2048 {
		t.Error("DailyStats fields mismatch")
	}
}

func TestSpeedTestResult_Fields(t *testing.T) {
	r := SpeedTestResult{
		Downloaded: 5242880,
		Duration:   5 * time.Second,
		SpeedMbps:  8.39,
	}
	if r.Downloaded != 5242880 || r.SpeedMbps != 8.39 {
		t.Error("SpeedTestResult fields mismatch")
	}
}

// --- SelfAssetName / FindSelfAssetURL tests ---
// These now live in selfupdate_test.go, which exercises the verified
// verified-self-update signatures (SelfAssetName(version) + SelfAssetURLs).

// --- CheckXrayVersionCompat tests ---

func TestCheckXrayVersionCompat_8C(t *testing.T) {
	// CheckXrayVersionCompat() takes no args and returns a warning string
	// (empty if OK, non-empty if xray is outdated/missing).
	result := CheckXrayVersionCompat()
	// Just exercise the function; result depends on environment
	_ = result
}

// --- compareVersions tests ---

func TestCompareVersions_8C(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"1.0.0", "1.0.0", 0},
		{"1.0.1", "1.0.0", 1},
		{"1.0.0", "1.0.1", -1},
		{"2.0.0", "1.9.9", 1},
		{"1.8.24", "1.8.0", 1},
		{"1.8.0", "1.8.24", -1},
	}

	for _, tc := range tests {
		t.Run(tc.a+"_vs_"+tc.b, func(t *testing.T) {
			got := compareVersions(tc.a, tc.b)
			if got != tc.want {
				t.Errorf("compareVersions(%q, %q) = %d, want %d", tc.a, tc.b, got, tc.want)
			}
		})
	}
}

// --- toInt64 tests ---

func TestToInt64(t *testing.T) {
	tests := []struct {
		name string
		val  interface{}
		want int64
	}{
		{"string", "12345", 12345},
		{"float64", float64(42), 42},
		{"int", nil, 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := toInt64(tc.val)
			if got != tc.want {
				t.Errorf("toInt64(%v) = %d, want %d", tc.val, got, tc.want)
			}
		})
	}
}

// --- HealthReportJSON tests ---

func TestHealthReportJSON_8C(t *testing.T) {
	report := &HealthReport{
		Checks: []CheckResult{
			{Name: "Process", OK: true, Detail: "xray running (PID 1234)"},
			{Name: "SOCKS5", OK: true, Detail: "127.0.0.1:10808 accepting"},
			{Name: "HTTP", OK: false, Detail: "127.0.0.1:10809 not accepting"},
		},
		AllPassed: false,
		Timestamp: time.Date(2026, 2, 27, 12, 0, 0, 0, time.UTC),
	}

	jsonStr, err := HealthReportJSON(report)
	if err != nil {
		t.Fatalf("HealthReportJSON() error = %v", err)
	}
	if !strings.Contains(jsonStr, "Process") {
		t.Error("JSON should contain check name")
	}
	if !strings.Contains(jsonStr, "AllPassed") || !strings.Contains(jsonStr, "false") {
		t.Error("JSON should contain AllPassed: false")
	}
}

func TestHealthReportJSON_Empty(t *testing.T) {
	report := &HealthReport{
		AllPassed: true,
		Timestamp: time.Now(),
	}
	jsonStr, err := HealthReportJSON(report)
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if jsonStr == "" {
		t.Error("should not be empty")
	}
}

// --- checkProcess tests ---

func TestCheckProcess_NotRunning_8C(t *testing.T) {
	xray := NewXrayProcess()
	result := checkProcess(xray)
	if result.OK {
		t.Error("checkProcess should return false when xray not running")
	}
	if result.Name != "Process" {
		t.Errorf("Name = %q, want Process", result.Name)
	}
	if !strings.Contains(result.Detail, "not running") {
		t.Errorf("Detail = %q, should mention not running", result.Detail)
	}
}

// --- checkPort tests ---

func TestCheckPort_Unavailable(t *testing.T) {
	result := checkPort("SOCKS5", "127.0.0.1", 59999, 1)
	if result.OK {
		t.Error("checkPort should fail for unavailable port")
	}
	if !strings.Contains(result.Detail, "not accepting") {
		t.Errorf("Detail = %q, should mention not accepting", result.Detail)
	}
}

func TestCheckPort_ZeroTimeout_8C(t *testing.T) {
	result := checkPort("HTTP", "127.0.0.1", 59998, 0)
	if result.OK {
		t.Error("checkPort should fail for unavailable port even with zero timeout")
	}
}

// --- ProbeProfile (stream transport, not connected) ---

func TestProbeProfile_Unavailable(t *testing.T) {
	server := config.ServerConfig{
		Address: "127.0.0.1",
		Port:    59991, // high port, unlikely to be listening
	}
	r := ProbeProfile(config.Profile{Server: server}, ProbeContext{Timeout: 1 * time.Second})
	if r.Status != LivenessFail {
		t.Error("ProbeProfile should return LivenessFail for unreachable server")
	}
}

// --- parseAllTrafficStats / parseStatsJSON / parseStatsText tests ---

func TestParseStatsJSON_Valid(t *testing.T) {
	input := `{"stat":[{"name":"outbound>>>proxy>>>traffic>>>uplink","value":"12345"},{"name":"outbound>>>proxy>>>traffic>>>downlink","value":"67890"}]}`
	stats := parseStatsJSON(input)
	if stats == nil {
		t.Fatal("should parse valid JSON stats")
	}
	if stats.Uplink != 12345 {
		t.Errorf("Uplink = %d, want 12345", stats.Uplink)
	}
	if stats.Downlink != 67890 {
		t.Errorf("Downlink = %d, want 67890", stats.Downlink)
	}
}

func TestParseStatsJSON_Float64Values(t *testing.T) {
	input := `{"stat":[{"name":"outbound>>>proxy>>>traffic>>>uplink","value":1000.0},{"name":"outbound>>>proxy>>>traffic>>>downlink","value":2000.0}]}`
	stats := parseStatsJSON(input)
	if stats == nil {
		t.Fatal("should parse JSON with float64 values")
	}
	if stats.Uplink != 1000 {
		t.Errorf("Uplink = %d, want 1000", stats.Uplink)
	}
}

func TestParseStatsJSON_Invalid(t *testing.T) {
	stats := parseStatsJSON("not json")
	if stats != nil {
		t.Error("should return nil for invalid JSON")
	}
}

func TestParseStatsJSON_NonProxyEntries(t *testing.T) {
	input := `{"stat":[{"name":"outbound>>>direct>>>traffic>>>uplink","value":"999"},{"name":"outbound>>>proxy>>>traffic>>>uplink","value":"100"}]}`
	stats := parseStatsJSON(input)
	if stats == nil {
		t.Fatal("should parse JSON")
	}
	if stats.Uplink != 100 {
		t.Errorf("Uplink = %d, want 100 (only proxy)", stats.Uplink)
	}
}

func TestParseStatsText_Valid(t *testing.T) {
	input := `stat: {
  name: "outbound>>>proxy>>>traffic>>>uplink"
  value: 12345
}
stat: {
  name: "outbound>>>proxy>>>traffic>>>downlink"
  value: 67890
}`
	stats := parseStatsText(input)
	if stats.Uplink != 12345 {
		t.Errorf("Uplink = %d, want 12345", stats.Uplink)
	}
	if stats.Downlink != 67890 {
		t.Errorf("Downlink = %d, want 67890", stats.Downlink)
	}
}

func TestParseStatsText_Empty(t *testing.T) {
	stats := parseStatsText("")
	if stats.Uplink != 0 || stats.Downlink != 0 {
		t.Error("empty input should yield zero stats")
	}
}

func TestParseAllTrafficStats_JSON(t *testing.T) {
	input := `{"stat":[{"name":"outbound>>>proxy>>>traffic>>>uplink","value":"500"}]}`
	stats := parseAllTrafficStats(input)
	if stats == nil {
		t.Fatal("should parse JSON format")
	}
	if stats.Uplink != 500 {
		t.Errorf("Uplink = %d, want 500", stats.Uplink)
	}
}

func TestParseAllTrafficStats_Text(t *testing.T) {
	input := `stat: {
  name: "outbound>>>proxy>>>traffic>>>downlink"
  value: 300
}`
	stats := parseAllTrafficStats(input)
	if stats == nil {
		t.Fatal("should parse text format")
	}
	if stats.Downlink != 300 {
		t.Errorf("Downlink = %d, want 300", stats.Downlink)
	}
}

// --- FormatUptime tests ---

func TestFormatUptime_Zero(t *testing.T) {
	if FormatUptime(0) != "-" {
		t.Errorf("FormatUptime(0) = %q, want -", FormatUptime(0))
	}
}

func TestFormatUptime_Minutes(t *testing.T) {
	result := FormatUptime(30 * time.Minute)
	if result != "30m" {
		t.Errorf("FormatUptime(30m) = %q, want 30m", result)
	}
}

func TestFormatUptime_Hours(t *testing.T) {
	result := FormatUptime(2*time.Hour + 15*time.Minute)
	if result != "2h 15m" {
		t.Errorf("FormatUptime(2h15m) = %q, want 2h 15m", result)
	}
}

// --- isPortOpen / checkPortAvailable tests ---

func TestIsPortOpen_Closed_8C(t *testing.T) {
	if isPortOpen("127.0.0.1", 59997) {
		t.Error("port 59997 should not be open")
	}
}

func TestCheckPortAvailable_UsedPort(t *testing.T) {
	// Port 59996 should be available since nothing is running
	err := checkPortAvailable("127.0.0.1", 59996)
	if err != nil {
		t.Logf("Port 59996 not available: %v", err)
	}
}

// --- PID file helpers ---

func TestPIDFileRoundtrip(t *testing.T) {
	tmpDir := t.TempDir()
	setTestHome(t, tmpDir)
	_ = config.EnsureDirs()

	writePIDFile(12345)
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

func TestReadPIDFile_NoFile(t *testing.T) {
	tmpDir := t.TempDir()
	setTestHome(t, tmpDir)
	_ = config.EnsureDirs()

	pid := readPIDFile()
	if pid != 0 {
		t.Errorf("readPIDFile() = %d, want 0 when no file", pid)
	}
}

// --- RotateBackups tests ---

func TestRotateBackups_8C(t *testing.T) {
	tmpDir := t.TempDir()
	setTestHome(t, tmpDir)
	_ = config.EnsureDirs()

	backupDir := config.BackupDir()
	// Create 8 files
	for i := 0; i < 8; i++ {
		f, err := os.Create(filepath.Join(backupDir, fmt.Sprintf("backup-%d.tar.gz", i)))
		if err != nil {
			t.Fatalf("create file: %v", err)
		}
		f.Close()
		time.Sleep(10 * time.Millisecond) // ensure different mod times
	}

	RotateBackups(5)

	entries, _ := os.ReadDir(backupDir)
	if len(entries) != 5 {
		t.Errorf("after RotateBackups(5), count = %d, want 5", len(entries))
	}
}

func TestRotateBackups_NoPrune(t *testing.T) {
	tmpDir := t.TempDir()
	setTestHome(t, tmpDir)
	_ = config.EnsureDirs()

	backupDir := config.BackupDir()
	for i := 0; i < 3; i++ {
		f, _ := os.Create(filepath.Join(backupDir, fmt.Sprintf("backup-%d.tar.gz", i)))
		f.Close()
	}

	RotateBackups(5) // fewer files than limit, should not prune

	entries, _ := os.ReadDir(backupDir)
	if len(entries) != 3 {
		t.Errorf("should not prune when below limit, count = %d", len(entries))
	}
}

func TestRotateBackups_ZeroMax(t *testing.T) {
	tmpDir := t.TempDir()
	setTestHome(t, tmpDir)
	_ = config.EnsureDirs()

	backupDir := config.BackupDir()
	for i := 0; i < 8; i++ {
		f, _ := os.Create(filepath.Join(backupDir, fmt.Sprintf("backup-%d.tar.gz", i)))
		f.Close()
		time.Sleep(10 * time.Millisecond)
	}

	RotateBackups(0) // 0 defaults to 5

	entries, _ := os.ReadDir(backupDir)
	if len(entries) != 5 {
		t.Errorf("RotateBackups(0) should default to 5, count = %d", len(entries))
	}
}

// --- RotateLogs tests ---

func TestRotateLogs_8C(t *testing.T) {
	tmpDir := t.TempDir()
	setTestHome(t, tmpDir)
	_ = config.EnsureDirs()

	settings := config.DefaultSettings()
	settings.Xray.MaxLogSize = 1 // 1 MB

	// Create a small access log (below threshold)
	accessLog := config.AccessLogPath()
	os.WriteFile(accessLog, []byte("small log"), 0644)

	RotateLogs(settings) // should not rotate

	if _, err := os.Stat(accessLog); err != nil {
		t.Error("small log should not be rotated")
	}
}

// --- InvalidateXrayVersionCache ---

func TestInvalidateXrayVersionCache_8C(t *testing.T) {
	// Just exercise the function; verify no panic
	InvalidateXrayVersionCache()
}

// --- GetXrayVersion exercises cache ---

func TestGetXrayVersion_8C(t *testing.T) {
	// Exercise version detection; result depends on environment
	v := GetXrayVersion()
	_ = v
}

// --- XrayProcess.Status ---

func TestXrayProcess_Status_NotRunning_8C(t *testing.T) {
	tmpDir := t.TempDir()
	setTestHome(t, tmpDir)
	_ = config.EnsureDirs()

	xray := NewXrayProcess()
	status := xray.Status()
	if status.Running {
		t.Error("should not be running")
	}
	if status.PID != 0 {
		t.Logf("PID = %d (may detect other xray)", status.PID)
	}
}

// --- XrayStatus fields ---

func TestXrayStatus_Fields(t *testing.T) {
	status := &XrayStatus{
		Running:   true,
		PID:       1234,
		Uptime:    5 * time.Minute,
		SocksOK:   true,
		HTTPOK:    false,
		StartedAt: time.Now().Add(-5 * time.Minute),
	}
	if !status.Running || status.PID != 1234 {
		t.Error("fields mismatch")
	}
}

// --- RunHealthCheck with non-running process ---

func TestRunHealthCheck_NotRunning_8C(t *testing.T) {
	tmpDir := t.TempDir()
	setTestHome(t, tmpDir)
	_ = config.EnsureDirs()

	xray := NewXrayProcess()
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
	settings := config.DefaultSettings()

	report := RunHealthCheck(xray, profile, settings)
	if report == nil {
		t.Fatal("report should not be nil")
	}
	if report.AllPassed {
		t.Error("should not all pass when xray is not running")
	}
	if len(report.Checks) == 0 {
		t.Error("should have at least one check")
	}
}

// --- copyFile tests ---

func TestCopyFile_8C(t *testing.T) {
	tmpDir := t.TempDir()
	src := filepath.Join(tmpDir, "source.txt")
	dst := filepath.Join(tmpDir, "dest.txt")

	os.WriteFile(src, []byte("hello world"), 0644)

	err := copyFile(src, dst)
	if err != nil {
		t.Fatalf("copyFile error: %v", err)
	}

	data, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("read dest error: %v", err)
	}
	if string(data) != "hello world" {
		t.Errorf("dest content = %q, want 'hello world'", data)
	}
}

func TestCopyFile_SourceNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	err := copyFile(filepath.Join(tmpDir, "nonexistent"), filepath.Join(tmpDir, "dest"))
	if err == nil {
		t.Error("should error for nonexistent source")
	}
}

// --- StatsManager file I/O ---

func TestStatsManager_Save_8C(t *testing.T) {
	tmpDir := t.TempDir()
	setTestHome(t, tmpDir)
	_ = config.EnsureDirs()

	sm := &StatsManager{
		history: &StatsHistory{
			Days: []DailyStats{
				{Date: "2026-02-27", Uplink: 100, Downlink: 200},
			},
			TotalUplink:   100,
			TotalDownlink: 200,
		},
	}

	err := sm.Save()
	if err != nil {
		t.Fatalf("Save error: %v", err)
	}

	data, err := os.ReadFile(config.StatsPath())
	if err != nil {
		t.Fatalf("reading stats file: %v", err)
	}
	if !strings.Contains(string(data), "2026-02-27") {
		t.Error("stats file should contain date")
	}
}

func TestStatsManager_Load_8C(t *testing.T) {
	tmpDir := t.TempDir()
	setTestHome(t, tmpDir)
	_ = config.EnsureDirs()

	statsJSON := `{"days":[{"date":"2026-02-27","uplink":100,"downlink":200}],"totalUplink":100,"totalDownlink":200}`
	os.WriteFile(config.StatsPath(), []byte(statsJSON), 0644)

	sm := &StatsManager{}
	sm.load()

	if len(sm.history.Days) != 1 {
		t.Fatalf("Days count = %d, want 1", len(sm.history.Days))
	}
	if sm.history.Days[0].Uplink != 100 {
		t.Errorf("Uplink = %d, want 100", sm.history.Days[0].Uplink)
	}
}

func TestStatsManager_Load_NoFile(t *testing.T) {
	tmpDir := t.TempDir()
	setTestHome(t, tmpDir)
	_ = config.EnsureDirs()

	sm := &StatsManager{}
	sm.load()

	if sm.history == nil {
		t.Fatal("history should not be nil after load")
	}
	if len(sm.history.Days) != 0 {
		t.Error("should have no days with no file")
	}
}

// --- rotateFile tests ---

func TestRotateFile_SmallFile_8C(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")
	os.WriteFile(logPath, []byte("small"), 0644)

	rotateFile(logPath, 1024*1024) // 1 MB threshold

	if _, err := os.Stat(logPath); err != nil {
		t.Error("small file should not be rotated")
	}
}

func TestRotateFile_LargeFile_8C(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")

	data := make([]byte, 2048)
	os.WriteFile(logPath, data, 0644)

	rotateFile(logPath, 1024) // 1 KB threshold

	if _, err := os.Stat(logPath); err == nil {
		t.Error("original file should be renamed after rotation")
	}
	if _, err := os.Stat(logPath + ".1"); err != nil {
		t.Error("rotated file .1 should exist")
	}
}

// --- AssetName extended test ---

func TestAssetName_8C(t *testing.T) {
	name := AssetName()
	if name == "" {
		t.Error("AssetName should not be empty on Linux")
	}
	if !strings.Contains(name, "Xray") {
		t.Errorf("AssetName = %q, should contain Xray", name)
	}
}

// --- parseStatsText with non-proxy entries ---

func TestParseStatsText_NonProxy(t *testing.T) {
	input := `stat: {
  name: "outbound>>>direct>>>traffic>>>uplink"
  value: 999
}`
	stats := parseStatsText(input)
	if stats.Uplink != 0 {
		t.Errorf("non-proxy uplink should be 0, got %d", stats.Uplink)
	}
}

// --- XrayProcess.IsRunning ---

func TestXrayProcess_IsRunning_Fresh(t *testing.T) {
	tmpDir := t.TempDir()
	setTestHome(t, tmpDir)
	_ = config.EnsureDirs()

	xray := NewXrayProcess()
	if xray.IsRunning() {
		t.Error("fresh XrayProcess should not be running")
	}
}

func TestXrayProcess_GetPID_Fresh(t *testing.T) {
	tmpDir := t.TempDir()
	setTestHome(t, tmpDir)
	_ = config.EnsureDirs()

	xray := NewXrayProcess()
	pid := xray.GetPID()
	_ = pid
}
