package core

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/rtxnik/lazyray/internal/config"
	"github.com/rtxnik/lazyray/internal/platform"
)

// ansiRegex strips ANSI escape sequences from command output.
var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

// XrayStatus represents the current state of the xray process.
type XrayStatus struct {
	Running   bool
	PID       int
	Uptime    time.Duration
	SocksOK   bool
	HTTPOK    bool
	StartedAt time.Time
}

// TrafficStats holds upload and download byte counts.
type TrafficStats struct {
	Uplink   int64
	Downlink int64
}

// XrayProcess manages the xray process lifecycle.
type XrayProcess struct {
	mu        sync.RWMutex // protects cmd and startedAt
	cmd       *exec.Cmd
	startedAt time.Time

	// Watchdog state
	watchdogMu   sync.Mutex
	watchdogStop chan struct{}
	watchdogOn   bool
}

// NewXrayProcess creates a new xray process manager.
func NewXrayProcess() *XrayProcess {
	return &XrayProcess{}
}

// Start launches the xray process as a child (for TUI use).
func (x *XrayProcess) Start() error {
	x.mu.Lock()
	defer x.mu.Unlock()
	return x.startLocked()
}

func (x *XrayProcess) startLocked() error {
	if x.isRunningLocked() {
		return fmt.Errorf("xray is already running (PID %d)", x.getPIDLocked())
	}

	xrayBin, xrayConfig, settings, err := prepareStart()
	if err != nil {
		return err
	}

	RotateLogs(settings)

	if err := checkPortAvailable(settings.Local.Listen, settings.Local.SocksPort); err != nil {
		return fmt.Errorf("SOCKS5 port %d: %w", settings.Local.SocksPort, err)
	}
	if err := checkPortAvailable(settings.Local.Listen, settings.Local.HTTPPort); err != nil {
		return fmt.Errorf("HTTP port %d: %w", settings.Local.HTTPPort, err)
	}

	x.cmd = exec.Command(xrayBin, "run", "-c", xrayConfig)
	var stderrBuf bytes.Buffer
	x.cmd.Stdout = nil
	x.cmd.Stderr = &stderrBuf

	if err := x.cmd.Start(); err != nil {
		return fmt.Errorf("starting xray: %w", err)
	}

	x.startedAt = time.Now()
	writePIDFile(x.cmd.Process.Pid)

	// Wait briefly and check if process is still alive
	time.Sleep(500 * time.Millisecond)
	if !x.isRunningLocked() {
		_ = x.cmd.Wait()
		removePIDFile()
		errOutput := strings.TrimSpace(stderrBuf.String())
		if errOutput != "" {
			return fmt.Errorf("xray process exited immediately:\n%s", errOutput)
		}
		return startFailedError(xrayBin, xrayConfig)
	}

	return nil
}

// StartDetached launches xray as a background process independent of the caller.
// The process survives after the parent (CLI) exits.
func (x *XrayProcess) StartDetached() (int, error) {
	if x.IsRunning() {
		return 0, fmt.Errorf("xray is already running (PID %d)", x.GetPID())
	}

	xrayBin, xrayConfig, settings, err := prepareStart()
	if err != nil {
		return 0, err
	}

	RotateLogs(settings)

	if err := checkPortAvailable(settings.Local.Listen, settings.Local.SocksPort); err != nil {
		return 0, fmt.Errorf("SOCKS5 port %d: %w", settings.Local.SocksPort, err)
	}
	if err := checkPortAvailable(settings.Local.Listen, settings.Local.HTTPPort); err != nil {
		return 0, fmt.Errorf("HTTP port %d: %w", settings.Local.HTTPPort, err)
	}

	cmd := exec.Command(xrayBin, "run", "-c", xrayConfig)
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.Stdin = nil
	cmd.SysProcAttr = detachedProcAttr()

	if err := cmd.Start(); err != nil {
		return 0, fmt.Errorf("starting xray: %w", err)
	}

	pid := cmd.Process.Pid
	writePIDFile(pid)

	// Release so the child is not reaped when we exit
	_ = cmd.Process.Release()

	// Wait briefly and verify the process is alive
	time.Sleep(500 * time.Millisecond)
	if !isProcessAlive(pid) {
		removePIDFile()
		return 0, startFailedError(xrayBin, xrayConfig)
	}

	return pid, nil
}

// Stop terminates the xray process and stops the watchdog.
func (x *XrayProcess) Stop() error {
	x.StopWatchdog()
	x.mu.Lock()
	defer x.mu.Unlock()
	return x.stopLocked()
}

func (x *XrayProcess) stopLocked() error {
	if x.cmd != nil && x.cmd.Process != nil {
		if err := gracefulKill(x.cmd.Process); err != nil {
			return fmt.Errorf("stopping xray: %w", err)
		}
		x.cmd = nil
		removePIDFile()
		return nil
	}

	// Try PID file first, then fallback to process search
	pid := readPIDFile()
	if pid == 0 {
		pid = findXrayPID()
	}
	if pid == 0 {
		return fmt.Errorf("xray is not running")
	}

	if !isOurXray(pid) {
		// Stale or reused PID: the recorded xray is gone. Drop the pidfile and
		// report not-running rather than signalling an unrelated process.
		removePIDFile()
		return fmt.Errorf("xray is not running")
	}

	proc, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("finding process %d: %w", pid, err)
	}

	if err := gracefulKill(proc); err != nil {
		return fmt.Errorf("killing process %d: %w", pid, err)
	}

	removePIDFile()
	return nil
}

// Restart stops and starts the xray process.
func (x *XrayProcess) Restart() error {
	x.StopWatchdog()
	x.mu.Lock()
	defer x.mu.Unlock()
	if x.isRunningLocked() {
		if err := x.stopLocked(); err != nil {
			return fmt.Errorf("stopping for restart: %w", err)
		}
		time.Sleep(500 * time.Millisecond)
	}
	return x.startLocked()
}

// IsRunning checks if the xray process is alive.
func (x *XrayProcess) IsRunning() bool {
	x.mu.RLock()
	defer x.mu.RUnlock()
	return x.isRunningLocked()
}

func (x *XrayProcess) isRunningLocked() bool {
	if x.cmd != nil && x.cmd.Process != nil {
		if x.cmd.ProcessState != nil && x.cmd.ProcessState.Exited() {
			return false
		}
		return isProcessAlive(x.cmd.Process.Pid)
	}
	// Check PID file first
	if pid := readPIDFile(); pid > 0 {
		if isProcessAlive(pid) {
			return true
		}
		// Stale PID file
		removePIDFile()
	}
	return findXrayPID() > 0
}

// GetPID returns the PID of the running xray process.
func (x *XrayProcess) GetPID() int {
	x.mu.RLock()
	defer x.mu.RUnlock()
	return x.getPIDLocked()
}

func (x *XrayProcess) getPIDLocked() int {
	if x.cmd != nil && x.cmd.Process != nil {
		return x.cmd.Process.Pid
	}
	if pid := readPIDFile(); pid > 0 && isProcessAlive(pid) {
		return pid
	}
	return findXrayPID()
}

// Status returns the current xray status.
func (x *XrayProcess) Status() *XrayStatus {
	// Read protected fields under lock, release before port checks
	x.mu.RLock()
	running := x.isRunningLocked()
	pid := x.getPIDLocked()
	startedAt := x.startedAt
	x.mu.RUnlock()

	settings, _ := config.LoadSettings()
	if settings == nil {
		settings = config.DefaultSettings()
	}

	status := &XrayStatus{
		Running: running,
		PID:     pid,
	}

	if running && !startedAt.IsZero() {
		status.StartedAt = startedAt
		status.Uptime = time.Since(startedAt)
	}

	listen := settings.Local.Listen
	status.SocksOK = isPortOpen(listen, settings.Local.SocksPort)
	status.HTTPOK = isPortOpen(listen, settings.Local.HTTPPort)

	return status
}

// FormatUptime returns a human-readable uptime string.
func FormatUptime(d time.Duration) string {
	if d == 0 {
		return "-"
	}
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
	return fmt.Sprintf("%dm", minutes)
}

// GetTrafficStats queries the xray stats API for traffic counters.
// Returns cumulative upload/download bytes for the proxy outbound.
func GetTrafficStats() *TrafficStats {
	xrayBin := config.XrayBinaryPath()
	if _, err := os.Stat(xrayBin); err != nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	serverAddr := fmt.Sprintf("127.0.0.1:%d", StatsAPIPort)

	// Query all stats at once (no filter flags — avoids compatibility issues
	// across xray-core versions). CombinedOutput captures both stdout and stderr.
	out, err := exec.CommandContext(ctx, xrayBin, "api", "statsquery",
		"--server="+serverAddr).CombinedOutput()
	if err != nil {
		return nil
	}

	return parseAllTrafficStats(string(out))
}

// FormatBytes formats byte count to a human-readable string.
func FormatBytes(b int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)
	switch {
	case b >= GB:
		return fmt.Sprintf("%.1f GB", float64(b)/float64(GB))
	case b >= MB:
		return fmt.Sprintf("%.0f MB", float64(b)/float64(MB))
	case b >= KB:
		return fmt.Sprintf("%.0f KB", float64(b)/float64(KB))
	default:
		return fmt.Sprintf("%d B", b)
	}
}

// maxLogArchives is the number of rotated log files to keep (.1, .2, .3).
const maxLogArchives = 3

// RotateLogs checks log file sizes and rotates if they exceed MaxLogSize.
// Keeps up to 3 archived copies (.1, .2, .3); oldest is deleted.
func RotateLogs(settings *config.Settings) {
	maxSize := settings.Xray.MaxLogSize
	if maxSize <= 0 {
		maxSize = 10
	}
	maxBytes := int64(maxSize) * 1024 * 1024

	for _, logPath := range []string{config.AccessLogPath(), config.ErrorLogPath()} {
		rotateFile(logPath, maxBytes)
	}
}

func rotateFile(path string, maxBytes int64) {
	info, err := os.Stat(path)
	if err != nil || info.Size() < maxBytes {
		return
	}

	// Delete oldest archive and shift others down
	for i := maxLogArchives; i >= 1; i-- {
		src := fmt.Sprintf("%s.%d", path, i)
		if i == maxLogArchives {
			_ = os.Remove(src)
		} else {
			dst := fmt.Sprintf("%s.%d", path, i+1)
			_ = os.Rename(src, dst)
		}
	}
	_ = os.Rename(path, path+".1")
}

// RotateBackups removes the oldest backup files when the count exceeds maxFiles.
// Files in BackupDir are sorted by modification time; oldest are deleted first.
func RotateBackups(maxFiles int) {
	if maxFiles <= 0 {
		maxFiles = 5
	}

	dir := config.BackupDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}

	// Collect only regular files
	type fileInfo struct {
		name    string
		modTime time.Time
	}
	var files []fileInfo
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		files = append(files, fileInfo{name: e.Name(), modTime: info.ModTime()})
	}

	if len(files) <= maxFiles {
		return
	}

	// Sort by modification time ascending (oldest first)
	sort.Slice(files, func(i, j int) bool {
		return files[i].modTime.Before(files[j].modTime)
	})

	// Remove oldest files
	toRemove := len(files) - maxFiles
	for i := 0; i < toRemove; i++ {
		_ = os.Remove(filepath.Join(dir, files[i].name))
	}
}

// parseAllTrafficStats parses the full output of `xray api statsquery` and
// extracts uplink/downlink traffic for the proxy outbound.
// Handles both JSON format (modern xray-core) and protobuf text format (older versions).
func parseAllTrafficStats(output string) *TrafficStats {
	// Try JSON format first (xray-core 1.8+)
	if stats := parseStatsJSON(output); stats != nil {
		return stats
	}
	// Fall back to protobuf text format
	return parseStatsText(output)
}

// parseStatsJSON parses JSON output from `xray api statsquery`.
// Format: {"stat":[{"name":"outbound>>>proxy>>>traffic>>>uplink","value":"12345"},...]}
func parseStatsJSON(output string) *TrafficStats {
	var resp struct {
		Stat []struct {
			Name  string      `json:"name"`
			Value interface{} `json:"value"`
		} `json:"stat"`
	}
	if err := json.Unmarshal([]byte(output), &resp); err != nil {
		return nil
	}
	stats := &TrafficStats{}
	for _, s := range resp.Stat {
		if !strings.Contains(s.Name, "proxy") {
			continue
		}
		val := toInt64(s.Value)
		if strings.HasSuffix(s.Name, "uplink") {
			stats.Uplink = val
		}
		if strings.HasSuffix(s.Name, "downlink") {
			stats.Downlink = val
		}
	}
	return stats
}

// toInt64 converts a JSON value (float64, string, or json.Number) to int64.
func toInt64(v interface{}) int64 {
	switch val := v.(type) {
	case float64:
		return int64(val)
	case string:
		n, _ := strconv.ParseInt(val, 10, 64)
		return n
	default:
		return 0
	}
}

// parseStatsText parses protobuf text format output from `xray api statsquery`.
// Format:
//
//	stat: {
//	  name: "outbound>>>proxy>>>traffic>>>uplink"
//	  value: 12345
//	}
func parseStatsText(output string) *TrafficStats {
	stats := &TrafficStats{}
	var currentName string

	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, "name:") {
			currentName = strings.TrimSpace(strings.TrimPrefix(line, "name:"))
			currentName = strings.Trim(currentName, "\"")
			continue
		}

		if strings.HasPrefix(line, "value:") && currentName != "" {
			valStr := strings.TrimSpace(strings.TrimPrefix(line, "value:"))
			valStr = strings.Trim(valStr, "\"")
			val, err := strconv.ParseInt(valStr, 10, 64)
			if err == nil {
				if strings.Contains(currentName, "proxy") && strings.HasSuffix(currentName, "uplink") {
					stats.Uplink = val
				}
				if strings.Contains(currentName, "proxy") && strings.HasSuffix(currentName, "downlink") {
					stats.Downlink = val
				}
			}
			currentName = ""
		}
	}

	return stats
}

// StartWatchdog launches a background goroutine that monitors the xray process.
// If xray crashes and autoRestart is enabled in settings, it attempts to restart
// up to maxRetries times. After exhausting retries, a system notification is sent.
func (x *XrayProcess) StartWatchdog(maxRetries int) {
	x.watchdogMu.Lock()
	defer x.watchdogMu.Unlock()

	if x.watchdogOn {
		return
	}

	settings, _ := config.LoadSettings()
	if settings == nil {
		settings = config.DefaultSettings()
	}
	if !settings.Xray.AutoRestart {
		return
	}

	x.watchdogStop = make(chan struct{})
	x.watchdogOn = true

	go func() {
		retries := 0
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-x.watchdogStop:
				return
			case <-ticker.C:
				if x.IsRunning() {
					retries = 0
					continue
				}

				// Process is not running — attempt restart
				if retries >= maxRetries {
					msg := fmt.Sprintf("xray crashed and failed to restart after %d attempts", maxRetries)
					if s, err := config.LoadSettings(); err == nil && s.Notifications.Enabled {
						_ = platform.Current().Notify("lazyray", msg)
					}
					x.watchdogMu.Lock()
					x.watchdogOn = false
					x.watchdogMu.Unlock()
					return
				}

				retries++
				if err := x.Start(); err != nil {
					continue
				}
				// Restart succeeded, reset counter
				retries = 0
			}
		}
	}()
}

// StopWatchdog stops the background watchdog goroutine.
func (x *XrayProcess) StopWatchdog() {
	x.watchdogMu.Lock()
	defer x.watchdogMu.Unlock()

	if x.watchdogOn && x.watchdogStop != nil {
		close(x.watchdogStop)
		x.watchdogOn = false
	}
}

// WatchdogRunning returns whether the watchdog is currently active.
func (x *XrayProcess) WatchdogRunning() bool {
	x.watchdogMu.Lock()
	defer x.watchdogMu.Unlock()
	return x.watchdogOn
}

// prepareStart validates prerequisites for starting xray.
func prepareStart() (xrayBin, xrayConfig string, settings *config.Settings, err error) {
	xrayBin = config.XrayBinaryPath()
	if _, e := os.Stat(xrayBin); os.IsNotExist(e) {
		return "", "", nil, fmt.Errorf("xray binary not found at %s (use 'lzr update apply' to download)", xrayBin)
	}

	xrayConfig = config.XrayConfigPath()
	if _, e := os.Stat(xrayConfig); os.IsNotExist(e) {
		return "", "", nil, fmt.Errorf("xray config not found at %s (use 'lzr import' to create one)", xrayConfig)
	}

	if e := config.EnsureDirs(); e != nil {
		return "", "", nil, fmt.Errorf("creating directories: %w", e)
	}

	settings, e := config.LoadSettings()
	if e != nil {
		return "", "", nil, fmt.Errorf("loading settings: %w", e)
	}

	if configUsesHysteria(xrayConfig) {
		if e := CheckProtocolXraySupport("hysteria2"); e != nil {
			return "", "", nil, e
		}
	}

	return xrayBin, xrayConfig, settings, nil
}

// insecureRemovedError returns a clear remediation error if the xray startup log
// shows the removed-allowInsecure failure (xray-core >= v26 removed the option),
// or nil otherwise. It translates xray's cryptic message into actionable guidance.
func insecureRemovedError(logContent string) error {
	if strings.Contains(logContent, "allowInsecure") && strings.Contains(logContent, "removed") {
		return fmt.Errorf("xray rejected insecure TLS: this xray version removed the \"allowInsecure\" option; " +
			"re-import the server with a pinSHA256 certificate pin, or pin the server certificate")
	}
	return nil
}

func startFailedError(xrayBin, xrayConfig string) error {
	if logData, err := os.ReadFile(config.ErrorLogPath()); err == nil && len(logData) > 0 {
		if e := insecureRemovedError(string(logData)); e != nil {
			return e
		}
		lines := strings.Split(strings.TrimSpace(string(logData)), "\n")
		start := 0
		if len(lines) > 10 {
			start = len(lines) - 10
		}
		return fmt.Errorf("xray process exited immediately — from %s:\n%s",
			config.ErrorLogPath(), strings.Join(lines[start:], "\n"))
	}
	return fmt.Errorf("xray process exited immediately (no error output captured)\n"+
		"  binary: %s\n  config: %s\n"+
		"  try running manually: %s run -c %s",
		xrayBin, xrayConfig, xrayBin, xrayConfig)
}

// PID file helpers

func writePIDFile(pid int) {
	_ = os.WriteFile(config.PIDFilePath(), []byte(strconv.Itoa(pid)), 0644)
}

func readPIDFile() int {
	data, err := os.ReadFile(config.PIDFilePath())
	if err != nil {
		return 0
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0
	}
	return pid
}

func removePIDFile() {
	_ = os.Remove(config.PIDFilePath())
}

func checkPortAvailable(host string, port int) error {
	addr := net.JoinHostPort(host, strconv.Itoa(port))
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("port already in use or unavailable: %w", err)
	}
	ln.Close()
	return nil
}

func isPortOpen(host string, port int) bool {
	addr := net.JoinHostPort(host, strconv.Itoa(port))
	conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// findXrayPID searches for a running xray process and returns its PID.
// Package-level var to allow mocking in tests.
var findXrayPID = findXrayPIDImpl

// findXrayPIDImpl matches ANY xray process by name. It is diagnostics-only
// (e.g. detecting a foreign xray); it MUST NOT be used as a termination target.
// The authoritative lifecycle source of truth is internal/lifecycle (state.json
// + supervisor.lock).
func findXrayPIDImpl() int {
	if runtime.GOOS == "windows" {
		out, err := exec.Command("tasklist", "/FI", "IMAGENAME eq xray.exe", "/FO", "CSV", "/NH").Output()
		if err != nil {
			return 0
		}
		lines := strings.Split(strings.TrimSpace(string(out)), "\n")
		for _, line := range lines {
			fields := strings.Split(line, ",")
			if len(fields) >= 2 {
				pidStr := strings.Trim(fields[1], "\" ")
				pid, err := strconv.Atoi(pidStr)
				if err == nil {
					return pid
				}
			}
		}
		return 0
	}

	out, err := exec.Command("pgrep", "-x", "xray").Output()
	if err != nil {
		return 0
	}

	pidStr := strings.TrimSpace(ansiRegex.ReplaceAllString(strings.Split(string(out), "\n")[0], ""))
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		return 0
	}
	return pid
}

var (
	cachedXrayVersion   string
	cachedXrayVersionAt time.Time
	xrayVersionMu       sync.Mutex
)

// GetXrayVersion returns the installed xray version (cached for 60s).
func GetXrayVersion() string {
	xrayVersionMu.Lock()
	defer xrayVersionMu.Unlock()

	if cachedXrayVersion != "" && time.Since(cachedXrayVersionAt) < 60*time.Second {
		return cachedXrayVersion
	}

	cachedXrayVersion = getXrayVersionUncached()
	cachedXrayVersionAt = time.Now()
	return cachedXrayVersion
}

// InvalidateXrayVersionCache forces the next GetXrayVersion call to re-check.
func InvalidateXrayVersionCache() {
	xrayVersionMu.Lock()
	cachedXrayVersion = ""
	xrayVersionMu.Unlock()
}

// MinXrayVersion is the minimum supported xray-core version.
const MinXrayVersion = "1.8.0"

// MinXrayVersionHysteria2 is the minimum xray-core version that correctly runs
// the hysteria2 outbound schema (hysteriaSettings + finalmask). Older builds
// silently ignore obfs/pinning, so hysteria2 is hard-gated. See the Hysteria2
// section of TROUBLESHOOTING.md.
const MinXrayVersionHysteria2 = "26.2.6"

// CheckProtocolXraySupport returns an error if the installed xray is too old to
// correctly run the given lazyray protocol. Unknown/unprobeable versions pass.
func CheckProtocolXraySupport(protocol string) error {
	spec, ok := protocolFor(protocol)
	if !ok || spec.MinXrayVersion == "" {
		return nil
	}
	ver := GetXrayVersion()
	if ver == "not installed" || ver == "unknown" || ver == "" {
		return nil
	}
	if compareVersions(ver, spec.MinXrayVersion) < 0 {
		return fmt.Errorf("xray %s is too old for %s (need >= %s); run 'lzr update apply' to upgrade",
			ver, protocol, spec.MinXrayVersion)
	}
	return nil
}

// configUsesHysteria reports whether the written xray config has a hysteria outbound.
func configUsesHysteria(configPath string) bool {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return false
	}
	var cfg struct {
		Outbounds []struct {
			Protocol string `json:"protocol"`
		} `json:"outbounds"`
	}
	if json.Unmarshal(data, &cfg) != nil {
		return false
	}
	for _, o := range cfg.Outbounds {
		if o.Protocol == "hysteria" { // xray wire name for hysteria2
			return true
		}
	}
	return false
}

// CheckXrayVersionCompat checks if the installed xray version meets the minimum.
// Returns a warning message if outdated, or empty string if OK.
func CheckXrayVersionCompat() string {
	ver := GetXrayVersion()
	if ver == "not installed" || ver == "unknown" || ver == "" {
		return ""
	}
	if compareVersions(ver, MinXrayVersion) < 0 {
		return fmt.Sprintf("Xray %s is outdated (min %s), press u to update", ver, MinXrayVersion)
	}
	return ""
}

// compareVersions compares two semver-like version strings.
// Returns -1 if a < b, 0 if equal, 1 if a > b.
func compareVersions(a, b string) int {
	a = strings.TrimPrefix(a, "v")
	b = strings.TrimPrefix(b, "v")
	aParts := strings.SplitN(a, ".", 3)
	bParts := strings.SplitN(b, ".", 3)
	for i := 0; i < 3; i++ {
		var av, bv int
		if i < len(aParts) {
			av, _ = strconv.Atoi(strings.Split(aParts[i], "-")[0])
		}
		if i < len(bParts) {
			bv, _ = strconv.Atoi(strings.Split(bParts[i], "-")[0])
		}
		if av < bv {
			return -1
		}
		if av > bv {
			return 1
		}
	}
	return 0
}

func getXrayVersionUncached() string {
	xrayBin := config.XrayBinaryPath()
	out, err := exec.Command(xrayBin, "version").Output()
	if err != nil {
		return "not installed"
	}

	cleaned := ansiRegex.ReplaceAllString(string(out), "")
	lines := strings.Split(cleaned, "\n")
	if len(lines) > 0 {
		parts := strings.Fields(lines[0])
		for i, p := range parts {
			if strings.EqualFold(p, "Xray") && i+1 < len(parts) {
				return parts[i+1]
			}
		}
	}
	return "unknown"
}
