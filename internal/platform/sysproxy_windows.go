package platform

import (
	"fmt"
	"os/exec"
	"strings"
)

type windowsSystemProxy struct{}

func currentSystemProxy() SystemProxy {
	return &windowsSystemProxy{}
}

const regPath = `HKCU\Software\Microsoft\Windows\Internet Settings`

func (w *windowsSystemProxy) EnableHTTPProxy(host string, port int) error {
	proxyAddr := fmt.Sprintf("%s:%d", host, port)

	if err := exec.Command("reg", "add", regPath, "/v", "ProxyEnable", "/t", "REG_DWORD", "/d", "1", "/f").Run(); err != nil {
		return fmt.Errorf("enabling proxy: %w", err)
	}
	if err := exec.Command("reg", "add", regPath, "/v", "ProxyServer", "/t", "REG_SZ", "/d", proxyAddr, "/f").Run(); err != nil {
		return fmt.Errorf("setting proxy server: %w", err)
	}
	return nil
}

func (w *windowsSystemProxy) EnableSOCKSProxy(host string, port int) error {
	proxyAddr := fmt.Sprintf("socks=%s:%d", host, port)

	if err := exec.Command("reg", "add", regPath, "/v", "ProxyEnable", "/t", "REG_DWORD", "/d", "1", "/f").Run(); err != nil {
		return fmt.Errorf("enabling proxy: %w", err)
	}
	if err := exec.Command("reg", "add", regPath, "/v", "ProxyServer", "/t", "REG_SZ", "/d", proxyAddr, "/f").Run(); err != nil {
		return fmt.Errorf("setting SOCKS proxy: %w", err)
	}
	return nil
}

func (w *windowsSystemProxy) EnablePACProxy(pacURL string) error {
	if err := exec.Command("reg", "add", regPath, "/v", "AutoConfigURL", "/t", "REG_SZ", "/d", pacURL, "/f").Run(); err != nil {
		return fmt.Errorf("setting PAC URL: %w", err)
	}
	// Disable manual proxy when using PAC
	_ = exec.Command("reg", "add", regPath, "/v", "ProxyEnable", "/t", "REG_DWORD", "/d", "0", "/f").Run()
	return nil
}

func (w *windowsSystemProxy) Disable() error {
	_ = exec.Command("reg", "add", regPath, "/v", "ProxyEnable", "/t", "REG_DWORD", "/d", "0", "/f").Run()
	_ = exec.Command("reg", "delete", regPath, "/v", "ProxyServer", "/f").Run()
	_ = exec.Command("reg", "delete", regPath, "/v", "AutoConfigURL", "/f").Run()
	return nil
}

func (w *windowsSystemProxy) Status() (*ProxyStatus, error) {
	status := &ProxyStatus{}

	// Check ProxyEnable
	if out, err := exec.Command("reg", "query", regPath, "/v", "ProxyEnable").Output(); err == nil {
		if strings.Contains(string(out), "0x1") {
			// Check ProxyServer
			if out2, err := exec.Command("reg", "query", regPath, "/v", "ProxyServer").Output(); err == nil {
				for _, line := range strings.Split(string(out2), "\n") {
					line = strings.TrimSpace(line)
					if strings.Contains(line, "ProxyServer") && strings.Contains(line, "REG_SZ") {
						parts := strings.Fields(line)
						if len(parts) >= 3 {
							addr := parts[len(parts)-1]
							if strings.HasPrefix(addr, "socks=") {
								addr = strings.TrimPrefix(addr, "socks=")
								status.SOCKSEnabled = true
								parseHostPort(addr, &status.SOCKSHost, &status.SOCKSPort)
							} else {
								status.HTTPEnabled = true
								parseHostPort(addr, &status.HTTPHost, &status.HTTPPort)
							}
						}
					}
				}
			}
		}
	}

	// Check AutoConfigURL
	if out, err := exec.Command("reg", "query", regPath, "/v", "AutoConfigURL").Output(); err == nil {
		for _, line := range strings.Split(string(out), "\n") {
			line = strings.TrimSpace(line)
			if strings.Contains(line, "AutoConfigURL") && strings.Contains(line, "REG_SZ") {
				parts := strings.Fields(line)
				if len(parts) >= 3 {
					status.PACEnabled = true
					status.PACURL = parts[len(parts)-1]
				}
			}
		}
	}

	return status, nil
}
