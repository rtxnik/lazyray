package core

import (
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/rtxnik/lazyray/internal/config"
)

// LivenessStatus is the tri-state outcome of a profile liveness probe.
type LivenessStatus int

const (
	LivenessOK      LivenessStatus = iota // reachable / measured
	LivenessFail                          // a real failure (down endpoint, broken proxy)
	LivenessSkipped                       // cannot be probed cheaply (datagram, not connected)
)

// LivenessResult is the outcome of probing one profile.
type LivenessResult struct {
	Status  LivenessStatus
	Latency time.Duration // valid when Status == LivenessOK
	Method  string        // "functional" | "tcp" | "" (skipped)
	Reason  string        // human hint, e.g. "UDP/QUIC — connect to verify"
	Err     error         // set when Status == LivenessFail
}

// ProbeContext carries connected-state the caller derives from lifecycle.State
// (which core cannot import). Active means "this profile is the one xray is
// currently routing"; SocksAddr/LatencyHost drive the functional probe.
type ProbeContext struct {
	Active      bool
	SocksAddr   string
	LatencyHost string
	Timeout     time.Duration
}

// ProbeProfile selects and runs the liveness strategy for a profile. It is pure:
// it never reads runtime state — the caller supplies connected-state via pc.
func ProbeProfile(profile config.Profile, pc ProbeContext) LivenessResult {
	if pc.Active {
		return functionalProbe(pc)
	}
	if isDatagramTransport(profile.Server.GetProtocol()) {
		return LivenessResult{Status: LivenessSkipped, Reason: "UDP/QUIC — connect to verify"}
	}
	return tcpProbe(profile.Server, pc.Timeout)
}

// functionalProbe measures end-to-end latency through the live local SOCKS proxy.
// A dial failure here means the proxy is up-but-not-working → a true Fail.
func functionalProbe(pc ProbeContext) LivenessResult {
	host := pc.LatencyHost
	if host == "" {
		host = "1.1.1.1:443"
	}
	t := pc.Timeout
	if t <= 0 {
		t = 3 * time.Second
	}
	dialer, err := proxyDialer(pc.SocksAddr, t)
	if err != nil {
		return LivenessResult{Status: LivenessFail, Method: "functional", Err: fmt.Errorf("socks dialer: %w", err)}
	}
	start := time.Now()
	conn, err := dialer.Dial("tcp", host)
	if err != nil {
		return LivenessResult{Status: LivenessFail, Method: "functional", Err: fmt.Errorf("functional probe via proxy failed: %w", err)}
	}
	conn.Close()
	return LivenessResult{Status: LivenessOK, Method: "functional", Latency: time.Since(start)}
}

// tcpProbe is the reachability probe for stream transports: a plain TCP connect.
func tcpProbe(server config.ServerConfig, timeout time.Duration) LivenessResult {
	if timeout <= 0 {
		timeout = 3 * time.Second
	}
	addr := net.JoinHostPort(server.Address, strconv.Itoa(server.Port))
	start := time.Now()
	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return LivenessResult{Status: LivenessFail, Method: "tcp", Err: fmt.Errorf("connection to %s failed: %w", addr, err)}
	}
	conn.Close()
	return LivenessResult{Status: LivenessOK, Method: "tcp", Latency: time.Since(start)}
}
