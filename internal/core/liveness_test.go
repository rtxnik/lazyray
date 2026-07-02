package core

import (
	"net"
	"strconv"
	"testing"
	"time"

	"github.com/rtxnik/lazyray/internal/config"
)

func TestIsDatagramTransport(t *testing.T) {
	cases := map[string]bool{
		"hysteria2": true, "vless": false, "vmess": false,
		"trojan": false, "shadowsocks": false, "": false,
	}
	for proto, want := range cases {
		if got := isDatagramTransport(proto); got != want {
			t.Errorf("isDatagramTransport(%q) = %v, want %v", proto, got, want)
		}
	}
}

func TestProbeProfile_DatagramNotConnected_Skipped(t *testing.T) {
	p := config.Profile{Name: "hy2", Server: config.ServerConfig{Address: "1.2.3.4", Port: 443, Protocol: "hysteria2"}}
	r := ProbeProfile(p, ProbeContext{Active: false, Timeout: time.Second})
	if r.Status != LivenessSkipped {
		t.Fatalf("Status = %v, want Skipped (%+v)", r.Status, r)
	}
	if r.Reason == "" {
		t.Error("Skipped result must carry a Reason")
	}
}

func TestProbeProfile_StreamNotConnected_TCPReachable(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	_, portStr, _ := net.SplitHostPort(ln.Addr().String())
	port, _ := strconv.Atoi(portStr)

	p := config.Profile{Name: "v", Server: config.ServerConfig{Address: "127.0.0.1", Port: port, Protocol: "vless"}}
	r := ProbeProfile(p, ProbeContext{Active: false, Timeout: time.Second})
	if r.Status != LivenessOK || r.Method != "tcp" {
		t.Fatalf("got %+v, want OK/tcp", r)
	}
}

func TestProbeProfile_StreamNotConnected_TCPDown_Fail(t *testing.T) {
	p := config.Profile{Name: "v", Server: config.ServerConfig{Address: "127.0.0.1", Port: 1, Protocol: "vless"}}
	r := ProbeProfile(p, ProbeContext{Active: false, Timeout: 500 * time.Millisecond})
	if r.Status != LivenessFail {
		t.Fatalf("Status = %v, want Fail (%+v)", r.Status, r)
	}
}

func TestProbeProfile_Active_TakesFunctionalBranch(t *testing.T) {
	// Active=true with a dead SOCKS addr → functional branch → Fail. Proves it did
	// NOT silently fall back to a TCP reachability dial of the raw endpoint.
	p := config.Profile{Name: "hy2", Server: config.ServerConfig{Address: "1.2.3.4", Port: 443, Protocol: "hysteria2"}}
	r := ProbeProfile(p, ProbeContext{Active: true, SocksAddr: "127.0.0.1:1", LatencyHost: "1.1.1.1:443", Timeout: 500 * time.Millisecond})
	if r.Status != LivenessFail || r.Method != "functional" {
		t.Fatalf("got %+v, want Fail/functional", r)
	}
}
