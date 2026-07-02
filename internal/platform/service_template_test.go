package platform

import (
	"strings"
	"testing"
)

func TestSystemdUnit_RunsSupervisorAndRestartsOnFailure(t *testing.T) {
	unit, err := renderSystemdUnit("/usr/local/bin/lzr")
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if !strings.Contains(unit, "/usr/local/bin/lzr __run --owner service") {
		t.Errorf("ExecStart should run the supervisor:\n%s", unit)
	}
	if !strings.Contains(unit, "Restart=on-failure") {
		t.Errorf("Restart must be on-failure (so clean stop is not respawned):\n%s", unit)
	}
	if strings.Contains(unit, "Restart=always") {
		t.Errorf("Restart=always must be gone:\n%s", unit)
	}
}

func TestPlist_RunsSupervisorAndKeepAliveSuccessfulExitFalse(t *testing.T) {
	plist, err := renderPlist("/usr/local/bin/lzr")
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if !strings.Contains(plist, "<string>__run</string>") || !strings.Contains(plist, "<string>service</string>") {
		t.Errorf("ProgramArguments should be lzr __run --owner service:\n%s", plist)
	}
	if !strings.Contains(plist, "SuccessfulExit") {
		t.Errorf("KeepAlive must be a dict with SuccessfulExit=false:\n%s", plist)
	}
}
