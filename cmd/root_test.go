// cmd/root_test.go
package cmd

import "testing"

func TestRootCmd(t *testing.T) {
	c := RootCmd()
	if c == nil {
		t.Fatal("RootCmd() returned nil")
	}
	if c.Use != "lzr" {
		t.Errorf("RootCmd().Use = %q, want %q", c.Use, "lzr")
	}
}
