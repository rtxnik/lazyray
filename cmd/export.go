package cmd

import (
	"errors"
	"fmt"
	"os"

	qrterminal "github.com/mdp/qrterminal/v3"
	"github.com/rtxnik/lazyray/internal/config"
	"github.com/rtxnik/lazyray/internal/core"
	"github.com/spf13/cobra"
)

var (
	exportAll            bool
	exportQR             bool
	exportEncrypt        bool
	exportPassphraseFile string
)

var exportCmd = &cobra.Command{
	Use:   "export [name]",
	Short: "Export profile as proxy URL",
	Long:  "Export a proxy profile as its protocol URL (VLESS, VMess, or Trojan). Without arguments, exports the default profile.",
	Example: `  lzr export
  lzr export home
  lzr export --all
  lzr export --qr
  lzr export --encrypt`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// The passphrase is never taken from argv; a leftover positional with
		// --encrypt is a migration error, checked before anything else.
		if exportEncrypt && len(args) > 0 {
			return fmt.Errorf("--encrypt no longer takes a value; supply the passphrase with --passphrase-file, the %s env var, or the interactive prompt", passphraseEnvVar)
		}

		servers, err := config.LoadServers()
		if err != nil {
			return fmt.Errorf("loading servers: %w", err)
		}

		if len(servers.Profiles) == 0 {
			return errNoProfilesConfigured()
		}

		// Encrypted export of all profiles.
		if exportEncrypt {
			pass, err := resolvePassphrase(exportPassphraseFile, true)
			if errors.Is(err, errNoPassphraseSource) {
				return fmt.Errorf("no passphrase source: provide --passphrase-file, set %s, or run interactively", passphraseEnvVar)
			}
			if err != nil {
				return err
			}
			encrypted, err := core.ExportEncrypted(servers.Profiles, pass)
			if err != nil {
				return fmt.Errorf("encrypted export: %w", err)
			}
			fmt.Println(encrypted)
			return nil
		}

		if exportAll {
			for i := range servers.Profiles {
				fmt.Println(core.ToProxyURL(&servers.Profiles[i]))
			}
			return nil
		}

		var profile *config.Profile
		if len(args) > 0 {
			name := args[0]
			for i := range servers.Profiles {
				if servers.Profiles[i].Name == name {
					profile = &servers.Profiles[i]
					break
				}
			}
			if profile == nil {
				return errProfileNotFound(name)
			}
		} else {
			profile = servers.DefaultProfile()
			if profile == nil {
				return errNoDefaultProfile()
			}
		}

		url := core.ToProxyURL(profile)
		if exportQR {
			qrterminal.GenerateWithConfig(url, qrterminal.Config{
				Level:     qrterminal.L,
				Writer:    os.Stdout,
				BlackChar: qrterminal.BLACK,
				WhiteChar: qrterminal.WHITE,
				QuietZone: 1,
			})
			fmt.Println(url)
		} else {
			fmt.Println(url)
		}
		return nil
	},
}

func init() {
	exportCmd.Flags().BoolVar(&exportAll, "all", false, "Export all profiles")
	exportCmd.Flags().BoolVar(&exportQR, "qr", false, "Display proxy URL as QR code in terminal")
	exportCmd.Flags().BoolVar(&exportEncrypt, "encrypt", false, "Export all profiles encrypted (passphrase from --passphrase-file, LAZYRAY_PASSPHRASE, or prompt)")
	exportCmd.Flags().StringVar(&exportPassphraseFile, "passphrase-file", "", "Read the encryption passphrase from the first line of this file")
	rootCmd.AddCommand(exportCmd)
}
