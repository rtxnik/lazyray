package cmd

import (
	"fmt"
	"time"

	"github.com/rtxnik/lazyray/internal/config"
	"github.com/rtxnik/lazyray/internal/core"
	"github.com/spf13/cobra"
)

var (
	speedtestDuration int
	speedtestURL      string
)

var speedtestCmd = &cobra.Command{
	Use:   "speedtest",
	Short: "Run a download speed test through the proxy",
	Long: `Measure download throughput through the active system proxy. Downloads data from a
test server for the given duration and reports the observed speed. Requires a running
connection — start one with 'lzr start' first. Use --duration to lengthen the sample
and --url to point at your own test file.`,
	Example: `  # Run the default 10-second speed test
  lzr speedtest

  # Sample for 15 seconds
  lzr speedtest --duration 15

  # Use a custom download URL
  lzr speedtest --url http://example.com/testfile`,
	RunE: func(cmd *cobra.Command, args []string) error {
		settings, err := config.LoadSettings()
		if err != nil {
			return fmt.Errorf("loading settings: %w", err)
		}

		duration := time.Duration(speedtestDuration) * time.Second
		fmt.Printf("Running speed test (%ds)...\n", speedtestDuration)

		result := core.RunSpeedTest(settings, speedtestURL, duration)
		fmt.Print(core.FormatSpeedTestResult(result))
		return nil
	},
}

func init() {
	speedtestCmd.Flags().IntVar(&speedtestDuration, "duration", 10, "Test duration in seconds")
	speedtestCmd.Flags().StringVar(&speedtestURL, "url", "", "Custom download test URL")
	rootCmd.AddCommand(speedtestCmd)
}
