package cmd

import (
	"fmt"
	"sort"
	"time"

	"github.com/rtxnik/lazyray/internal/clihint"
	"github.com/rtxnik/lazyray/internal/config"
	"github.com/rtxnik/lazyray/internal/core"
	"github.com/rtxnik/lazyray/internal/lifecycle"
	"github.com/spf13/cobra"
)

// Actionable error constructors for the testing family (see internal/clihint).
func errTestProfileNotFound(name string) error {
	return clihint.Errorf("list profiles with 'lzr config list'", "profile %q not found", name)
}

func errTestNoProfiles() error {
	return clihint.Errorf("import a profile with 'lzr import <url>'", "no profiles configured")
}

func errTestConnFailed(err error) error {
	return clihint.Errorf("diagnose with 'lzr doctor'", "FAIL: %w", err)
}

var testAllFlag bool

var testCmd = &cobra.Command{
	Use:   "test [profile-name]",
	Short: "Test connection to a profile's server",
	Long: `Test reachability of a proxy profile's server endpoint with a TCP connection probe.
Uses the default profile when no name is given. Pass --all to probe every proxy profile
and rank them by latency (failed profiles sort last). This is a fast liveness check; for a
full connectivity probe through the system proxy use 'lzr health', and for environment
diagnostics use 'lzr doctor'.`,
	Example: `  # Test the default proxy profile
  lzr test

  # Test a specific proxy profile by name
  lzr test work

  # Test every proxy profile and sort by latency
  lzr test --all`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		servers, err := config.LoadServers()
		if err != nil {
			return fmt.Errorf("loading profiles: %w", err)
		}

		if testAllFlag {
			return testAllProfiles(servers)
		}

		var profile *config.Profile
		if len(args) > 0 {
			for i := range servers.Profiles {
				if servers.Profiles[i].Name == args[0] {
					profile = &servers.Profiles[i]
					break
				}
			}
			if profile == nil {
				return errTestProfileNotFound(args[0])
			}
		} else {
			profile = servers.DefaultProfile()
			if profile == nil {
				return errTestNoProfiles()
			}
		}

		fmt.Printf("Testing connection to %s (%s:%d)...\n",
			core.StripControl(profile.Name), core.StripControl(profile.Server.Address), profile.Server.Port)

		settings, err := config.LoadSettings()
		if err != nil {
			return fmt.Errorf("loading settings: %w", err)
		}

		r := core.ProbeProfile(*profile, lifecycle.ProbeContextFor(profile.Name, settings))
		switch r.Status {
		case core.LivenessOK:
			fmt.Printf("OK: connected in %dms (%s)\n", r.Latency.Milliseconds(), r.Method)
		case core.LivenessSkipped:
			fmt.Printf("SKIP: %s\n", r.Reason)
		default:
			return errTestConnFailed(r.Err)
		}

		// Chain hops are bare servers, never the active profile → reachability only.
		for i, srv := range profile.Chain {
			fmt.Printf("Testing chain hop %d (%s:%d)...\n", i+1, core.StripControl(srv.Address), srv.Port)
			cr := core.ProbeProfile(config.Profile{Server: srv}, core.ProbeContext{Timeout: 3 * time.Second})
			switch cr.Status {
			case core.LivenessOK:
				fmt.Printf("OK: connected in %dms\n", cr.Latency.Milliseconds())
			case core.LivenessSkipped:
				fmt.Printf("SKIP: %s\n", cr.Reason)
			default:
				fmt.Printf("FAIL: %v\n", cr.Err)
			}
		}

		return nil
	},
}

const latencySkipped int64 = -2 // profile.Latency: probed but not cheaply checkable (datagram, not connected)

type profileLatency struct {
	name    string
	latency time.Duration
	skipped bool
	err     error
}

// rankProfiles orders results: working (asc latency) → skipped → failed.
func rankProfiles(results []profileLatency) {
	rank := func(r profileLatency) int {
		switch {
		case r.err != nil:
			return 2
		case r.skipped:
			return 1
		default:
			return 0
		}
	}
	sort.SliceStable(results, func(i, j int) bool {
		ri, rj := rank(results[i]), rank(results[j])
		if ri != rj {
			return ri < rj
		}
		if ri == 0 {
			return results[i].latency < results[j].latency
		}
		return false
	})
}

func testAllProfiles(servers *config.ServersConfig) error {
	if len(servers.Profiles) == 0 {
		return errTestNoProfiles()
	}

	settings, err := config.LoadSettings()
	if err != nil {
		return fmt.Errorf("loading settings: %w", err)
	}

	fmt.Printf("Testing %d profiles...\n\n", len(servers.Profiles))

	var results []profileLatency
	for i := range servers.Profiles {
		p := &servers.Profiles[i]
		fmt.Printf("  Testing %s (%s:%d)... ", core.StripControl(p.Name), core.StripControl(p.Server.Address), p.Server.Port)

		r := core.ProbeProfile(*p, lifecycle.ProbeContextFor(p.Name, settings))
		switch r.Status {
		case core.LivenessOK:
			fmt.Printf("OK %dms\n", r.Latency.Milliseconds())
			results = append(results, profileLatency{name: core.StripControl(p.Name), latency: r.Latency})
			p.Latency = r.Latency.Milliseconds()
		case core.LivenessSkipped:
			fmt.Printf("n/a (%s)\n", r.Reason)
			results = append(results, profileLatency{name: core.StripControl(p.Name), skipped: true})
			p.Latency = latencySkipped
		default:
			fmt.Printf("FAIL: %v\n", r.Err)
			results = append(results, profileLatency{name: core.StripControl(p.Name), err: r.Err})
			p.Latency = -1
		}
	}

	if err := config.SaveServers(servers); err != nil {
		fmt.Printf("\nWarning: could not save latency data: %v\n", err)
	}

	rankProfiles(results)

	fmt.Printf("\nSorted by latency:\n")
	for i, r := range results {
		switch {
		case r.err != nil:
			fmt.Printf("  %d. %s — FAIL\n", i+1, r.name)
		case r.skipped:
			fmt.Printf("  %d. %s — n/a\n", i+1, r.name)
		default:
			fmt.Printf("  %d. %s — %dms\n", i+1, r.name, r.latency.Milliseconds())
		}
	}

	if len(results) > 0 && results[0].err == nil && !results[0].skipped {
		fmt.Printf("\nFastest: %s (%dms)\n", results[0].name, results[0].latency.Milliseconds())
	}

	return nil
}

func init() {
	testCmd.Flags().BoolVarP(&testAllFlag, "all", "a", false, "Test all profiles and sort by latency")
	rootCmd.AddCommand(testCmd)
}
