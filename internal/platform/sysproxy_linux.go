package platform

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

type linuxSystemProxy struct{}

func currentSystemProxy() SystemProxy {
	return &linuxSystemProxy{}
}

// desktopEnv detects the current desktop environment.
func desktopEnv() string {
	de := os.Getenv("XDG_CURRENT_DESKTOP")
	de = strings.ToLower(de)
	if strings.Contains(de, "gnome") || strings.Contains(de, "unity") || strings.Contains(de, "cinnamon") {
		return "gnome"
	}
	if strings.Contains(de, "kde") {
		return "kde"
	}
	return ""
}

func (l *linuxSystemProxy) EnableHTTPProxy(host string, port int) error {
	addr := fmt.Sprintf("http://%s:%d", host, port)

	switch desktopEnv() {
	case "gnome":
		cmds := [][]string{
			{"gsettings", "set", "org.gnome.system.proxy", "mode", "manual"},
			{"gsettings", "set", "org.gnome.system.proxy.http", "host", host},
			{"gsettings", "set", "org.gnome.system.proxy.http", "port", strconv.Itoa(port)},
			{"gsettings", "set", "org.gnome.system.proxy.https", "host", host},
			{"gsettings", "set", "org.gnome.system.proxy.https", "port", strconv.Itoa(port)},
		}
		for _, c := range cmds {
			if err := exec.Command(c[0], c[1:]...).Run(); err != nil {
				return fmt.Errorf("gsettings %s: %w", c[3], err)
			}
		}
		return nil

	case "kde":
		cmds := [][]string{
			{"kwriteconfig5", "--file", "kioslaverc", "--group", "Proxy Settings", "--key", "ProxyType", "1"},
			{"kwriteconfig5", "--file", "kioslaverc", "--group", "Proxy Settings", "--key", "httpProxy", addr},
			{"kwriteconfig5", "--file", "kioslaverc", "--group", "Proxy Settings", "--key", "httpsProxy", addr},
		}
		for _, c := range cmds {
			if err := exec.Command(c[0], c[1:]...).Run(); err != nil {
				return fmt.Errorf("kwriteconfig5: %w", err)
			}
		}
		return nil
	}

	// Fallback: print env vars advice
	return fmt.Errorf("no supported desktop environment detected; set http_proxy=%s and https_proxy=%s manually: %w", addr, addr, ErrNoDesktopEnv)
}

func (l *linuxSystemProxy) EnableSOCKSProxy(host string, port int) error {
	addr := fmt.Sprintf("socks5://%s:%d", host, port)

	switch desktopEnv() {
	case "gnome":
		cmds := [][]string{
			{"gsettings", "set", "org.gnome.system.proxy", "mode", "manual"},
			{"gsettings", "set", "org.gnome.system.proxy.socks", "host", host},
			{"gsettings", "set", "org.gnome.system.proxy.socks", "port", strconv.Itoa(port)},
		}
		for _, c := range cmds {
			if err := exec.Command(c[0], c[1:]...).Run(); err != nil {
				return fmt.Errorf("gsettings: %w", err)
			}
		}
		return nil

	case "kde":
		cmds := [][]string{
			{"kwriteconfig5", "--file", "kioslaverc", "--group", "Proxy Settings", "--key", "ProxyType", "1"},
			{"kwriteconfig5", "--file", "kioslaverc", "--group", "Proxy Settings", "--key", "socksProxy", addr},
		}
		for _, c := range cmds {
			if err := exec.Command(c[0], c[1:]...).Run(); err != nil {
				return fmt.Errorf("kwriteconfig5: %w", err)
			}
		}
		return nil
	}

	return fmt.Errorf("no supported desktop environment detected; set all_proxy=%s manually: %w", addr, ErrNoDesktopEnv)
}

func (l *linuxSystemProxy) EnablePACProxy(pacURL string) error {
	switch desktopEnv() {
	case "gnome":
		cmds := [][]string{
			{"gsettings", "set", "org.gnome.system.proxy", "mode", "auto"},
			{"gsettings", "set", "org.gnome.system.proxy", "autoconfig-url", pacURL},
		}
		for _, c := range cmds {
			if err := exec.Command(c[0], c[1:]...).Run(); err != nil {
				return fmt.Errorf("gsettings: %w", err)
			}
		}
		return nil

	case "kde":
		cmds := [][]string{
			{"kwriteconfig5", "--file", "kioslaverc", "--group", "Proxy Settings", "--key", "ProxyType", "2"},
			{"kwriteconfig5", "--file", "kioslaverc", "--group", "Proxy Settings", "--key", "Proxy Config Script", pacURL},
		}
		for _, c := range cmds {
			if err := exec.Command(c[0], c[1:]...).Run(); err != nil {
				return fmt.Errorf("kwriteconfig5: %w", err)
			}
		}
		return nil
	}

	return fmt.Errorf("no supported desktop environment detected; configure PAC URL %s manually", pacURL)
}

func (l *linuxSystemProxy) Disable() error {
	switch desktopEnv() {
	case "gnome":
		return exec.Command("gsettings", "set", "org.gnome.system.proxy", "mode", "none").Run()
	case "kde":
		return exec.Command("kwriteconfig5", "--file", "kioslaverc", "--group", "Proxy Settings", "--key", "ProxyType", "0").Run()
	}
	return nil
}

func (l *linuxSystemProxy) Status() (*ProxyStatus, error) {
	status := &ProxyStatus{}

	switch desktopEnv() {
	case "gnome":
		if out, err := exec.Command("gsettings", "get", "org.gnome.system.proxy", "mode").Output(); err == nil {
			mode := strings.Trim(strings.TrimSpace(string(out)), "'")
			if mode == "manual" {
				// HTTP
				if h, err := exec.Command("gsettings", "get", "org.gnome.system.proxy.http", "host").Output(); err == nil {
					host := strings.Trim(strings.TrimSpace(string(h)), "'")
					if host != "" {
						status.HTTPEnabled = true
						status.HTTPHost = host
					}
				}
				if p, err := exec.Command("gsettings", "get", "org.gnome.system.proxy.http", "port").Output(); err == nil {
					status.HTTPPort, _ = strconv.Atoi(strings.TrimSpace(string(p)))
				}
				// SOCKS
				if h, err := exec.Command("gsettings", "get", "org.gnome.system.proxy.socks", "host").Output(); err == nil {
					host := strings.Trim(strings.TrimSpace(string(h)), "'")
					if host != "" {
						status.SOCKSEnabled = true
						status.SOCKSHost = host
					}
				}
				if p, err := exec.Command("gsettings", "get", "org.gnome.system.proxy.socks", "port").Output(); err == nil {
					status.SOCKSPort, _ = strconv.Atoi(strings.TrimSpace(string(p)))
				}
			}
			if mode == "auto" {
				status.PACEnabled = true
				if u, err := exec.Command("gsettings", "get", "org.gnome.system.proxy", "autoconfig-url").Output(); err == nil {
					status.PACURL = strings.Trim(strings.TrimSpace(string(u)), "'")
				}
			}
		}
	}
	return status, nil
}
