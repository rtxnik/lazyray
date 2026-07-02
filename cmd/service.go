package cmd

import (
	"fmt"
	"os"

	"github.com/rtxnik/lazyray/internal/clihint"
	"github.com/rtxnik/lazyray/internal/platform"
	"github.com/spf13/cobra"
)

var serviceCmd = &cobra.Command{
	Use:   "service",
	Short: "Manage xray system service (autostart)",
	Long: "Manage the user-scoped system service that starts xray-core automatically on login. " +
		"Use 'lzr service install' to enable autostart, 'lzr service uninstall' to remove it, and " +
		"'lzr service status' to check it.",
}

var serviceInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Install xray as a system service",
	Long: "Install xray-core as a user-scoped system service so it starts automatically on login. " +
		"The service runs under your user account and does not require root; if installation fails, " +
		"run 'lzr doctor' and see TROUBLESHOOTING.md.",
	Example: "  lzr service install",
	RunE: func(cmd *cobra.Command, args []string) error {
		p := platform.Current()
		exe, err := os.Executable()
		if err != nil {
			return serviceLocateError(err)
		}

		if err := p.ServiceInstall(exe); err != nil {
			return serviceInstallError(err)
		}

		fmt.Println("Service installed and started")
		return nil
	},
}

var serviceUninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Remove xray system service",
	Long: "Remove the user-scoped system service that starts xray-core on login. " +
		"This stops xray-core from autostarting but does not remove any proxy profiles or settings.",
	Example: "  lzr service uninstall",
	RunE: func(cmd *cobra.Command, args []string) error {
		p := platform.Current()
		if err := p.ServiceUninstall(); err != nil {
			return fmt.Errorf("uninstalling service: %w", err)
		}

		fmt.Println("Service removed")
		return nil
	},
}

var serviceStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show service status",
	Long: "Report whether the user-scoped xray-core autostart service is installed and whether it " +
		"is currently running. For full diagnostics of the running proxy use 'lzr doctor'.",
	Example: "  lzr service status",
	RunE: func(cmd *cobra.Command, args []string) error {
		p := platform.Current()
		installed, running, err := p.ServiceStatus()
		if err != nil {
			return fmt.Errorf("checking service status: %w", err)
		}

		if !installed {
			fmt.Println("Service: not installed")
		} else if running {
			fmt.Println("Service: installed and running")
		} else {
			fmt.Println("Service: installed but not running")
		}
		return nil
	},
}

// serviceLocateError reports that lazyray could not locate its own binary while
// installing the user-scoped service, with the service-install hint.
func serviceLocateError(err error) error {
	return clihint.Errorf(
		"services are user-scoped — see 'lzr doctor' and TROUBLESHOOTING.md",
		"locating lzr binary: %w", err)
}

// serviceInstallError reports that installing the user-scoped system service
// failed (commonly a permission or service-locator problem), with the
// service-install hint.
func serviceInstallError(err error) error {
	return clihint.Errorf(
		"services are user-scoped — see 'lzr doctor' and TROUBLESHOOTING.md",
		"installing service: %w", err)
}

func init() {
	serviceCmd.AddCommand(serviceInstallCmd, serviceUninstallCmd, serviceStatusCmd)
	rootCmd.AddCommand(serviceCmd)
}
