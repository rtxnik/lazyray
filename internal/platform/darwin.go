package platform

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/rtxnik/lazyray/internal/config"
)

type darwinPlatform struct{}

const launchdLabel = "com.lazyray.xray"

const plistTemplate = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key><string>{{.Label}}</string>
    <key>ProgramArguments</key>
    <array>
        <string>{{.ExecPath}}</string>
        <string>__run</string>
        <string>--owner</string>
        <string>service</string>
    </array>
    <key>RunAtLoad</key><true/>
    <key>KeepAlive</key>
    <dict>
        <key>SuccessfulExit</key><false/>
    </dict>
    <key>StandardOutPath</key><string>{{.LogPath}}</string>
    <key>StandardErrorPath</key><string>{{.ErrorLogPath}}</string>
</dict>
</plist>`

func (d *darwinPlatform) plistPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Library", "LaunchAgents", launchdLabel+".plist")
}

// renderPlist returns the plist contents for the given lzr path.
func renderPlist(execPath string) (string, error) {
	tmpl, err := template.New("plist").Parse(plistTemplate)
	if err != nil {
		return "", err
	}
	logDir := config.LogDir()
	var b strings.Builder
	if err := tmpl.Execute(&b, map[string]string{
		"Label":        launchdLabel,
		"ExecPath":     execPath,
		"LogPath":      filepath.Join(logDir, "xray.log"),
		"ErrorLogPath": filepath.Join(logDir, "xray-error.log"),
	}); err != nil {
		return "", err
	}
	return b.String(), nil
}

func (d *darwinPlatform) ServiceInstall(execPath string) error {
	plist, err := renderPlist(execPath)
	if err != nil {
		return fmt.Errorf("rendering plist: %w", err)
	}
	if err := os.WriteFile(d.plistPath(), []byte(plist), 0644); err != nil {
		return fmt.Errorf("writing plist: %w", err)
	}
	return exec.Command("launchctl", "load", d.plistPath()).Run()
}

func (d *darwinPlatform) ServiceUninstall() error {
	path := d.plistPath()
	_ = exec.Command("launchctl", "unload", path).Run()
	return os.Remove(path)
}

func (d *darwinPlatform) ServiceStatus() (bool, bool, error) {
	path := d.plistPath()
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return false, false, nil
	}

	out, err := exec.Command("launchctl", "list", launchdLabel).Output()
	if err != nil {
		return true, false, nil
	}
	_ = out
	return true, true, nil
}

func (d *darwinPlatform) Notify(title, message string) error {
	// Prefer terminal-notifier if available (richer notifications)
	if path, err := exec.LookPath("terminal-notifier"); err == nil {
		return exec.Command(path, "-title", title, "-message", message, "-group", "lazyray").Run()
	}
	script := fmt.Sprintf(`display notification "%s" with title "%s"`, message, title)
	return exec.Command("osascript", "-e", script).Run()
}

func (d *darwinPlatform) ClearQuarantine(path string) error {
	return exec.Command("xattr", "-cr", path).Run()
}

func (d *darwinPlatform) OpenURL(url string) error {
	return exec.Command("open", url).Run()
}
