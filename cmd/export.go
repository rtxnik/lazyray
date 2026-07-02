package cmd

import (
	"fmt"
	"os"

	qrterminal "github.com/mdp/qrterminal/v3"
	"github.com/rtxnik/lazyray/internal/config"
	"github.com/rtxnik/lazyray/internal/core"
	"github.com/spf13/cobra"
)

var (
	exportAll       bool
	exportQR        bool
	exportEncrypted string
)

var exportCmd = &cobra.Command{
	Use:   "export [name]",
	Short: "Export profile as proxy URL",
	Long:  "Export a proxy profile as its protocol URL (VLESS, VMess, or Trojan). Without arguments, exports the default profile.",
	Example: `  lzr export
  lzr export home
  lzr export --all
  lzr export --qr
  lzr export --encrypt "passphrase"`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		servers, err := config.LoadServers()
		if err != nil {
			return fmt.Errorf("loading servers: %w", err)
		}

		if len(servers.Profiles) == 0 {
			return errNoProfilesConfigured()
		}

		// Encrypted export of all profiles
		if exportEncrypted != "" {
			encrypted, err := core.ExportEncrypted(servers.Profiles, exportEncrypted)
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
	exportCmd.Flags().StringVar(&exportEncrypted, "encrypt", "", "Export all profiles encrypted with password")
	rootCmd.AddCommand(exportCmd)
}
