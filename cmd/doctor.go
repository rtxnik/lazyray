// cmd/doctor.go
package cmd

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/rtxnik/lazyray/internal/clihint"
	"github.com/rtxnik/lazyray/internal/doctor"
	"github.com/spf13/cobra"
)

// errDoctorChecksFailed reports failing diagnostics, preserving the generic exit
// code while attaching the family's canonical diagnostics hint.
func errDoctorChecksFailed(n int) error {
	return &ExitError{
		Code: ExitGeneric,
		Err:  clihint.Errorf("see the report above for details", "%d check(s) failed", n),
	}
}

// errDoctorStrictWarnings reports warnings promoted to failures under --strict,
// preserving the generic exit code.
func errDoctorStrictWarnings(n int) error {
	return &ExitError{
		Code: ExitGeneric,
		Err:  clihint.Errorf("see the report above for details", "%d warning(s) with --strict", n),
	}
}

var (
	doctorJSON   bool
	doctorStrict bool
)

// doctorRunAll is the seam through which the command obtains a Report. It
// defaults to the real doctor pipeline (DefaultEnv -> RunAll) and is swapped
// for a fake in tests so exit-code/render logic can be exercised without
// probing the host.
var doctorRunAll = func(ctx context.Context, env *doctor.Env) doctor.Report {
	return doctor.RunAll(ctx, env)
}

type doctorJSONCheck struct {
	Group    string `json:"group"`
	Name     string `json:"name"`
	Severity string `json:"severity"`
	Detail   string `json:"detail"`
	Hint     string `json:"hint"`
}

type doctorJSONSummary struct {
	OK   int `json:"ok"`
	Info int `json:"info"`
	Warn int `json:"warn"`
	Fail int `json:"fail"`
}

type doctorJSONOutput struct {
	Checks  []doctorJSONCheck `json:"checks"`
	Summary doctorJSONSummary `json:"summary"`
}

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Diagnose the lazyray installation, config, and connectivity",
	Long: `Diagnose the lazyray installation, profile store, and connectivity.

doctor runs grouped diagnostics over your environment — the xray-core binary and
its geoip/geosite data, the profile store and default proxy profile, the
supervisor and system proxy, and basic connectivity — and prints each finding
with a hint. It is the home for diagnostics; most error hints from other
commands point here. A non-zero exit means problems were found, which is the
normal outcome of a failing gate rather than a usage error.`,
	Example: `  # Run all diagnostics
  lzr doctor

  # Fail (non-zero exit) on warnings too, for use in CI
  lzr doctor --strict

  # Machine-readable report for scripts
  lzr doctor --json`,
	// A non-zero exit means "problems found" — the normal outcome of a gate, not
	// a usage error. Silence cobra's usage dump and its duplicate error print so
	// the report (and --json output) stays clean; Execute() prints the error once.
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		report := doctorRunAll(context.Background(), doctor.DefaultEnv())

		if doctorJSON {
			if err := renderDoctorJSON(cmd, report); err != nil {
				return err
			}
		} else {
			renderDoctorHuman(cmd, report)
		}

		// Exit-code derivation:
		//   Fail > 0                              -> ExitGeneric
		//   --strict and Warn > 0                 -> ExitGeneric
		//   otherwise (Warn ignored unless strict) -> nil (exit 0)
		if report.Summary.Fail > 0 {
			return errDoctorChecksFailed(report.Summary.Fail)
		}
		if doctorStrict && report.Summary.Warn > 0 {
			return errDoctorStrictWarnings(report.Summary.Warn)
		}
		return nil
	},
}

func renderDoctorJSON(cmd *cobra.Command, report doctor.Report) error {
	out := doctorJSONOutput{
		Checks:  make([]doctorJSONCheck, 0, len(report.Checks)),
		Summary: doctorJSONSummary(report.Summary),
	}
	for _, c := range report.Checks {
		out.Checks = append(out.Checks, doctorJSONCheck{
			Group:    c.Group,
			Name:     c.Name,
			Severity: c.Severity.String(),
			Detail:   c.Detail,
			Hint:     c.Hint,
		})
	}
	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling doctor report: %w", err)
	}
	fmt.Fprintln(cmd.OutOrStdout(), string(data))
	return nil
}

func renderDoctorHuman(cmd *cobra.Command, report doctor.Report) {
	w := cmd.OutOrStdout()
	lastGroup := ""
	for _, c := range report.Checks {
		if c.Group != lastGroup {
			if lastGroup != "" {
				fmt.Fprintln(w)
			}
			fmt.Fprintf(w, "%s\n", c.Group)
			lastGroup = c.Group
		}
		fmt.Fprintf(w, "  [%s] %-14s %s\n", doctorIcon(c.Severity), c.Name, c.Detail)
		if c.Hint != "" {
			fmt.Fprintf(w, "         hint: %s\n", c.Hint)
		}
	}
	fmt.Fprintf(w, "\nSummary: %d OK, %d INFO, %d WARN, %d FAIL\n",
		report.Summary.OK, report.Summary.Info, report.Summary.Warn, report.Summary.Fail)
}

// doctorIcon renders a fixed-width bracket label matching the existing
// cmd/health.go bracket style; doctor uses four severity labels, each padded to
// a 4-rune inner width: " OK ", "INFO", "WARN", "FAIL".
func doctorIcon(s doctor.Severity) string {
	switch s {
	case doctor.SeverityOK:
		return " OK "
	case doctor.SeverityInfo:
		return "INFO"
	case doctor.SeverityWarn:
		return "WARN"
	case doctor.SeverityFail:
		return "FAIL"
	default:
		return "????"
	}
}

func init() {
	doctorCmd.Flags().BoolVar(&doctorJSON, "json", false, "Print the diagnostics report as JSON instead of human-readable text")
	doctorCmd.Flags().BoolVar(&doctorStrict, "strict", false, "Treat warnings as failures, so any warning yields a non-zero exit")
	rootCmd.AddCommand(doctorCmd)
}
