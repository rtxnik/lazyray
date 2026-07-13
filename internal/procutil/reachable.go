package procutil

import (
	"net"
	"time"
)

// Reachable dials addr over TCP with the given timeout and returns nil if the
// connect succeeds (the connection is closed immediately), else the dial error.
// The caller owns timeout policy (e.g. defaulting a non-positive value).
func Reachable(addr string, timeout time.Duration) error {
	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return err
	}
	_ = conn.Close()
	return nil
}
