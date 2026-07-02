package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/rtxnik/lazyray/internal/core"
	"github.com/spf13/cobra"
)

var statsJSON bool

var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show traffic consumption statistics",
	Long: `Show traffic consumption recorded by lazyray — today's usage, the current month,
and the all-time total — as a human-readable report. Pass --json to emit the raw history
for scripting. Stats accrue while a proxy profile is connected; this command reads the
stored history and does not require an active connection.`,
	Example: `  # Show the traffic report
  lzr stats

  # Emit the raw history as JSON
  lzr stats --json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		sm := core.GetStatsManager()
		history := sm.GetHistory()

		if statsJSON {
			data, err := json.MarshalIndent(history, "", "  ")
			if err != nil {
				return fmt.Errorf("marshaling JSON: %w", err)
			}
			fmt.Fprintln(os.Stdout, string(data))
			return nil
		}

		fmt.Print(core.FormatStatsReport(history))
		return nil
	},
}

func init() {
	statsCmd.Flags().BoolVar(&statsJSON, "json", false, "Output in JSON format")
	rootCmd.AddCommand(statsCmd)
}
