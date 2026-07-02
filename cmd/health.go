package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/rtxnik/lazyray/internal/clihint"
	"github.com/rtxnik/lazyray/internal/config"
	"github.com/rtxnik/lazyray/internal/core"
	"github.com/spf13/cobra"
)

// errHealthChecksFailed reports that the connectivity probe found failing
// checks, routing the user to diagnostics.
func errHealthChecksFailed(n int) error {
	return clihint.Errorf("diagnose with 'lzr doctor'", "health check failed: %d check(s) did not pass", n)
}

var healthJSON bool

var healthCmd = &cobra.Command{
	Use:   "health",
	Short: "Run health check on proxy connection",
	Long: `Run a connectivity probe against the default proxy profile.

health runs a sequence of checks (xray-core startup, handshake, and exit-IP
reachability) through the proxy server the profile points to and reports which
passed. It is a fast connectivity probe — for full installation and config
diagnostics use 'lzr doctor' instead.`,
	Example: `  # Probe the default proxy profile
  lzr health

  # Machine-readable output for scripts
  lzr health --json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		servers, err := config.LoadServers()
		if err != nil {
			return fmt.Errorf("loading servers: %w", err)
		}

		profile := servers.DefaultProfile()
		if profile == nil {
			return errNoProfilesConfigured()
		}

		settings, err := config.LoadSettings()
		if err != nil {
			return fmt.Errorf("loading settings: %w", err)
		}

		xray := core.NewXrayProcess()
		report := core.RunHealthCheck(xray, profile, settings)

		if healthJSON {
			out, _ := json.MarshalIndent(report, "", "  ")
			fmt.Println(string(out))
			return nil
		}

		for _, c := range report.Checks {
			icon := "PASS"
			if !c.OK {
				icon = "FAIL"
			}
			fmt.Printf("  [%s] %-10s %s\n", icon, c.Name, c.Detail)
		}

		if report.AllPassed {
			fmt.Println("\nAll checks passed")
			return nil
		}

		failed := 0
		for _, c := range report.Checks {
			if !c.OK {
				failed++
			}
		}
		fmt.Println("\nSome checks failed")
		return errHealthChecksFailed(failed)
	},
}

func init() {
	healthCmd.Flags().BoolVar(&healthJSON, "json", false, "Print the health report as JSON instead of human-readable text")
	rootCmd.AddCommand(healthCmd)
}
