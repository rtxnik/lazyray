package cmd

import (
	"errors"
	"fmt"

	"github.com/rtxnik/lazyray/internal/app"
	"github.com/rtxnik/lazyray/internal/clihint"
	"github.com/rtxnik/lazyray/internal/config"
	"github.com/rtxnik/lazyray/internal/core"
	"github.com/spf13/cobra"
)

var (
	importName         string
	importForce        bool
	importSub          string
	importDecrypt      string
	importAllowRouting bool
)

var importCmd = &cobra.Command{
	Use:   "import [url]",
	Short: "Import proxy configuration URL or subscription",
	Long:  `Import a single proxy URL (VLESS, VMess, Trojan, Shadowsocks, Hysteria2) into a new proxy profile, or import all profiles from a subscription URL with --sub. The first imported profile becomes the default.`,
	Example: `  lzr import vless://uuid@host:port?params#name
  lzr import vmess://base64...
  lzr import trojan://pass@host:port?params#name
  lzr import --sub https://example.com/sub`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if importSub != "" {
			return importSubscription(cmd, importSub)
		}

		if len(args) == 0 {
			return clihint.Errorf("see usage with 'lzr import --help'", "provide a proxy URL (vless://, vmess://, trojan://, ss://, hysteria2://) or use --sub/--decrypt for import")
		}

		// Check if encrypted import
		if importDecrypt != "" || core.IsEncryptedExport(args[0]) {
			return importEncrypted(cmd, args[0])
		}

		return importSingleProfile(cmd, args[0])
	},
}

func importSingleProfile(cmd *cobra.Command, rawURL string) error {
	profile, err := core.ParseProxyURL(rawURL)
	if err != nil {
		return fmt.Errorf("parsing URL: %w", err)
	}

	if importName != "" {
		profile.Name = core.StripControl(importName)
	}

	if err := core.ValidateProfile(profile); err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "Warning: %v\n", err)
	}

	if core.Hysteria2HasPortHopping(rawURL) {
		fmt.Fprintf(cmd.ErrOrStderr(), "Note: port-hopping range preserved (base port %d).\n", profile.Server.Port)
	}

	servers, err := config.LoadServers()
	if err != nil {
		return fmt.Errorf("loading servers: %w", err)
	}

	svc := app.NewService()
	if _, err := svc.ImportProfile(servers, profile, importForce); err != nil {
		var dup *app.DuplicateUUIDError
		if errors.As(err, &dup) {
			return fmt.Errorf("UUID already used by profile %q (use --force to import anyway)", dup.ExistingName)
		}
		return fmt.Errorf("saving servers: %w", err)
	}

	// Generate xray config with the new profile
	settings, err := config.LoadSettings()
	if err != nil {
		return fmt.Errorf("loading settings: %w", err)
	}

	if err := svc.WriteActiveConfig(profile, settings); err != nil {
		return fmt.Errorf("writing xray config: %w", err)
	}

	fmt.Printf("Imported profile: %s\n", core.StripControl(profile.Name))
	fmt.Printf("  Server:  %s:%d\n", core.StripControl(profile.Server.Address), profile.Server.Port)
	fmt.Printf("  UUID:    %s\n", config.MaskUUID(profile.Server.UUID))
	fmt.Printf("  Network: %s\n", core.StripControl(profile.Server.Transport.Network))
	fmt.Println("Config written to", config.XrayConfigPath())
	return nil
}

func importSubscription(cmd *cobra.Command, subURL string) error {
	servers, err := config.LoadServers()
	if err != nil {
		return fmt.Errorf("loading servers: %w", err)
	}

	subName := importName
	if subName == "" {
		subName = "subscription"
	}

	fmt.Printf("Fetching subscription: %s\n", subURL)

	svc := app.NewService()
	added, updated, err := svc.ImportSubscription(servers, subURL, subName)
	if err != nil {
		return fmt.Errorf("importing subscription: %w", err)
	}

	if err := config.SaveServers(servers); err != nil {
		return fmt.Errorf("saving servers: %w", err)
	}

	// Save subscription URL to settings
	settings, err := config.LoadSettings()
	if err != nil {
		return fmt.Errorf("loading settings: %w", err)
	}
	settings.UpsertSubscription(subURL, subName)
	if err := config.SaveSettings(settings); err != nil {
		return fmt.Errorf("saving settings: %w", err)
	}

	fmt.Printf("Subscription imported: %d added, %d updated\n", added, updated)
	return nil
}

func importEncrypted(cmd *cobra.Command, data string) error {
	password := importDecrypt
	if password == "" {
		return clihint.Errorf("supply the password with 'lzr import --decrypt <password> <data>'", "provide password with --decrypt flag for encrypted import")
	}

	profiles, err := core.ImportEncrypted(data, password)
	if err != nil {
		return fmt.Errorf("decrypting import: %w", err)
	}

	servers, err := config.LoadServers()
	if err != nil {
		return fmt.Errorf("loading servers: %w", err)
	}

	added := 0
	for _, p := range profiles {
		if core.HasRoutingOverrides(&p) {
			if !importAllowRouting {
				fmt.Fprintf(cmd.ErrOrStderr(),
					"Warning: %q carries routing/DNS overrides (%d bypass, %d block, %d dns rules); dropping them. Re-import with --allow-routing to keep them.\n",
					p.Name, len(p.Routing.Bypass), len(p.Routing.Block), len(p.Routing.DNSRules))
				p.Routing = config.ProfileRouting{}
			} else {
				bad := false
				for _, rule := range p.Routing.DNSRules {
					if err := core.ValidateDNSServer(rule.Server); err != nil {
						fmt.Fprintf(cmd.ErrOrStderr(), "Skipping %q: %v\n", p.Name, err)
						bad = true
						break
					}
				}
				if bad {
					continue
				}
			}
		}
		if _, exists := servers.HasUUID(p.Server.UUID); exists && !importForce {
			fmt.Fprintf(cmd.ErrOrStderr(), "Skipping %q (UUID exists, use --force)\n", core.StripControl(p.Name))
			continue
		}
		if len(servers.Profiles) == 0 {
			p.Default = true
		}
		servers.Profiles = append(servers.Profiles, p)
		added++
	}

	if err := config.SaveServers(servers); err != nil {
		return fmt.Errorf("saving servers: %w", err)
	}

	fmt.Printf("Encrypted import: %d profiles added\n", added)
	return nil
}

func init() {
	importCmd.Flags().StringVarP(&importName, "name", "n", "", "Profile name (default: from URL fragment)")
	importCmd.Flags().BoolVarP(&importForce, "force", "f", false, "Import even if UUID already exists")
	importCmd.Flags().StringVar(&importSub, "sub", "", "Import from subscription URL")
	importCmd.Flags().StringVar(&importDecrypt, "decrypt", "", "Decrypt encrypted export with password")
	importCmd.Flags().BoolVar(&importAllowRouting, "allow-routing", false, "Honor routing/DNS overrides carried by an encrypted import (validated against an allowlist)")
	rootCmd.AddCommand(importCmd)
}
