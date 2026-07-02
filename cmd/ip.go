package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/rtxnik/lazyray/internal/config"
	"github.com/rtxnik/lazyray/internal/core"
	"github.com/spf13/cobra"
)

var ipJSON bool

var ipCmd = &cobra.Command{
	Use:   "ip",
	Short: "Show proxy and direct IP addresses",
	Long: `Show your direct public IP and the IP seen through the proxy.

ip fetches your direct exit IP and the exit IP observed through the active proxy
server, so you can confirm at a glance that traffic is actually leaving via the
proxy and not the local network. Each lookup is reported independently; a failed
lookup is shown inline rather than aborting the command.`,
	Example: `  # Compare direct and proxy exit IPs
  lzr ip

  # Machine-readable output for scripts
  lzr ip --json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		settings, err := config.LoadSettings()
		if err != nil {
			return fmt.Errorf("loading settings: %w", err)
		}

		directIP, directErr := core.GetDirectIP(settings)
		proxyIP, proxyErr := core.GetExitIP(settings)

		if ipJSON {
			data := map[string]interface{}{
				"directIP": directIP,
				"proxyIP":  proxyIP,
			}
			if directErr != nil {
				data["directError"] = directErr.Error()
			}
			if proxyErr != nil {
				data["proxyError"] = proxyErr.Error()
			}
			out, _ := json.MarshalIndent(data, "", "  ")
			fmt.Println(string(out))
			return nil
		}

		if directErr != nil {
			fmt.Printf("Direct IP: error (%v)\n", directErr)
		} else {
			fmt.Printf("Direct IP: %s\n", directIP)
		}

		if proxyErr != nil {
			fmt.Printf("Proxy IP:  error (%v)\n", proxyErr)
		} else {
			fmt.Printf("Proxy IP:  %s\n", proxyIP)
		}

		return nil
	},
}

func init() {
	ipCmd.Flags().BoolVar(&ipJSON, "json", false, "Print the direct and proxy IPs as JSON instead of human-readable text")
	rootCmd.AddCommand(ipCmd)
}
