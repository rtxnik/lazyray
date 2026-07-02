package platform

import (
	"runtime"
	"testing"
)

func TestCurrent_ReturnsNonNil(t *testing.T) {
	p := Current()
	if p == nil {
		t.Fatal("Current() returned nil")
	}
}

func TestCurrent_ReturnsCorrectType(t *testing.T) {
	p := Current()
	switch runtime.GOOS {
	case "darwin":
		if _, ok := p.(*darwinPlatform); !ok {
			t.Errorf("on darwin, Current() should return *darwinPlatform, got %T", p)
		}
	case "linux":
		if _, ok := p.(*linuxPlatform); !ok {
			t.Errorf("on linux, Current() should return *linuxPlatform, got %T", p)
		}
	case "windows":
		if _, ok := p.(*windowsPlatform); !ok {
			t.Errorf("on windows, Current() should return *windowsPlatform, got %T", p)
		}
	}
}

func TestPlatform_ImplementsInterface(t *testing.T) {
	// Verify all platform types implement the Platform interface at compile time
	var _ Platform = &darwinPlatform{}
	var _ Platform = &linuxPlatform{}
	var _ Platform = &windowsPlatform{}
}
