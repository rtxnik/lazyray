package platform

import (
	"testing"
)

func TestLinuxPlatform_ClearQuarantine_NoOp(t *testing.T) {
	p := &linuxPlatform{}
	err := p.ClearQuarantine("/some/path")
	if err != nil {
		t.Errorf("ClearQuarantine on linux should be no-op, got error: %v", err)
	}
}

func TestLinuxPlatform_UnitPath(t *testing.T) {
	p := &linuxPlatform{}
	path := p.unitPath()
	if path == "" {
		t.Error("unitPath should not be empty")
	}
}

func TestLinuxPlatform_UnitDir(t *testing.T) {
	p := &linuxPlatform{}
	dir := p.unitDir()
	if dir == "" {
		t.Error("unitDir should not be empty")
	}
}

func TestDarwinPlatform_ClearQuarantine(t *testing.T) {
	// ClearQuarantine on darwin calls xattr -cr, which will fail on non-existent file
	p := &darwinPlatform{}
	err := p.ClearQuarantine("/nonexistent/file")
	// We expect an error since the file doesn't exist
	if err == nil {
		t.Log("ClearQuarantine succeeded (xattr might be available)")
	}
}

func TestWindowsPlatform_ClearQuarantine_NoOp(t *testing.T) {
	p := &windowsPlatform{}
	err := p.ClearQuarantine("/some/path")
	if err != nil {
		t.Errorf("ClearQuarantine on windows should be no-op, got error: %v", err)
	}
}
