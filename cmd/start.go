// cmd/start.go
package cmd

import (
	"fmt"
	"time"

	"github.com/rtxnik/lazyray/internal/app"
	"github.com/rtxnik/lazyray/internal/clihint"
	"github.com/rtxnik/lazyray/internal/config"
	"github.com/rtxnik/lazyray/internal/lifecycle"
	"github.com/rtxnik/lazyray/internal/platform"
	"github.com/spf13/cobra"
)

var (
	startProxy   bool
	startNoProxy bool
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the xray proxy supervisor in the background",
	Long: `Start the xray-core supervisor as a background process using the default proxy profile from your settings.

This is the headless counterpart to launching the TUI: it spawns the resident supervisor, waits for it to take the lock, and prints the local SOCKS5/HTTP listen addresses. Use --proxy or --no-proxy to override whether the system proxy is enabled for this session. Running 'lzr start' when lazyray is already running is a no-op.`,
	Example: `  # Start in the background using your default profile and settings
  lzr start

  # Start and force the system proxy on for this session
  lzr start --proxy

  # Start without touching system proxy settings
  lzr start --no-proxy`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Self-heal any dangling state from a previous crash before deciding.
		_ = lifecycle.Reconcile(platform.CurrentSystemProxy())

		if lifecycle.SupervisorAlive() {
			fmt.Println("lazyray is already running")
			return nil // idempotent: exit 0
		}

		settings, err := config.LoadSettings()
		if err != nil {
			return &ExitError{Code: ExitConfig, Err: fmt.Errorf("loading settings: %w", err)}
		}

		extra := []string{"--owner", string(lifecycle.OwnerDaemon)}
		if startNoProxy {
			extra = append(extra, "--no-proxy")
		} else if startProxy {
			extra = append(extra, "--proxy")
		}

		svc := app.NewService()
		if err := svc.Connect(extra); err != nil {
			return &ExitError{Code: ExitConfig, Err: fmt.Errorf("spawning supervisor: %w", err)}
		}

		// Wait for the supervisor to take the lock (confirms it came up).
		if !waitSupervisorUp(3 * time.Second) {
			return supervisorNotUpError("supervisor did not start")
		}

		st, _ := lifecycle.ReadState()
		fmt.Println("lazyray started in background")
		if st != nil {
			fmt.Printf("  SOCKS5: %s:%d\n", settings.Local.Listen, st.SocksPort)
			fmt.Printf("  HTTP:   %s:%d\n", settings.Local.Listen, st.HTTPPort)
			if st.Routing.SystemProxy {
				fmt.Println("  System proxy: enabled")
			}
		}
		return nil
	},
}

// waitSupervisorUp polls until a supervisor holds the lock or the timeout hits.
func waitSupervisorUp(timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if lifecycle.SupervisorAlive() {
			return true
		}
		time.Sleep(50 * time.Millisecond)
	}
	return lifecycle.SupervisorAlive()
}

// supervisorNotUpError builds the exit-aware, hinted error used when the
// supervisor fails to take the lock within the start/restart timeout.
func supervisorNotUpError(msg string) error {
	return &ExitError{
		Code: ExitConfig,
		Err:  clihint.Errorf("check 'lzr status' and logs, then 'lzr doctor'", "%s", msg),
	}
}

func init() {
	startCmd.Flags().BoolVar(&startProxy, "proxy", false, "enable the system proxy for this session, overriding the profile store default")
	startCmd.Flags().BoolVar(&startNoProxy, "no-proxy", false, "leave the system proxy untouched for this session, overriding the profile store default")
	rootCmd.AddCommand(startCmd)
}
