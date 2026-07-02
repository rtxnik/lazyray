package cmd

import (
	"fmt"

	"github.com/rtxnik/lazyray/internal/config"
	"github.com/rtxnik/lazyray/internal/platform"
	"github.com/spf13/cobra"
)

var proxyCmd = &cobra.Command{
	Use:   "proxy",
	Short: "System proxy configuration",
	Long:  "Manage the system proxy — the OS-level proxy settings that route applications through lazyray's local ports. Use the subcommands to turn the system proxy on or off and to inspect its current state.",
}

var proxyOnCmd = &cobra.Command{
	Use:     "on",
	Short:   "Enable system proxy",
	Long:    "Enable the system proxy: point the OS HTTP and SOCKS proxy settings at lazyray's local listener so traffic is routed through the active proxy profile. Run this after starting lazyray to send system-wide traffic through the proxy server.",
	Example: "  lzr proxy on",
	RunE: func(cmd *cobra.Command, args []string) error {
		settings, err := config.LoadSettings()
		if err != nil {
			return fmt.Errorf("loading settings: %w", err)
		}

		sp := platform.CurrentSystemProxy()

		if err := sp.EnableHTTPProxy(settings.Local.Listen, settings.Local.HTTPPort); err != nil {
			return fmt.Errorf("setting HTTP proxy: %w", err)
		}
		if err := sp.EnableSOCKSProxy(settings.Local.Listen, settings.Local.SocksPort); err != nil {
			return fmt.Errorf("setting SOCKS proxy: %w", err)
		}

		fmt.Printf("System proxy enabled: HTTP %s:%d, SOCKS5 %s:%d\n",
			settings.Local.Listen, settings.Local.HTTPPort,
			settings.Local.Listen, settings.Local.SocksPort)
		return nil
	},
}

var proxyOffCmd = &cobra.Command{
	Use:     "off",
	Short:   "Disable system proxy",
	Long:    "Disable the system proxy: remove the OS-level HTTP and SOCKS proxy settings that lazyray applied, restoring direct connections. Run this when you want applications to stop routing through lazyray.",
	Example: "  lzr proxy off",
	RunE: func(cmd *cobra.Command, args []string) error {
		sp := platform.CurrentSystemProxy()
		if err := sp.Disable(); err != nil {
			return fmt.Errorf("disabling system proxy: %w", err)
		}
		fmt.Println("System proxy disabled")
		return nil
	},
}

var proxyStatusCmd = &cobra.Command{
	Use:     "status",
	Short:   "Show system proxy status",
	Long:    "Show the current system proxy state: whether the OS HTTP, SOCKS, and PAC proxy settings are enabled and which host and ports they point at. Use it to confirm whether 'lzr proxy on' took effect.",
	Example: "  lzr proxy status",
	RunE: func(cmd *cobra.Command, args []string) error {
		sp := platform.CurrentSystemProxy()
		status, err := sp.Status()
		if err != nil {
			return fmt.Errorf("getting proxy status: %w", err)
		}

		if status.HTTPEnabled {
			fmt.Printf("HTTP proxy:  enabled (%s:%d)\n", status.HTTPHost, status.HTTPPort)
		} else {
			fmt.Println("HTTP proxy:  disabled")
		}
		if status.SOCKSEnabled {
			fmt.Printf("SOCKS proxy: enabled (%s:%d)\n", status.SOCKSHost, status.SOCKSPort)
		} else {
			fmt.Println("SOCKS proxy: disabled")
		}
		if status.PACEnabled {
			fmt.Printf("PAC URL:     enabled (%s)\n", status.PACURL)
		} else {
			fmt.Println("PAC URL:     disabled")
		}
		return nil
	},
}

func init() {
	proxyCmd.AddCommand(proxyOnCmd)
	proxyCmd.AddCommand(proxyOffCmd)
	proxyCmd.AddCommand(proxyStatusCmd)
	rootCmd.AddCommand(proxyCmd)
}
