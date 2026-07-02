package cmd

import (
	"fmt"
	"time"

	"github.com/rtxnik/lazyray/internal/app"
	"github.com/rtxnik/lazyray/internal/config"
	"github.com/rtxnik/lazyray/internal/lifecycle"
	"github.com/rtxnik/lazyray/internal/platform"
	"github.com/spf13/cobra"
)

var restartCmd = &cobra.Command{
	Use:   "restart",
	Short: "Restart the xray proxy supervisor",
	Long: `Stop the running xray-core supervisor (if any) and start a fresh one using the current settings.

Use this after editing the profile store or switching the default proxy profile so the new configuration takes effect. The system proxy default for the new session comes from your settings, the same as a plain 'lzr start'.`,
	Example: `  # Apply new settings by restarting the supervisor
  lzr restart`,
	RunE: func(cmd *cobra.Command, args []string) error {
		sp := platform.CurrentSystemProxy()

		// Stop if running.
		if lifecycle.SupervisorAlive() {
			if st, _ := lifecycle.ReadState(); st != nil && st.SupervisorPID > 0 {
				_ = lifecycle.SignalSupervisor(st.SupervisorPID)
				if !waitSupervisorDown(7 * time.Second) {
					_ = forceKillSupervisor(st.SupervisorPID)
					_ = waitSupervisorDown(5 * time.Second)
				}
			}
			_ = lifecycle.Reconcile(sp)
		} else {
			_ = lifecycle.Reconcile(sp)
		}

		// Start fresh (proxy default comes from settings via __run).
		if _, err := config.LoadSettings(); err != nil {
			return &ExitError{Code: ExitConfig, Err: fmt.Errorf("loading settings: %w", err)}
		}
		svc := app.NewService()
		if err := svc.Connect([]string{"--owner", string(lifecycle.OwnerDaemon)}); err != nil {
			return &ExitError{Code: ExitConfig, Err: fmt.Errorf("spawning supervisor: %w", err)}
		}
		if !waitSupervisorUp(3 * time.Second) {
			return supervisorNotUpError("supervisor did not restart")
		}
		fmt.Println("lazyray restarted")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(restartCmd)
}
