package procutil

import (
	"net"
	"testing"
	"time"
)

func TestReachable_OpenPort(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()
	if err := Reachable(ln.Addr().String(), time.Second); err != nil {
		t.Errorf("Reachable(open) = %v, want nil", err)
	}
}

func TestReachable_ClosedPort(t *testing.T) {
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	addr := ln.Addr().String()
	ln.Close() // now nobody is listening
	if err := Reachable(addr, 300*time.Millisecond); err == nil {
		t.Error("Reachable(closed) = nil, want error")
	}
}
