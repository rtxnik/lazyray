package platform

import "runtime"

// Platform defines platform-specific operations.
type Platform interface {
	// ServiceInstall installs lazyray's supervisor as a user service.
	ServiceInstall(execPath string) error
	// ServiceUninstall removes the xray system service.
	ServiceUninstall() error
	// ServiceStatus returns whether the service is installed and running.
	ServiceStatus() (installed bool, running bool, err error)
	// Notify sends a system notification.
	Notify(title, message string) error
	// ClearQuarantine removes quarantine attributes from a binary.
	ClearQuarantine(path string) error
	// OpenURL opens a URL in the default browser.
	OpenURL(url string) error
}

// Current returns the platform implementation for the current OS.
func Current() Platform {
	switch runtime.GOOS {
	case "darwin":
		return &darwinPlatform{}
	case "linux":
		return &linuxPlatform{}
	case "windows":
		return &windowsPlatform{}
	default:
		return &linuxPlatform{}
	}
}
