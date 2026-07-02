package cmd

import (
	"fmt"
	"strings"

	"github.com/rtxnik/lazyray/internal/config"
	"github.com/rtxnik/lazyray/internal/core"
	"github.com/spf13/cobra"
)

var tunnelManager = core.NewTunnelManager()

var tunnelCmd = &cobra.Command{
	Use:     "tunnel [name]",
	Aliases: []string{"ssh-tunnel"},
	Short:   "Manage SSH tunnels to server panels",
	Long: `Open or inspect an SSH tunnel to a profile's management panel.

This is an SSH tunnel to the server's admin panel — separate from the proxy itself: it does not route your traffic through xray-core and does not change the active proxy profile. With no argument it lists SSH-capable profiles and their tunnel state; with a profile name it opens a persistent tunnel to that profile's panel. Tear tunnels down with 'lzr tunnel close'.`,
	Example: "  lzr tunnel\n  lzr tunnel ru\n  lzr tunnel close",
	Args:    cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return tunnelStatus()
		}

		servers, err := config.LoadServers()
		if err != nil {
			return fmt.Errorf("loading servers: %w", err)
		}

		return tunnelConnectByName(servers, strings.ToLower(args[0]))
	},
}

var tunnelCloseCmd = &cobra.Command{
	Use:     "close",
	Short:   "Close all SSH tunnels",
	Long:    "Close every open SSH tunnel that 'lzr tunnel' started. SSH tunnels persist after the command that opened them exits, so use this to tear them all down.",
	Example: "  lzr tunnel close",
	RunE: func(cmd *cobra.Command, args []string) error {
		tunnelManager.DisconnectAll()
		fmt.Println("All tunnels closed")
		return nil
	},
}

func tunnelConnectByName(servers *config.ServersConfig, target string) error {
	for i := range servers.Profiles {
		p := &servers.Profiles[i]
		if strings.EqualFold(p.Name, target) || matchesShortName(p.Name, target) {
			if err := tunnelManager.Connect(p); err != nil {
				return err
			}
			statuses := tunnelManager.Status(servers.Profiles)
			for _, s := range statuses {
				if s.Name == p.Name && s.Connected {
					fmt.Printf("Connected to %s (PID %d)\n", s.Name, s.PID)
					fmt.Printf("  Panel: %s\n", s.PanelURL)
					fmt.Println("  Tunnel will persist after this command exits")
					fmt.Println("  Close with: lzr tunnel close")
					return nil
				}
			}
			return nil
		}
	}

	return errProfileNotFound(target)
}

func tunnelStatus() error {
	servers, err := config.LoadServers()
	if err != nil {
		return fmt.Errorf("loading servers: %w", err)
	}

	statuses := tunnelManager.Status(servers.Profiles)
	if len(statuses) == 0 {
		fmt.Println("No SSH-capable profiles configured")
		return nil
	}

	for _, s := range statuses {
		state := "disconnected"
		if s.Connected {
			state = fmt.Sprintf("connected (PID %d) → %s", s.PID, s.PanelURL)
		}
		fmt.Printf("  %s: %s\n", s.Name, state)
	}
	return nil
}

func matchesShortName(profileName, target string) bool {
	// Allow matching "al" to "Alpha→Beta Cascade" etc.
	lower := strings.ToLower(profileName)
	return strings.HasPrefix(lower, target)
}

func init() {
	tunnelCmd.AddCommand(tunnelCloseCmd)
	rootCmd.AddCommand(tunnelCmd)
}
