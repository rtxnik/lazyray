// cmd/stop.go
package cmd

import (
	"fmt"
	"time"

	"github.com/rtxnik/lazyray/internal/lifecycle"
	"github.com/rtxnik/lazyray/internal/platform"
	"github.com/spf13/cobra"
)

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the xray proxy and revert system routing",
	Long: `Stop the running xray-core supervisor and revert any system proxy changes it made.

This asks the supervisor to shut down gracefully so it can tear down the SSH tunnel and restore your OS-level system proxy settings; if it does not respond it is hard-killed and the session is self-healed. Running 'lzr stop' when lazyray is not running is a no-op.`,
	Example: `  # Stop the supervisor and restore system routing
  lzr stop`,
	RunE: func(cmd *cobra.Command, args []string) error {
		sp := platform.CurrentSystemProxy()

		if !lifecycle.SupervisorAlive() {
			// Nothing live; self-heal any dangling proxy/state and report.
			_ = lifecycle.Reconcile(sp)
			fmt.Println("lazyray is not running")
			return nil // idempotent: exit 0
		}

		st, err := lifecycle.ReadState()
		if err != nil || st == nil || st.SupervisorPID <= 0 {
			// Lock held but no usable state: best-effort reconcile.
			_ = lifecycle.Reconcile(sp)
			fmt.Println("lazyray stopped")
			return nil
		}

		// Ask the supervisor to shut down gracefully; it runs Teardown and
		// releases the lock on clean exit.
		_ = lifecycle.SignalSupervisor(st.SupervisorPID)

		if waitSupervisorDown(7 * time.Second) {
			fmt.Println("lazyray stopped")
			return nil
		}

		// Supervisor ignored SIGTERM: hard-kill it, then self-heal the session.
		_ = forceKillSupervisor(st.SupervisorPID)
		_ = waitSupervisorDown(5 * time.Second) // let the kernel release the supervisor's flock
		_ = lifecycle.Reconcile(sp)
		fmt.Println("lazyray stopped (forced)")
		return nil
	},
}

// waitSupervisorDown polls until the lock is free (supervisor exited).
func waitSupervisorDown(timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if !lifecycle.SupervisorAlive() {
			return true
		}
		time.Sleep(50 * time.Millisecond)
	}
	return !lifecycle.SupervisorAlive()
}

func init() {
	rootCmd.AddCommand(stopCmd)
}
