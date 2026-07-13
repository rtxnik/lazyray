package cmd

import (
	"errors"
	"fmt"
	"runtime"

	"github.com/rtxnik/lazyray/internal/clihint"
	"github.com/rtxnik/lazyray/internal/config"
	"github.com/rtxnik/lazyray/internal/core"
	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Manage xray-core updates",
	Long: "Manage the pinned xray-core engine that lazyray runs as its proxy core. " +
		"Use 'lzr update check' to compare the installed xray-core against the pinned release, " +
		"and 'lzr update apply' to download and install it.",
}

// updateXrayVersion holds the value of the apply command's --version flag.
// Empty means "use settings.Update.XrayVersion".
var updateXrayVersion string

var (
	updateAllowDowngrade  bool
	updateAllowUnverified bool
)

// xrayUpdateGate is the outcome of comparing a target xray-core release
// against the installed version and the hard version floor.
type xrayUpdateGate int

const (
	gateOK xrayUpdateGate = iota
	gateBelowFloor
	gateDowngrade
	gateUpToDate
)

// xrayUpdateDecision gates an xray-core update: it enforces the hard minimum
// version floor (no override) and blocks downgrades from the installed
// version unless allowDowngrade is set. Equal versions are treated as
// up-to-date, not a downgrade.
func xrayUpdateDecision(target, installed string, allowDowngrade bool) xrayUpdateGate {
	if err := core.XrayUpdateAllowed(target, installed, allowDowngrade); err != nil {
		if errors.Is(err, core.ErrXrayBelowFloor) {
			return gateBelowFloor
		}
		return gateDowngrade
	}
	switch installed {
	case "not installed", "unknown", "":
		return gateOK
	}
	if core.CompareVersions(target, installed) == 0 {
		return gateUpToDate
	}
	return gateOK
}

// resolveXrayVersion returns the version tag to act on: the --version override
// if set, otherwise the pinned settings.Update.XrayVersion.
func resolveXrayVersion(override string) string {
	if override != "" {
		return override
	}
	settings, _ := config.LoadSettings()
	if settings == nil {
		settings = config.DefaultSettings()
	}
	return settings.Update.XrayVersion
}

var updateCheckCmd = &cobra.Command{
	Use:   "check",
	Short: "Check for xray-core updates",
	Long: "Compare the installed xray-core against the pinned release for this platform " +
		"and report whether an update is available. This does not download anything; " +
		"run 'lzr update apply' to install.",
	Example: "  lzr update check",
	RunE: func(cmd *cobra.Command, args []string) error {
		current := core.GetXrayVersion()
		fmt.Printf("Current: %s\n", current)
		fmt.Printf("Platform: %s/%s\n", runtime.GOOS, runtime.GOARCH)

		target := resolveXrayVersion("")
		release, err := core.CheckUpdate(target)
		if err != nil {
			return err
		}

		fmt.Printf("Pinned:  %s\n", release.TagName)

		if current == release.TagName {
			fmt.Println("Already up to date")
		} else {
			fmt.Println("Update available — run 'lzr update apply'")
		}
		return nil
	},
}

var updateApplyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Download and install pinned xray-core",
	Long: "Download and install the pinned xray-core release for this platform, replacing the " +
		"current xray-core binary. Use --version to install a specific release tag instead of the " +
		"one pinned in settings. Run this when 'lzr doctor' reports xray-core is missing or too old.",
	Example: "  lzr update apply\n  lzr update apply --version v26.3.27",
	RunE: func(cmd *cobra.Command, args []string) error {
		target := resolveXrayVersion(updateXrayVersion)

		release, err := core.CheckUpdate(target)
		if err != nil {
			return err
		}

		downloadURL, err := core.FindAssetURL(release)
		if err != nil {
			return err
		}

		settings, _ := config.LoadSettings()
		if settings == nil {
			settings = config.DefaultSettings()
		}

		switch xrayUpdateDecision(release.TagName, core.GetXrayVersion(), updateAllowDowngrade) {
		case gateUpToDate:
			fmt.Printf("Already up to date (%s)\n", release.TagName)
			return nil
		case gateBelowFloor:
			return fmt.Errorf("xray %s is below the minimum supported %s; refusing to install", release.TagName, core.MinXrayVersion)
		case gateDowngrade:
			return fmt.Errorf("refusing to install xray %s (not newer than installed %s); pass --allow-downgrade to override", release.TagName, core.GetXrayVersion())
		}

		fmt.Printf("Downloading xray %s for %s/%s...\n", release.TagName, runtime.GOOS, runtime.GOARCH)

		xray := core.NewXrayProcess()
		if err := core.ApplyUpdate(xray, release, downloadURL, settings.Update.BackupBefore, updateAllowUnverified, updateAllowDowngrade); err != nil {
			return xrayMissingError(err)
		}

		fmt.Printf("Updated to %s\n", release.TagName)
		return nil
	},
}

// xrayMissingError reports that the pinned xray-core could not be resolved or
// installed, with the hint to fetch xray-core via 'lzr update apply'.
func xrayMissingError(err error) error {
	return clihint.Errorf(
		"fetch xray-core with 'lzr update apply'",
		"applying xray-core update: %w", err)
}

func init() {
	updateApplyCmd.Flags().StringVar(&updateXrayVersion, "version", "", "xray-core release tag to install (overrides settings.update.xrayVersion)")
	updateApplyCmd.Flags().BoolVar(&updateAllowDowngrade, "allow-downgrade", false, "allow installing an older xray-core than the one installed")
	updateApplyCmd.Flags().BoolVar(&updateAllowUnverified, "allow-unverified-xray", false, "install without pin verification (checksum-only; NOT a security guarantee)")
	updateCmd.AddCommand(updateCheckCmd, updateApplyCmd)
	rootCmd.AddCommand(updateCmd)
}
