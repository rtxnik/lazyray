package platform

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
)

type linuxPlatform struct{}

const systemdUnitTemplate = `[Unit]
Description=lazyray Xray supervisor
After=network.target

[Service]
ExecStart={{.ExecPath}} __run --owner service
Restart=on-failure
RestartSec=5

[Install]
WantedBy=default.target`

// renderSystemdUnit returns the unit file contents for the given lzr path.
func renderSystemdUnit(execPath string) (string, error) {
	tmpl, err := template.New("unit").Parse(systemdUnitTemplate)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	if err := tmpl.Execute(&b, map[string]string{"ExecPath": execPath}); err != nil {
		return "", err
	}
	return b.String(), nil
}

func (l *linuxPlatform) unitDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "systemd", "user")
}

func (l *linuxPlatform) unitPath() string {
	return filepath.Join(l.unitDir(), "lazyray.service")
}

func (l *linuxPlatform) ServiceInstall(execPath string) error {
	if err := os.MkdirAll(l.unitDir(), 0755); err != nil {
		return fmt.Errorf("creating systemd user dir: %w", err)
	}

	unit, err := renderSystemdUnit(execPath)
	if err != nil {
		return fmt.Errorf("rendering unit: %w", err)
	}

	if err := os.WriteFile(l.unitPath(), []byte(unit), 0644); err != nil {
		return fmt.Errorf("writing unit file: %w", err)
	}

	if err := exec.Command("systemctl", "--user", "daemon-reload").Run(); err != nil {
		return fmt.Errorf("daemon-reload: %w", err)
	}

	return exec.Command("systemctl", "--user", "enable", "--now", "lazyray.service").Run()
}

func (l *linuxPlatform) ServiceUninstall() error {
	_ = exec.Command("systemctl", "--user", "disable", "--now", "lazyray.service").Run()
	_ = exec.Command("systemctl", "--user", "daemon-reload").Run()
	return os.Remove(l.unitPath())
}

func (l *linuxPlatform) ServiceStatus() (bool, bool, error) {
	if _, err := os.Stat(l.unitPath()); os.IsNotExist(err) {
		return false, false, nil
	}

	out, err := exec.Command("systemctl", "--user", "is-active", "lazyray.service").Output()
	if err != nil {
		return true, false, nil
	}

	active := strings.TrimSpace(string(out)) == "active"
	return true, active, nil
}

func (l *linuxPlatform) Notify(title, message string) error {
	// Try notify-send first, fall back to dunstify
	if path, err := exec.LookPath("notify-send"); err == nil {
		return exec.Command(path, title, message).Run()
	}
	if path, err := exec.LookPath("dunstify"); err == nil {
		return exec.Command(path, title, message).Run()
	}
	return fmt.Errorf("no notification tool available (install notify-send or dunstify)")
}

func (l *linuxPlatform) ClearQuarantine(_ string) error {
	return nil
}

func (l *linuxPlatform) OpenURL(url string) error {
	return exec.Command("xdg-open", url).Run()
}
