package cmd

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/rtxnik/lazyray/internal/clihint"
	"github.com/rtxnik/lazyray/internal/tui"
	"github.com/spf13/cobra"
)

var version = "dev"

// SetVersion sets the application version from ldflags.
func SetVersion(v string) {
	version = v
	rootCmd.Version = v
}

// Version returns the current version string.
func Version() string {
	return version
}

var rootCmd = &cobra.Command{
	Use:   "lzr",
	Short: "Terminal UI for managing Xray-core proxy configurations",
	Long:  "lazyray — a terminal application for managing VLESS proxy configurations powered by Xray-core.",
	RunE: func(cmd *cobra.Command, args []string) error {
		m := tui.NewApp(version)
		p := tea.NewProgram(m, tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			return fmt.Errorf("running TUI: %w", err)
		}
		return nil
	},
}

func init() {
	rootCmd.Version = version
	rootCmd.SilenceErrors = true
	rootCmd.SilenceUsage = true
}

// Execute runs the root command and returns a process exit code.
func Execute() int {
	if err := rootCmd.Execute(); err != nil {
		clihint.Render(os.Stderr, err)
		return exitCodeFor(err)
	}
	return ExitOK
}

// RootCmd returns the configured root command. It exists so external tooling
// (e.g. tools/gen-docs man-page generation) can traverse the command tree
// without reaching into package internals.
func RootCmd() *cobra.Command {
	return rootCmd
}
