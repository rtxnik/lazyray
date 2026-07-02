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

		if latest == current {
			fmt.Printf("Already up to date (v%s)\n", current)
			return nil
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

func init() {
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
