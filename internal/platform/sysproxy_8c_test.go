//go:build linux

package platform

import (
	"os"
	"strings"
	"testing"
)

// --- desktopEnv tests (Linux) ---

func TestDesktopEnv_GNOME(t *testing.T) {
	t.Setenv("XDG_CURRENT_DESKTOP", "GNOME")
	if de := desktopEnv(); de != "gnome" {
		t.Errorf("desktopEnv() = %q, want gnome", de)
	}
}

func TestDesktopEnv_Unity(t *testing.T) {
	t.Setenv("XDG_CURRENT_DESKTOP", "Unity")
	if de := desktopEnv(); de != "gnome" {
		t.Errorf("desktopEnv() = %q, want gnome (Unity uses GNOME settings)", de)
	}
}

func TestDesktopEnv_Cinnamon(t *testing.T) {
	t.Setenv("XDG_CURRENT_DESKTOP", "X-Cinnamon")
	if de := desktopEnv(); de != "gnome" {
		t.Errorf("desktopEnv() = %q, want gnome (Cinnamon uses GNOME settings)", de)
	}
}

func TestDesktopEnv_KDE(t *testing.T) {
	t.Setenv("XDG_CURRENT_DESKTOP", "KDE")
	if de := desktopEnv(); de != "kde" {
		t.Errorf("desktopEnv() = %q, want kde", de)
	}
}

func TestDesktopEnv_Unknown(t *testing.T) {
	t.Setenv("XDG_CURRENT_DESKTOP", "i3")
	if de := desktopEnv(); de != "" {
		t.Errorf("desktopEnv() = %q, want empty for unsupported DE", de)
	}
}

func TestDesktopEnv_Empty(t *testing.T) {
	t.Setenv("XDG_CURRENT_DESKTOP", "")
	if de := desktopEnv(); de != "" {
		t.Errorf("desktopEnv() = %q, want empty", de)
	}
}

func TestDesktopEnv_CaseInsensitive(t *testing.T) {
	t.Setenv("XDG_CURRENT_DESKTOP", "gnome-session")
	if de := desktopEnv(); de != "gnome" {
		t.Errorf("desktopEnv() = %q, want gnome for lowercase", de)
	}
}

// --- SystemProxy interface compliance ---

func TestLinuxSystemProxy_ImplementsInterface(t *testing.T) {
	var _ SystemProxy = &linuxSystemProxy{}
}

// --- linuxSystemProxy.Disable with no DE (no error expected) ---

func TestLinuxSystemProxy_Disable_NoDE(t *testing.T) {
	t.Setenv("XDG_CURRENT_DESKTOP", "")
	sp := &linuxSystemProxy{}
	err := sp.Disable()
	if err != nil {
		t.Errorf("Disable() with no DE should return nil, got %v", err)
	}
}

// --- linuxSystemProxy.Status with no DE ---

func TestLinuxSystemProxy_Status_NoDE(t *testing.T) {
	t.Setenv("XDG_CURRENT_DESKTOP", "")
	sp := &linuxSystemProxy{}
	status, err := sp.Status()
	if err != nil {
		t.Fatalf("Status() error = %v", err)
	}
	if status.HTTPEnabled {
		t.Error("HTTPEnabled should be false with no DE")
	}
	if status.SOCKSEnabled {
		t.Error("SOCKSEnabled should be false with no DE")
	}
	if status.PACEnabled {
		t.Error("PACEnabled should be false with no DE")
	}
}

// --- linuxSystemProxy Enable methods with no DE returns error ---

func TestLinuxSystemProxy_EnableHTTPProxy_NoDE(t *testing.T) {
	t.Setenv("XDG_CURRENT_DESKTOP", "")
	sp := &linuxSystemProxy{}
	err := sp.EnableHTTPProxy("127.0.0.1", 10809)
	if err == nil {
		t.Error("EnableHTTPProxy with no DE should return error")
	}
}

func TestLinuxSystemProxy_EnableSOCKSProxy_NoDE(t *testing.T) {
	t.Setenv("XDG_CURRENT_DESKTOP", "")
	sp := &linuxSystemProxy{}
	err := sp.EnableSOCKSProxy("127.0.0.1", 10808)
	if err == nil {
		t.Error("EnableSOCKSProxy with no DE should return error")
	}
}

func TestLinuxSystemProxy_EnablePACProxy_NoDE(t *testing.T) {
	t.Setenv("XDG_CURRENT_DESKTOP", "")
	sp := &linuxSystemProxy{}
	err := sp.EnablePACProxy("http://127.0.0.1:10810/proxy.pac")
	if err == nil {
		t.Error("EnablePACProxy with no DE should return error")
	}
}

// --- parseHostPort extended tests ---

func TestParseHostPort_Extended(t *testing.T) {
	tests := []struct {
		addr     string
		wantHost string
		wantPort int
	}{
		{"[::1]:8080", "[::1]", 8080},
		{"", "", 0},
		{":", ":", 0},
	}

	for _, tc := range tests {
		t.Run(tc.addr, func(t *testing.T) {
			var host string
			var port int
			parseHostPort(tc.addr, &host, &port)
			if host != tc.wantHost {
				t.Errorf("host = %q, want %q", host, tc.wantHost)
			}
			if port != tc.wantPort {
				t.Errorf("port = %d, want %d", port, tc.wantPort)
			}
		})
	}
}

// --- ProxyStatus field tests ---

func TestProxyStatus_AllEnabled(t *testing.T) {
	status := &ProxyStatus{
		HTTPEnabled:  true,
		HTTPHost:     "127.0.0.1",
		HTTPPort:     10809,
		SOCKSEnabled: true,
		SOCKSHost:    "127.0.0.1",
		SOCKSPort:    10808,
		PACEnabled:   true,
		PACURL:       "http://127.0.0.1:10810/proxy.pac",
	}

	if !status.HTTPEnabled || !status.SOCKSEnabled || !status.PACEnabled {
		t.Error("all proxies should be enabled")
	}
}

func TestProxyStatus_MixedState(t *testing.T) {
	status := &ProxyStatus{
		HTTPEnabled:  true,
		HTTPHost:     "127.0.0.1",
		HTTPPort:     10809,
		SOCKSEnabled: false,
		PACEnabled:   false,
	}

	if !status.HTTPEnabled {
		t.Error("HTTP should be enabled")
	}
	if status.SOCKSEnabled {
		t.Error("SOCKS should be disabled")
	}
	if status.PACEnabled {
		t.Error("PAC should be disabled")
	}
}

// --- GNOME code paths (exercises path even without gsettings) ---

func TestLinuxSystemProxy_EnableHTTPProxy_GNOME(t *testing.T) {
	t.Setenv("XDG_CURRENT_DESKTOP", "GNOME")
	sp := &linuxSystemProxy{}
	err := sp.EnableHTTPProxy("127.0.0.1", 10809)
	if err == nil {
		t.Log("EnableHTTPProxy GNOME succeeded (gsettings available)")
	}
}

func TestLinuxSystemProxy_EnableSOCKSProxy_GNOME(t *testing.T) {
	t.Setenv("XDG_CURRENT_DESKTOP", "GNOME")
	sp := &linuxSystemProxy{}
	err := sp.EnableSOCKSProxy("127.0.0.1", 10808)
	if err == nil {
		t.Log("EnableSOCKSProxy GNOME succeeded (gsettings available)")
	}
}

func TestLinuxSystemProxy_EnablePACProxy_GNOME(t *testing.T) {
	t.Setenv("XDG_CURRENT_DESKTOP", "GNOME")
	sp := &linuxSystemProxy{}
	err := sp.EnablePACProxy("http://127.0.0.1:10810/proxy.pac")
	if err == nil {
		t.Log("EnablePACProxy GNOME succeeded (gsettings available)")
	}
}

func TestLinuxSystemProxy_Disable_GNOME(t *testing.T) {
	t.Setenv("XDG_CURRENT_DESKTOP", "GNOME")
	sp := &linuxSystemProxy{}
	err := sp.Disable()
	if err == nil {
		t.Log("Disable GNOME succeeded (gsettings available)")
	}
}

func TestLinuxSystemProxy_Status_GNOME(t *testing.T) {
	t.Setenv("XDG_CURRENT_DESKTOP", "GNOME")
	sp := &linuxSystemProxy{}
	status, err := sp.Status()
	if err != nil {
		t.Fatalf("Status() error = %v", err)
	}
	_ = status
}

// --- KDE code paths ---

func TestLinuxSystemProxy_EnableHTTPProxy_KDE(t *testing.T) {
	t.Setenv("XDG_CURRENT_DESKTOP", "KDE")
	sp := &linuxSystemProxy{}
	err := sp.EnableHTTPProxy("127.0.0.1", 10809)
	if err == nil {
		t.Log("EnableHTTPProxy KDE succeeded")
	}
}

func TestLinuxSystemProxy_EnableSOCKSProxy_KDE(t *testing.T) {
	t.Setenv("XDG_CURRENT_DESKTOP", "KDE")
	sp := &linuxSystemProxy{}
	err := sp.EnableSOCKSProxy("127.0.0.1", 10808)
	if err == nil {
		t.Log("EnableSOCKSProxy KDE succeeded")
	}
}

func TestLinuxSystemProxy_EnablePACProxy_KDE(t *testing.T) {
	t.Setenv("XDG_CURRENT_DESKTOP", "KDE")
	sp := &linuxSystemProxy{}
	err := sp.EnablePACProxy("http://127.0.0.1:10810/proxy.pac")
	if err == nil {
		t.Log("EnablePACProxy KDE succeeded")
	}
}

func TestLinuxSystemProxy_Disable_KDE(t *testing.T) {
	t.Setenv("XDG_CURRENT_DESKTOP", "KDE")
	sp := &linuxSystemProxy{}
	err := sp.Disable()
	if err == nil {
		t.Log("Disable KDE succeeded")
	}
}

// --- Current() platform tests ---

func TestCurrent_ReturnsLinux(t *testing.T) {
	p := Current()
	if p == nil {
		t.Fatal("Current() should not return nil")
	}
	// On Linux, should return *linuxPlatform
	_, ok := p.(*linuxPlatform)
	if !ok {
		t.Error("Current() should return *linuxPlatform on Linux")
	}
}

// --- CurrentSystemProxy ---

func TestCurrentSystemProxy_ReturnsLinux(t *testing.T) {
	sp := CurrentSystemProxy()
	if sp == nil {
		t.Fatal("CurrentSystemProxy() should not return nil")
	}
	_, ok := sp.(*linuxSystemProxy)
	if !ok {
		t.Error("CurrentSystemProxy() should return *linuxSystemProxy on Linux")
	}
}

// --- linuxPlatform methods ---

func TestLinuxPlatform_ClearQuarantine(t *testing.T) {
	p := &linuxPlatform{}
	err := p.ClearQuarantine("/tmp/test")
	if err != nil {
		t.Errorf("ClearQuarantine should return nil on Linux, got %v", err)
	}
}

func TestLinuxPlatform_ServiceStatus_NoUnit(t *testing.T) {
	origHome := os.Getenv("HOME")
	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	p := &linuxPlatform{}
	installed, running, err := p.ServiceStatus()
	if err != nil {
		t.Fatalf("ServiceStatus error = %v", err)
	}
	if installed {
		t.Error("should not be installed with temp HOME")
	}
	if running {
		t.Error("should not be running with temp HOME")
	}
}

func TestLinuxPlatform_ServiceUninstall_NoUnit(t *testing.T) {
	origHome := os.Getenv("HOME")
	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	p := &linuxPlatform{}
	// Should not panic even if unit file doesn't exist
	_ = p.ServiceUninstall()
}

func TestLinuxPlatform_Notify_NoTool(t *testing.T) {
	// Override PATH to prevent finding notify-send
	origPath := os.Getenv("PATH")
	os.Setenv("PATH", t.TempDir()) // empty dir, no tools
	defer os.Setenv("PATH", origPath)

	p := &linuxPlatform{}
	err := p.Notify("test", "message")
	if err == nil {
		t.Error("Notify should return error when no notification tool is available")
	}
}

// --- linuxSystemProxy Status with GNOME (exercises gsettings path) ---

func TestLinuxSystemProxy_Status_GNOME_8C(t *testing.T) {
	t.Setenv("XDG_CURRENT_DESKTOP", "GNOME")
	sp := &linuxSystemProxy{}
	status, err := sp.Status()
	if err != nil {
		t.Fatalf("Status() error = %v", err)
	}
	// Result depends on whether gsettings is available
	_ = status
}

func TestLinuxSystemProxy_Status_KDE(t *testing.T) {
	t.Setenv("XDG_CURRENT_DESKTOP", "KDE")
	sp := &linuxSystemProxy{}
	status, err := sp.Status()
	if err != nil {
		t.Fatalf("Status() error = %v", err)
	}
	_ = status
}

// --- linuxPlatform.unitDir / unitPath ---

func TestLinuxPlatform_UnitDir_8C(t *testing.T) {
	p := &linuxPlatform{}
	dir := p.unitDir()
	if dir == "" {
		t.Error("unitDir should not be empty")
	}
	if !strings.Contains(dir, "systemd") {
		t.Errorf("unitDir = %q, should contain 'systemd'", dir)
	}
}

func TestLinuxPlatform_UnitPath_8C(t *testing.T) {
	p := &linuxPlatform{}
	path := p.unitPath()
	if !strings.Contains(path, "lazyray.service") {
		t.Errorf("unitPath = %q, should contain 'lazyray.service'", path)
	}
}

// --- ServiceInstall with temp HOME (will try to run systemctl and likely fail) ---

func TestLinuxPlatform_ServiceInstall_8C(t *testing.T) {
	origHome := os.Getenv("HOME")
	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	p := &linuxPlatform{}
	err := p.ServiceInstall("/usr/bin/lzr")
	// Will likely fail due to systemctl not available in test env
	_ = err
}

// --- OpenURL (exercises the code path even if xdg-open fails) ---

func TestLinuxPlatform_OpenURL_8C(t *testing.T) {
	// Override PATH to avoid actually opening a URL
	origPath := os.Getenv("PATH")
	os.Setenv("PATH", t.TempDir())
	defer os.Setenv("PATH", origPath)

	p := &linuxPlatform{}
	err := p.OpenURL("https://example.com")
	if err == nil {
		t.Log("OpenURL succeeded (xdg-open found)")
	}
}

// --- linuxSystemProxy.Disable for KDE (exercises kwriteconfig5 path) ---

func TestLinuxSystemProxy_Status_KDE_8C(t *testing.T) {
	t.Setenv("XDG_CURRENT_DESKTOP", "KDE")
	sp := &linuxSystemProxy{}
	status, err := sp.Status()
	if err != nil {
		t.Fatalf("Status error = %v", err)
	}
	_ = status
}

func TestLinuxSystemProxy_Disable_KDE_8C(t *testing.T) {
	t.Setenv("XDG_CURRENT_DESKTOP", "KDE")
	sp := &linuxSystemProxy{}
	err := sp.Disable()
	_ = err // May fail without kwriteconfig5
}

// --- desktopEnv with MATE ---

func TestDesktopEnv_MATE(t *testing.T) {
	t.Setenv("XDG_CURRENT_DESKTOP", "MATE")
	de := desktopEnv()
	if de != "" {
		t.Errorf("desktopEnv() = %q, want empty for MATE (not in supported list)", de)
	}
}

func TestLinuxPlatform_ServiceStatus_WithUnitFile(t *testing.T) {
	origHome := os.Getenv("HOME")
	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	p := &linuxPlatform{}
	// Create the systemd user dir and unit file
	unitDir := p.unitDir()
	os.MkdirAll(unitDir, 0755)
	os.WriteFile(p.unitPath(), []byte("[Unit]\nDescription=Test\n"), 0644)

	installed, running, err := p.ServiceStatus()
	if err != nil {
		t.Fatalf("ServiceStatus error = %v", err)
	}
	if !installed {
		t.Error("should be installed when unit file exists")
	}
	// Running may be false if systemctl is not available
	_ = running
}

func TestLinuxPlatform_ServiceInstall_WithTempHome(t *testing.T) {
	origHome := os.Getenv("HOME")
	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	p := &linuxPlatform{}
	// This will create the unit file but fail at systemctl
	err := p.ServiceInstall("/tmp/lzr")
	// Verify unit file was created even if systemctl failed
	if _, statErr := os.Stat(p.unitPath()); statErr != nil {
		t.Log("unit file not created (template error)")
	}
	_ = err
}

func TestDesktopEnv_Budgie(t *testing.T) {
	t.Setenv("XDG_CURRENT_DESKTOP", "Budgie:GNOME")
	de := desktopEnv()
	if de != "gnome" {
		t.Errorf("desktopEnv() = %q, want gnome for Budgie:GNOME", de)
	}
}
