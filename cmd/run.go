// cmd/run.go
package cmd

import (
	"context"
	"fmt"

	"github.com/rtxnik/lazyray/internal/config"
	"github.com/rtxnik/lazyray/internal/lifecycle"
	"github.com/spf13/cobra"
)

var (
	runOwner   string
	runProxy   bool
	runNoProxy bool
)

// runCmd is the resident supervisor process. Hidden: users go through
// `lzr start` / the OS service, both of which exec `lzr __run`.
var runCmd = &cobra.Command{
	Use:    "__run",
	Hidden: true,
	Short:  "Internal: run the resident xray supervisor (foreground)",
	RunE: func(cmd *cobra.Command, args []string) error {
		servers, err := config.LoadServers()
		if err != nil {
			return fmt.Errorf("loading servers: %w", err)
		}
		profile := servers.DefaultProfile()
		if profile == nil {
			return fmt.Errorf("no profiles configured — use 'lzr import <vless://...>' first")
		}
		settings, err := config.LoadSettings()
		if err != nil {
			return fmt.Errorf("loading settings: %w", err)
		}

		mode := lifecycle.ProxyDefault
		if runNoProxy {
			mode = lifecycle.ProxyForceOff
		} else if runProxy {
			mode = lifecycle.ProxyForceOn
		}

		sup := &lifecycle.Supervisor{
			Owner:    lifecycle.Owner(runOwner),
			Profile:  profile,
			Settings: settings,
			Proxy:    mode,
		}
		if err := sup.Run(context.Background()); err != nil {
			// Record only genuine startup-phase failures. A post-success teardown
			// error is not a startup failure, and lock contention is the expected
			// outcome of a double-start — neither should make `lzr doctor` report
			// a spurious startup FAIL.
			if stage := lifecycle.StartupFailureStage(err); stage != "" {
				_ = lifecycle.WriteStartupError(stage, err)
			}
			return err
		}
		return nil
	},
}

func init() {
	runCmd.Flags().StringVar(&runOwner, "owner", string(lifecycle.OwnerDaemon), "supervisor owner tag")
	runCmd.Flags().BoolVar(&runProxy, "proxy", false, "force-enable system proxy")
	runCmd.Flags().BoolVar(&runNoProxy, "no-proxy", false, "force-disable system proxy")
	rootCmd.AddCommand(runCmd)
}
