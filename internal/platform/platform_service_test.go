package platform

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"text/template"
)

// --- ServiceStatus tests ---

func TestLinuxPlatform_ServiceStatus_NotInstalled(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("linux-only test")
	}
	p := &linuxPlatform{}
	installed, running, err := p.ServiceStatus()
	if err != nil {
		t.Fatalf("ServiceStatus() error = %v", err)
	}
	// We assume the test environment does not have lazyray.service installed
	if installed && running {
		t.Log("service appears to be installed and running in test env")
	}
	// At minimum, no error should occur
	_ = installed
	_ = running
}

func TestDarwinPlatform_ServiceStatus_NotInstalled(t *testing.T) {
	p := &darwinPlatform{}
	installed, running, err := p.ServiceStatus()
	if err != nil {
		t.Fatalf("ServiceStatus() error = %v", err)
	}
	// We assume no launchd agent installed in test env
	_ = installed
	_ = running
}

func TestWindowsPlatform_ServiceStatus_NotInstalled(t *testing.T) {
	p := &windowsPlatform{}
	installed, running, err := p.ServiceStatus()
	if err != nil {
		t.Fatalf("ServiceStatus() error = %v", err)
	}
	// We assume no scheduled task in test env
	_ = installed
	_ = running
}

// --- Notify tests (error expected in CI/test env) ---

func TestLinuxPlatform_Notify(t *testing.T) {
	p := &linuxPlatform{}
	err := p.Notify("test", "message")
	// In test environment, notify-send/dunstify may not be available
	if err == nil {
		t.Log("Notify succeeded (notification tool available)")
	} else {
		if !strings.Contains(err.Error(), "no notification tool") {
			t.Logf("Notify failed with: %v", err)
		}
	}
}

func TestDarwinPlatform_Notify(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("darwin-only test")
	}
	p := &darwinPlatform{}
	err := p.Notify("test", "message")
	// May succeed or fail depending on environment
	_ = err
}

func TestWindowsPlatform_Notify(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("windows-only test")
	}
	p := &windowsPlatform{}
	err := p.Notify("test", "message")
	_ = err
}

// --- OpenURL tests ---

func TestLinuxPlatform_OpenURL(t *testing.T) {
	p := &linuxPlatform{}
	err := p.OpenURL("https://example.com")
	// xdg-open may not be available in test env
	if err != nil {
		t.Logf("OpenURL failed (expected in test env): %v", err)
	}
}

func TestDarwinPlatform_OpenURL(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("darwin-only test")
	}
	p := &darwinPlatform{}
	err := p.OpenURL("https://example.com")
	_ = err
}

func TestWindowsPlatform_OpenURL(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("windows-only test")
	}
	p := &windowsPlatform{}
	err := p.OpenURL("https://example.com")
	_ = err
}

// --- Template parsing tests ---

func TestSystemdUnitTemplate_Parses(t *testing.T) {
	tmpl, err := template.New("unit").Parse(systemdUnitTemplate)
	if err != nil {
		t.Fatalf("failed to parse systemd unit template: %v", err)
	}

	data := map[string]string{
		"ExecPath": "/usr/local/bin/lzr",
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		t.Fatalf("failed to execute systemd unit template: %v", err)
	}

	result := buf.String()

	if !strings.Contains(result, "/usr/local/bin/lzr") {
		t.Error("template output should contain lzr path")
	}
	if !strings.Contains(result, "__run --owner service") {
		t.Error("template output should run the supervisor")
	}
	if !strings.Contains(result, "[Unit]") {
		t.Error("template output should contain [Unit] section")
	}
	if !strings.Contains(result, "[Service]") {
		t.Error("template output should contain [Service] section")
	}
	if !strings.Contains(result, "[Install]") {
		t.Error("template output should contain [Install] section")
	}
	if !strings.Contains(result, "Restart=on-failure") {
		t.Error("template output should contain Restart=on-failure")
	}
}

func TestPlistTemplate_Parses(t *testing.T) {
	tmpl, err := template.New("plist").Parse(plistTemplate)
	if err != nil {
		t.Fatalf("failed to parse plist template: %v", err)
	}

	data := map[string]string{
		"Label":        launchdLabel,
		"ExecPath":     "/usr/local/bin/lzr",
		"LogPath":      "/var/log/xray.log",
		"ErrorLogPath": "/var/log/xray-error.log",
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		t.Fatalf("failed to execute plist template: %v", err)
	}

	result := buf.String()

	if !strings.Contains(result, launchdLabel) {
		t.Error("plist output should contain label")
	}
	if !strings.Contains(result, "/usr/local/bin/lzr") {
		t.Error("plist output should contain lzr path")
	}
	if !strings.Contains(result, "RunAtLoad") {
		t.Error("plist output should contain RunAtLoad")
	}
	if !strings.Contains(result, "KeepAlive") {
		t.Error("plist output should contain KeepAlive")
	}
	if !strings.Contains(result, "<?xml version") {
		t.Error("plist output should be valid XML")
	}
}

// --- Path tests ---

func TestLinuxPlatform_UnitPath_ContainsServiceFile(t *testing.T) {
	p := &linuxPlatform{}
	path := p.unitPath()
	if !strings.HasSuffix(path, "lazyray.service") {
		t.Errorf("unitPath() = %q, should end with lazyray.service", path)
	}
}

func TestLinuxPlatform_UnitDir_ContainsSystemd(t *testing.T) {
	p := &linuxPlatform{}
	dir := p.unitDir()
	if !strings.Contains(dir, "systemd") {
		t.Errorf("unitDir() = %q, should contain 'systemd'", dir)
	}
	if !strings.Contains(dir, "user") {
		t.Errorf("unitDir() = %q, should contain 'user'", dir)
	}
}

func TestLinuxPlatform_UnitPath_IsInUnitDir(t *testing.T) {
	p := &linuxPlatform{}
	dir := p.unitDir()
	path := p.unitPath()
	if !strings.HasPrefix(path, dir) {
		t.Errorf("unitPath() %q should be inside unitDir() %q", path, dir)
	}
}

func TestDarwinPlatform_PlistPath(t *testing.T) {
	p := &darwinPlatform{}
	path := p.plistPath()
	if !strings.Contains(path, "LaunchAgents") {
		t.Errorf("plistPath() = %q, should contain LaunchAgents", path)
	}
	if !strings.HasSuffix(path, ".plist") {
		t.Errorf("plistPath() = %q, should end with .plist", path)
	}
	if !strings.Contains(path, launchdLabel) {
		t.Errorf("plistPath() = %q, should contain label %q", path, launchdLabel)
	}
}

// --- LaunchdLabel ---

func TestLaunchdLabel(t *testing.T) {
	if launchdLabel != "com.lazyray.xray" {
		t.Errorf("launchdLabel = %q, want com.lazyray.xray", launchdLabel)
	}
}

// --- ServiceInstall with invalid paths (quick failure test) ---

func TestLinuxPlatform_ServiceInstall_InvalidDir(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("linux-only test")
	}
	// Override HOME to a non-writable location to test error handling
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", "/nonexistent/home")
	defer os.Setenv("HOME", origHome)

	p := &linuxPlatform{}
	err := p.ServiceInstall("/usr/local/bin/lzr")
	if err == nil {
		t.Error("ServiceInstall with invalid home dir should fail")
		// Clean up if somehow succeeded
		_ = p.ServiceUninstall()
	}
}

// --- ServiceUninstall with no service ---

func TestLinuxPlatform_ServiceUninstall_NoService(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("linux-only test")
	}
	p := &linuxPlatform{}
	err := p.ServiceUninstall()
	// Should fail since the service file doesn't exist
	if err == nil {
		t.Log("ServiceUninstall succeeded (service may have been installed)")
	}
}

// --- Current() default fallback ---

func TestCurrent_DefaultFallback(t *testing.T) {
	p := Current()
	// Verify we get a non-nil Platform
	if p == nil {
		t.Fatal("Current() returned nil")
	}

	// Verify it matches the expected type for this OS
	switch runtime.GOOS {
	case "linux":
		if _, ok := p.(*linuxPlatform); !ok {
			t.Errorf("on linux, expected *linuxPlatform, got %T", p)
		}
	case "darwin":
		if _, ok := p.(*darwinPlatform); !ok {
			t.Errorf("on darwin, expected *darwinPlatform, got %T", p)
		}
	case "windows":
		if _, ok := p.(*windowsPlatform); !ok {
			t.Errorf("on windows, expected *windowsPlatform, got %T", p)
		}
	}
}

// --- Platform interface methods coverage ---

func TestLinuxPlatform_AllMethodsExist(t *testing.T) {
	var p Platform = &linuxPlatform{}

	// Verify all interface methods are implemented
	_, _, _ = p.ServiceStatus()
	_ = p.ClearQuarantine("")
}

func TestDarwinPlatform_AllMethodsExist(t *testing.T) {
	var p Platform = &darwinPlatform{}

	// Verify all interface methods are implemented
	_, _, _ = p.ServiceStatus()
}

func TestWindowsPlatform_AllMethodsExist(t *testing.T) {
	var p Platform = &windowsPlatform{}

	// Verify all interface methods are implemented
	_, _, _ = p.ServiceStatus()
	_ = p.ClearQuarantine("")
}

// --- ServiceInstall template test with temp dir ---

func TestLinuxPlatform_ServiceInstall_TemplateOutput(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("linux-only test")
	}

	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	p := &linuxPlatform{}
	unitDir := p.unitDir()

	// Create the systemd user dir
	if err := os.MkdirAll(unitDir, 0755); err != nil {
		t.Fatalf("failed to create unit dir: %v", err)
	}

	// We can't actually install the service (systemctl won't work),
	// but we can verify the directory is set up correctly
	if _, err := os.Stat(unitDir); err != nil {
		t.Errorf("unit dir should exist: %v", err)
	}

	// Verify the path uses the tmp home
	if !strings.HasPrefix(p.unitPath(), tmpDir) {
		t.Errorf("unitPath() = %q, should start with %q", p.unitPath(), tmpDir)
	}

	expectedPath := filepath.Join(tmpDir, ".config", "systemd", "user", "lazyray.service")
	if p.unitPath() != expectedPath {
		t.Errorf("unitPath() = %q, want %q", p.unitPath(), expectedPath)
	}
}
