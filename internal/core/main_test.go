package core

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	// Mock findXrayPID to prevent tests from detecting a real xray
	// process running on the host machine.
	findXrayPID = func() int { return 0 }
	os.Exit(m.Run())
}
