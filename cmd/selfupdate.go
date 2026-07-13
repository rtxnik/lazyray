package cmd

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/rtxnik/lazyray/internal/core"
	"github.com/rtxnik/lazyray/internal/release"
	"github.com/spf13/cobra"
)

var selfUpdateCmd = &cobra.Command{
	Use:     "self-update",
	Short:   "Update lazyray to the latest version",
	Long:    "Check for and install the latest version of lazyray from GitHub releases.",
	Example: "  lzr self-update",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Checking for updates...")
		release, err := core.CheckSelfUpdate()
		if err != nil {
			return fmt.Errorf("checking for updates: %w", err)
		}

		latest := strings.TrimPrefix(release.TagName, "v")
		current := strings.TrimPrefix(version, "v")

		switch selfUpdateDecision(latest, current, selfUpdateAllowDowngrade) {
		case upToDate:
			fmt.Printf("Already up to date (v%s)\n", current)
			return nil
		case refuseDowngrade:
			return fmt.Errorf("refusing downgrade v%s -> v%s (older than installed); pass --allow-downgrade to override", current, latest)
		}

		fmt.Printf("Current: v%s → Latest: v%s\n", current, latest)

		urls, err := core.FindSelfAssetURL(release)
		if err != nil {
			return fmt.Errorf("finding download: %w", err)
		}

		execPath, err := os.Executable()
		if err != nil {
			return fmt.Errorf("locating current executable: %w", err)
		}

		fmt.Printf("Downloading and verifying %s...\n", urls.AssetName)
		if err := core.ApplySelfUpdate(urls, execPath); err != nil {
			return errors.New(selfUpdateUserMessage(err))
		}

		fmt.Printf("Updated to v%s. Restart lazyray to use the new version.\n", latest)
		return nil
	},
}

// selfUpdateAction is the outcome of comparing the latest release against the
// currently installed version.
type selfUpdateAction int

const (
	upToDate selfUpdateAction = iota
	proceed
	refuseDowngrade
)

// selfUpdateDecision decides whether a self-update should proceed, no-op, or
// be refused as a downgrade. allowDowngrade overrides the anti-rollback
// refusal.
func selfUpdateDecision(latest, current string, allowDowngrade bool) selfUpdateAction {
	switch cmp := core.CompareVersions(latest, current); {
	case cmp == 0:
		return upToDate
	case cmp < 0 && !allowDowngrade:
		return refuseDowngrade
	default:
		return proceed
	}
}

var selfUpdateAllowDowngrade bool

func init() {
	selfUpdateCmd.Flags().BoolVar(&selfUpdateAllowDowngrade, "allow-downgrade", false,
		"allow installing an older version than the one currently running")
	rootCmd.AddCommand(selfUpdateCmd)
}

// selfUpdateUserMessage maps a verified-self-update error to an actionable
// message. The typed sentinels from internal/release get bespoke guidance; any
// other error is passed through verbatim.
func selfUpdateUserMessage(err error) string {
	switch {
	case err == nil:
		return ""
	case errors.Is(err, release.ErrSignatureInvalid):
		return "update aborted: the release signature did not verify against the embedded key — the download may be tampered or the release was not signed; not installing"
	case errors.Is(err, release.ErrChecksumMismatch):
		return "update aborted: the downloaded archive does not match the signed checksum manifest — the download is corrupt or tampered; not installing"
	case errors.Is(err, release.ErrAssetNotFound):
		return "update aborted: this release is missing a required asset (the archive or its signed checksums) — wait for a fully published, signed release"
	default:
		return fmt.Sprintf("applying update: %v", err)
	}
}
