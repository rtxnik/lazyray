package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/rtxnik/lazyray/internal/config"
	"github.com/spf13/cobra"
)

var logsError bool
var logsLines int

var logsCmd = &cobra.Command{
	Use:   "logs",
	Short: "Show xray log output",
	Long: `Show the tail of the xray-core log files.

logs prints the most recent lines of the xray-core access log, or the error log
with --error. Use it to see what the engine recorded for the active proxy
connection when a session misbehaves; pair it with 'lzr doctor' for a full
diagnosis. If no log file exists yet, logs says so rather than failing.`,
	Example: `  # Last 50 lines of the access log
  lzr logs

  # Last 200 lines of the error log
  lzr logs --error --lines 200`,
	RunE: func(cmd *cobra.Command, args []string) error {
		logPath := config.AccessLogPath()
		if logsError {
			logPath = config.ErrorLogPath()
		}

		data, err := os.ReadFile(logPath)
		if err != nil {
			if os.IsNotExist(err) {
				fmt.Println("No log file found at", logPath)
				return nil
			}
			return fmt.Errorf("reading log: %w", err)
		}

		lines := strings.Split(strings.TrimSpace(string(data)), "\n")
		if logsLines > 0 && len(lines) > logsLines {
			lines = lines[len(lines)-logsLines:]
		}

		for _, line := range lines {
			fmt.Println(line)
		}
		return nil
	},
}

func init() {
	logsCmd.Flags().BoolVarP(&logsError, "error", "e", false, "Show the xray-core error log instead of the access log")
	logsCmd.Flags().IntVarP(&logsLines, "lines", "n", 50, "Number of trailing log lines to show (0 shows all)")
	rootCmd.AddCommand(logsCmd)
}
