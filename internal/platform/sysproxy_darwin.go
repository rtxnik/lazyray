package platform

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

type darwinSystemProxy struct{}

func currentSystemProxy() SystemProxy {
	return &darwinSystemProxy{}
}

// networkServices returns the list of active network service names.
func (d *darwinSystemProxy) networkServices() ([]string, error) {
	out, err := exec.Command("networksetup", "-listallnetworkservices").Output()
	if err != nil {
		return nil, fmt.Errorf("listing network services: %w", err)
	}
	var services []string
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		line = strings.TrimSpace(line)
		// Skip the header line ("An asterisk (*) denotes...")
		if line == "" || strings.HasPrefix(line, "An asterisk") {
			continue
		}
		// Skip disabled services (prefixed with *)
		line = strings.TrimPrefix(line, "* ")
		services = append(services, line)
	}
	return services, nil
}

func (d *darwinSystemProxy) EnableHTTPProxy(host string, port int) error {
	services, err := d.networkServices()
	if err != nil {
		return err
	}
	portStr := strconv.Itoa(port)
	for _, svc := range services {
		if err := exec.Command("networksetup", "-setwebproxy", svc, host, portStr).Run(); err != nil {
			return fmt.Errorf("setting web proxy on %s: %w", svc, err)
		}
		if err := exec.Command("networksetup", "-setsecurewebproxy", svc, host, portStr).Run(); err != nil {
			return fmt.Errorf("setting secure web proxy on %s: %w", svc, err)
		}
	}
	return nil
}

func (d *darwinSystemProxy) EnableSOCKSProxy(host string, port int) error {
	services, err := d.networkServices()
	if err != nil {
		return err
	}
	portStr := strconv.Itoa(port)
	for _, svc := range services {
		if err := exec.Command("networksetup", "-setsocksfirewallproxy", svc, host, portStr).Run(); err != nil {
			return fmt.Errorf("setting SOCKS proxy on %s: %w", svc, err)
		}
	}
	return nil
}

func (d *darwinSystemProxy) EnablePACProxy(pacURL string) error {
	services, err := d.networkServices()
	if err != nil {
		return err
	}
	for _, svc := range services {
		if err := exec.Command("networksetup", "-setautoproxyurl", svc, pacURL).Run(); err != nil {
			return fmt.Errorf("setting PAC URL on %s: %w", svc, err)
		}
	}
	return nil
}

func (d *darwinSystemProxy) Disable() error {
	services, err := d.networkServices()
	if err != nil {
		return err
	}
	for _, svc := range services {
		_ = exec.Command("networksetup", "-setwebproxystate", svc, "off").Run()
		_ = exec.Command("networksetup", "-setsecurewebproxystate", svc, "off").Run()
		_ = exec.Command("networksetup", "-setsocksfirewallproxystate", svc, "off").Run()
		_ = exec.Command("networksetup", "-setautoproxystate", svc, "off").Run()
	}
	return nil
}

func (d *darwinSystemProxy) Status() (*ProxyStatus, error) {
	services, err := d.networkServices()
	if err != nil {
		return nil, err
	}
	if len(services) == 0 {
		return &ProxyStatus{}, nil
	}

	status := &ProxyStatus{}
	svc := services[0]

	// Check HTTP proxy
	if out, err := exec.Command("networksetup", "-getwebproxy", svc).Output(); err == nil {
		parseNetworksetupProxy(string(out), &status.HTTPEnabled, &status.HTTPHost, &status.HTTPPort)
	}

	// Check SOCKS proxy
	if out, err := exec.Command("networksetup", "-getsocksfirewallproxy", svc).Output(); err == nil {
		parseNetworksetupProxy(string(out), &status.SOCKSEnabled, &status.SOCKSHost, &status.SOCKSPort)
	}

	// Check PAC URL
	if out, err := exec.Command("networksetup", "-getautoproxyurl", svc).Output(); err == nil {
		for _, line := range strings.Split(string(out), "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "URL: ") {
				url := strings.TrimPrefix(line, "URL: ")
				if url != "(null)" && url != "" {
					status.PACURL = url
				}
			}
			if strings.HasPrefix(line, "Enabled: ") {
				status.PACEnabled = strings.TrimPrefix(line, "Enabled: ") == "Yes"
			}
		}
	}

	return status, nil
}

// parseNetworksetupProxy parses macOS networksetup proxy output.
func parseNetworksetupProxy(output string, enabled *bool, host *string, port *int) {
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Enabled: ") {
			*enabled = strings.TrimPrefix(line, "Enabled: ") == "Yes"
		}
		if strings.HasPrefix(line, "Server: ") {
			*host = strings.TrimPrefix(line, "Server: ")
		}
		if strings.HasPrefix(line, "Port: ") {
			*port, _ = strconv.Atoi(strings.TrimPrefix(line, "Port: "))
		}
	}
}
