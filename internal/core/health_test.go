package core

import (
	"net"
	"strconv"
	"strings"
	"testing"

	"github.com/rtxnik/lazyray/internal/config"
)

func TestCheckPort_Open(t *testing.T) {
	// Start a TCP listener on a random port
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to start listener: %v", err)
	}
	defer ln.Close()

	addr := ln.Addr().(*net.TCPAddr)
	result := checkPort("test", "127.0.0.1", addr.Port, 3)

	if !result.OK {
		t.Errorf("checkPort() returned OK=false for open port %d: %s", addr.Port, result.Detail)
	}
	if result.Name != "test" {
		t.Errorf("checkPort() Name = %q, want %q", result.Name, "test")
	}
}

func TestCheckPort_Closed(t *testing.T) {
	// Find a port that is not listening
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to get free port: %v", err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	ln.Close() // Close immediately so port is freed

	result := checkPort("test", "127.0.0.1", port, 1)

	if result.OK {
		t.Errorf("checkPort() returned OK=true for closed port %d", port)
	}
}

func TestCheckPort_Detail(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to start listener: %v", err)
	}
	defer ln.Close()

	addr := ln.Addr().(*net.TCPAddr)
	result := checkPort("SOCKS5", "127.0.0.1", addr.Port, 3)

	expected := "127.0.0.1:" + strconv.Itoa(addr.Port) + " accepting"
	if result.Detail != expected {
		t.Errorf("checkPort() Detail = %q, want %q", result.Detail, expected)
	}
}

func TestCheckPort_ZeroTimeout(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to start listener: %v", err)
	}
	defer ln.Close()

	addr := ln.Addr().(*net.TCPAddr)
	// timeout=0 should fallback to 3 seconds
	result := checkPort("test", "127.0.0.1", addr.Port, 0)

	if !result.OK {
		t.Errorf("checkPort() with timeout=0 should still work for open port: %s", result.Detail)
	}
}

func TestGetDirectIP_RejectsNonHTTPS(t *testing.T) {
	// Routing GetDirectIP through safeGet makes it https-only: a non-https
	// IPCheckURL (e.g. a plaintext metadata URL) must be refused before any dial.
	s := &config.Settings{}
	s.Health.IPCheckURL = "http://169.254.169.254/latest/meta-data/"
	_, err := GetDirectIP(s)
	if err == nil {
		t.Fatal("GetDirectIP accepted a non-https IPCheckURL; want safeGet https-only rejection")
	}
	if !strings.Contains(err.Error(), "https") {
		t.Errorf("error %q should mention the https-only requirement", err.Error())
	}
}
