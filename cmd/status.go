package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/rtxnik/lazyray/internal/status"
	"github.com/spf13/cobra"
)

var statusJSON bool

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show xray proxy status",
	Long: `Show whether the xray-core supervisor is running and the health of its local listeners.

Reports the supervisor state and PID, uptime, the active proxy profile, the xray-core version, and the local SOCKS5/HTTP listen addresses with an up/down probe of each. Use --json for a machine-readable snapshot; for a full diagnostic sweep use 'lzr doctor' instead.`,
	Example: `  # Human-readable status
  lzr status

  # Machine-readable snapshot
  lzr status --json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		snap, err := status.Get()
		if err != nil {
			return fmt.Errorf("collecting status: %w", err)
		}

		if statusJSON {
			data, err := json.MarshalIndent(snap, "", "  ")
			if err != nil {
				return fmt.Errorf("marshaling JSON: %w", err)
			}
			fmt.Fprintln(os.Stdout, string(data))
			return nil
		}

		state := "Stopped"
		if snap.Running {
			state = fmt.Sprintf("Running (PID %d)", snap.PID)
		}

		fmt.Printf("Status:  %s\n", state)
		if snap.Running {
			fmt.Printf("Uptime:  %s\n", snap.Uptime)
		}
		fmt.Printf("Profile: %s\n", snap.Profile)
		fmt.Printf("Xray:    %s\n", snap.XrayVersion)

		socksState := "DOWN"
		if snap.SocksOK {
			socksState = "OK"
		}
		httpState := "DOWN"
		if snap.HTTPOK {
			httpState = "OK"
		}
		fmt.Printf("SOCKS5:  %s [%s]\n", snap.SocksAddr, socksState)
		fmt.Printf("HTTP:    %s [%s]\n", snap.HTTPAddr, httpState)

		return nil
	},
}

func init() {
	statusCmd.Flags().BoolVar(&statusJSON, "json", false, "output a machine-readable JSON status snapshot")
	rootCmd.AddCommand(statusCmd)
}
