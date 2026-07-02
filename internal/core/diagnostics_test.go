package core

import (
	"os"
	"testing"
)

func TestScanXrayPID_DelegatesToFindXrayPID(t *testing.T) {
	orig := findXrayPID
	t.Cleanup(func() { findXrayPID = orig })

	findXrayPID = func() int { return 4242 }
	if got := ScanXrayPID(); got != 4242 {
		t.Fatalf("ScanXrayPID() = %d, want 4242 (must delegate to findXrayPID)", got)
	}

	findXrayPID = func() int { return 0 }
	if got := ScanXrayPID(); got != 0 {
		t.Fatalf("ScanXrayPID() = %d, want 0 when no xray found", got)
	}
}

func TestIsProcessAlive_ExportedCurrentProcess(t *testing.T) {
	if !IsProcessAlive(os.Getpid()) {
		t.Errorf("IsProcessAlive(self=%d) = false, want true", os.Getpid())
	}
}

func TestIsProcessAlive_ExportedImpossiblePID(t *testing.T) {
	if IsProcessAlive(-1) {
		t.Error("IsProcessAlive(-1) = true, want false")
	}
}
