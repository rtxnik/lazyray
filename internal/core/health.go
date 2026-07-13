package core

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/rtxnik/lazyray/internal/config"
	"github.com/rtxnik/lazyray/internal/platform"
	"github.com/rtxnik/lazyray/internal/procutil"
)

const maxIPCheckBytes = 64 * 1024 // exit-IP responses are tiny; cap to avoid unbounded reads

// readCappedBody reads at most maxIPCheckBytes from rc, guarding against an
// unbounded response from the IP-check endpoint.
func readCappedBody(rc io.ReadCloser) ([]byte, error) {
	return io.ReadAll(io.LimitReader(rc, maxIPCheckBytes))
}

// CheckResult represents a single health check result.
type CheckResult struct {
	Name    string
	OK      bool
	Detail  string
	Latency time.Duration
}

// HealthReport contains all health check results.
type HealthReport struct {
	Checks    []CheckResult
	AllPassed bool
	Timestamp time.Time
}

// RunHealthCheck performs a full health check.
func RunHealthCheck(xrayProc *XrayProcess, profile *config.Profile, settings *config.Settings) *HealthReport {
	report := &HealthReport{
		Timestamp: time.Now(),
		AllPassed: true,
	}

	addCheck := func(c CheckResult) {
		report.Checks = append(report.Checks, c)
		if !c.OK {
			report.AllPassed = false
		}
	}

	// 1. Check process
	addCheck(checkProcess(xrayProc))

	listen := settings.Local.Listen

	timeout := settings.Health.Timeout

	// 2. Check SOCKS5 port
	socksCheck := checkPort("SOCKS5", listen, settings.Local.SocksPort, timeout)
	addCheck(socksCheck)

	// 3. Check HTTP port
	addCheck(checkPort("HTTP", listen, settings.Local.HTTPPort, timeout))

	// 4. Check exit IP (only if SOCKS5 is up)
	if socksCheck.OK {
		expectedIP := ""
		if profile != nil {
			expectedIP = profile.ExpectedExitIP
		}
		addCheck(checkExitIP(listen, settings.Local.SocksPort, expectedIP, settings.Health.Timeout, settings.Health.IPCheckURL))
	} else {
		addCheck(CheckResult{
			Name:   "Exit IP",
			OK:     false,
			Detail: "skipped (SOCKS5 unavailable)",
		})
	}

	// 5. DNS leak check via proxy
	if socksCheck.OK {
		dohConfigured := hasDNSEncryption(settings.Local.DNS)
		addCheck(checkDNSLeak(listen, settings.Local.SocksPort, settings.Health.Timeout, settings.Health.DNSCheckHost, dohConfigured))
	}

	// 6. Latency check
	if socksCheck.OK {
		addCheck(checkLatency(listen, settings.Local.SocksPort, settings.Health.Timeout, settings.Health.LatencyHost))
	}

	// Send system notification on failure
	if !report.AllPassed && settings.Health.AlertOnFailure && settings.Notifications.Enabled {
		var failed []string
		for _, c := range report.Checks {
			if !c.OK {
				failed = append(failed, c.Name)
			}
		}
		msg := fmt.Sprintf("Health check failed: %s", strings.Join(failed, ", "))
		_ = platform.Current().Notify("lazyray", msg)
	}

	return report
}

func checkProcess(xrayProc *XrayProcess) CheckResult {
	if xrayProc.IsRunning() {
		return CheckResult{
			Name:   "Process",
			OK:     true,
			Detail: fmt.Sprintf("xray running (PID %d)", xrayProc.GetPID()),
		}
	}
	return CheckResult{
		Name:   "Process",
		OK:     false,
		Detail: "xray is not running",
	}
}

func checkPort(name, host string, port int, timeout int) CheckResult {
	t := time.Duration(timeout) * time.Second
	if t <= 0 {
		t = 3 * time.Second
	}
	addr := net.JoinHostPort(host, strconv.Itoa(port))
	if err := procutil.Reachable(addr, t); err != nil {
		return CheckResult{
			Name:   name,
			OK:     false,
			Detail: fmt.Sprintf("%s not accepting connections", addr),
		}
	}
	return CheckResult{
		Name:   name,
		OK:     true,
		Detail: fmt.Sprintf("%s accepting", addr),
	}
}

func checkExitIP(listen string, socksPort int, expectedIP string, timeout int, ipCheckURL string) CheckResult {
	if ipCheckURL == "" {
		ipCheckURL = "https://ifconfig.me/ip"
	}
	socksAddr := net.JoinHostPort(listen, strconv.Itoa(socksPort))
	client, err := proxyClient(socksAddr, time.Duration(timeout)*time.Second)
	if err != nil {
		return CheckResult{
			Name:   "Exit IP",
			OK:     false,
			Detail: fmt.Sprintf("SOCKS5 dialer error: %v", err),
		}
	}

	resp, err := client.Get(ipCheckURL)
	if err != nil {
		return CheckResult{
			Name:   "Exit IP",
			OK:     false,
			Detail: fmt.Sprintf("request failed: %v", err),
		}
	}
	defer resp.Body.Close()

	body, err := readCappedBody(resp.Body)
	if err != nil {
		return CheckResult{
			Name:   "Exit IP",
			OK:     false,
			Detail: fmt.Sprintf("read failed: %v", err),
		}
	}

	ip := string(body)
	if expectedIP != "" && ip != expectedIP {
		return CheckResult{
			Name:   "Exit IP",
			OK:     false,
			Detail: fmt.Sprintf("got %s, expected %s", ip, expectedIP),
		}
	}

	detail := ip
	if expectedIP != "" {
		detail = fmt.Sprintf("%s (expected %s)", ip, "OK")
	}
	return CheckResult{
		Name:   "Exit IP",
		OK:     true,
		Detail: detail,
	}
}

func checkDNSLeak(listen string, socksPort int, timeout int, dnsCheckHost string, dohConfigured bool) CheckResult {
	if dnsCheckHost == "" {
		dnsCheckHost = "dns.google:443"
	}
	t := time.Duration(timeout) * time.Second
	if t <= 0 {
		t = 5 * time.Second
	}
	socksAddr := net.JoinHostPort(listen, strconv.Itoa(socksPort))
	dialer, err := proxyDialer(socksAddr, t)
	if err != nil {
		return CheckResult{
			Name:   "DNS Leak",
			OK:     false,
			Detail: fmt.Sprintf("SOCKS5 dialer error: %v", err),
		}
	}

	// Resolve a domain through the proxy by connecting to it
	start := time.Now()
	conn, err := dialer.Dial("tcp", dnsCheckHost)
	elapsed := time.Since(start)

	if err != nil {
		return CheckResult{
			Name:   "DNS Leak",
			OK:     false,
			Detail: fmt.Sprintf("DNS resolution through proxy failed: %v", err),
		}
	}
	conn.Close()

	detail := fmt.Sprintf("resolved via proxy (%dms)", elapsed.Milliseconds())
	if dohConfigured {
		detail += " [encrypted DNS configured]"
	}

	// If resolution succeeded through proxy, DNS is being resolved remotely
	return CheckResult{
		Name:   "DNS Leak",
		OK:     true,
		Detail: detail,
	}
}

// hasDNSEncryption returns true if any DNS server uses DoH or DoT.
func hasDNSEncryption(servers []string) bool {
	for _, s := range servers {
		if strings.HasPrefix(s, "https://") ||
			strings.HasPrefix(s, "https+local://") ||
			strings.HasPrefix(s, "tcp://") {
			return true
		}
	}
	return false
}

func checkLatency(listen string, socksPort int, timeout int, latencyHost string) CheckResult {
	if latencyHost == "" {
		latencyHost = "1.1.1.1:443"
	}
	t := time.Duration(timeout) * time.Second
	if t <= 0 {
		t = 5 * time.Second
	}
	socksAddr := net.JoinHostPort(listen, strconv.Itoa(socksPort))
	dialer, err := proxyDialer(socksAddr, t)
	if err != nil {
		return CheckResult{
			Name:   "Latency",
			OK:     false,
			Detail: fmt.Sprintf("SOCKS5 dialer error: %v", err),
		}
	}

	start := time.Now()
	conn, err := dialer.Dial("tcp", latencyHost)
	latency := time.Since(start)

	if err != nil {
		return CheckResult{
			Name:   "Latency",
			OK:     false,
			Detail: fmt.Sprintf("connection failed: %v", err),
		}
	}
	conn.Close()

	return CheckResult{
		Name:    "Latency",
		OK:      true,
		Detail:  fmt.Sprintf("%dms", latency.Milliseconds()),
		Latency: latency,
	}
}

// GetExitIP returns the current proxy exit IP.
func GetExitIP(settings *config.Settings) (string, error) {
	ipCheckURL := settings.Health.IPCheckURL
	if ipCheckURL == "" {
		ipCheckURL = "https://ifconfig.me/ip"
	}
	socksAddr := net.JoinHostPort(settings.Local.Listen, strconv.Itoa(settings.Local.SocksPort))
	client, err := proxyClient(socksAddr, time.Duration(settings.Health.Timeout)*time.Second)
	if err != nil {
		return "", fmt.Errorf("creating SOCKS5 dialer: %w", err)
	}

	resp, err := client.Get(ipCheckURL)
	if err != nil {
		return "", fmt.Errorf("requesting exit IP: %w", err)
	}
	defer resp.Body.Close()

	body, err := readCappedBody(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading response: %w", err)
	}

	return string(body), nil
}

// GetDirectIP returns the current direct (non-proxy) IP.
func GetDirectIP(settings *config.Settings) (string, error) {
	ipCheckURL := "https://ifconfig.me/ip"
	if settings != nil && settings.Health.IPCheckURL != "" {
		ipCheckURL = settings.Health.IPCheckURL
	}
	client := directClient(5 * time.Second)
	resp, err := safeGet(context.Background(), client, ipCheckURL, 64*1024)
	if err != nil {
		return "", fmt.Errorf("requesting direct IP: %w", err)
	}
	defer resp.Body.Close()

	// Body is already capped by safeGet's io.LimitReader (64*1024 above).
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading response: %w", err)
	}

	return string(body), nil
}

// HealthReportJSON returns the health report as JSON.
func HealthReportJSON(report *HealthReport) (string, error) {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}
