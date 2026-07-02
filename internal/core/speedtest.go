package core

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/rtxnik/lazyray/internal/config"
)

// SpeedTestResult holds the result of a download speed test.
type SpeedTestResult struct {
	Downloaded int64         // Bytes downloaded
	Duration   time.Duration // Time taken
	SpeedMbps  float64       // Megabits per second
	Error      error
}

// RunSpeedTest performs a download speed test through the proxy.
// Downloads data from testURL for the specified duration, measuring throughput.
func RunSpeedTest(settings *config.Settings, testURL string, duration time.Duration) *SpeedTestResult {
	if testURL == "" {
		testURL = "http://speed.cloudflare.com/__down?bytes=104857600"
	}
	if duration == 0 {
		duration = 10 * time.Second
	}

	socksAddr := net.JoinHostPort(settings.Local.Listen, strconv.Itoa(settings.Local.SocksPort))
	client, err := proxyClient(socksAddr, duration+5*time.Second)
	if err != nil {
		return &SpeedTestResult{Error: fmt.Errorf("SOCKS5 dialer: %w", err)}
	}

	ctx, cancel := context.WithTimeout(context.Background(), duration)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", testURL, nil)
	if err != nil {
		return &SpeedTestResult{Error: fmt.Errorf("creating request: %w", err)}
	}

	start := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		elapsed := time.Since(start)
		// If context was cancelled (test duration elapsed), that's expected
		if ctx.Err() != nil && elapsed >= duration-time.Second {
			return &SpeedTestResult{Error: fmt.Errorf("connection failed before test could start: %w", err)}
		}
		return &SpeedTestResult{Error: fmt.Errorf("download request: %w", err)}
	}
	defer resp.Body.Close()

	// Read data until context expires or download completes
	buf := make([]byte, 32*1024) // 32KB buffer
	var totalBytes int64

	for {
		n, err := resp.Body.Read(buf)
		totalBytes += int64(n)
		if err != nil {
			break
		}
	}

	elapsed := time.Since(start)
	if elapsed < time.Millisecond {
		elapsed = time.Millisecond
	}

	speedMbps := float64(totalBytes) * 8 / elapsed.Seconds() / 1_000_000

	return &SpeedTestResult{
		Downloaded: totalBytes,
		Duration:   elapsed,
		SpeedMbps:  speedMbps,
	}
}

// RunSpeedTestWithCallback performs a speed test and reports progress via callback.
func RunSpeedTestWithCallback(settings *config.Settings, testURL string, duration time.Duration, callback func(downloaded int64, elapsed time.Duration)) *SpeedTestResult {
	if testURL == "" {
		testURL = "http://speed.cloudflare.com/__down?bytes=104857600"
	}
	if duration == 0 {
		duration = 10 * time.Second
	}

	socksAddr := net.JoinHostPort(settings.Local.Listen, strconv.Itoa(settings.Local.SocksPort))
	client, err := proxyClient(socksAddr, duration+5*time.Second)
	if err != nil {
		return &SpeedTestResult{Error: fmt.Errorf("SOCKS5 dialer: %w", err)}
	}

	ctx, cancel := context.WithTimeout(context.Background(), duration)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", testURL, nil)
	if err != nil {
		return &SpeedTestResult{Error: fmt.Errorf("creating request: %w", err)}
	}

	start := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		return &SpeedTestResult{Error: fmt.Errorf("download request: %w", err)}
	}
	defer resp.Body.Close()

	buf := make([]byte, 32*1024)
	var totalBytes int64
	lastReport := start

	for {
		n, err := resp.Body.Read(buf)
		totalBytes += int64(n)

		now := time.Now()
		if callback != nil && now.Sub(lastReport) >= 500*time.Millisecond {
			callback(totalBytes, now.Sub(start))
			lastReport = now
		}

		if err != nil {
			break
		}
	}

	elapsed := time.Since(start)
	if elapsed < time.Millisecond {
		elapsed = time.Millisecond
	}

	speedMbps := float64(totalBytes) * 8 / elapsed.Seconds() / 1_000_000

	return &SpeedTestResult{
		Downloaded: totalBytes,
		Duration:   elapsed,
		SpeedMbps:  speedMbps,
	}
}

// FormatSpeedTestResult returns a human-readable report of the speed test.
func FormatSpeedTestResult(result *SpeedTestResult) string {
	if result.Error != nil {
		return fmt.Sprintf("Speed test failed: %v", result.Error)
	}

	return fmt.Sprintf("Downloaded: %s in %.1fs\nSpeed: %.2f Mbps\n",
		FormatBytes(result.Downloaded),
		result.Duration.Seconds(),
		result.SpeedMbps)
}
