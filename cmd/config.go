package cmd

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/rtxnik/lazyray/internal/config"
	"github.com/rtxnik/lazyray/internal/core"
	"github.com/rtxnik/lazyray/internal/fsutil"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage configuration",
	Long:  `Manage the profile store and the generated xray config. The "config" group covers two distinct things: the profile store (the YAML at servers.yaml/lazyray.yaml that holds your proxy profiles and settings) and the generated xray config (the xray-core JSON that 'config show' and 'config edit' open). Use the subcommands to list, switch, inspect, duplicate, delete, back up, and restore profiles.`,
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current xray configuration",
	Long:  `Print the generated xray config (the xray-core JSON that lazyray builds from your active proxy profile) to stdout. Use it to inspect exactly what xray-core will run, or to copy the config elsewhere.`,
	Example: `  lzr config show
  lzr config show | jq .`,
	RunE: func(cmd *cobra.Command, args []string) error {
		data, err := os.ReadFile(config.XrayConfigPath())
		if err != nil {
			return fmt.Errorf("reading xray config: %w", err)
		}
		fmt.Println(string(data))
		return nil
	},
}

var configListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all profiles",
	Long:  `List every proxy profile in the profile store, marking the default with an asterisk. Use it to see what you can switch to or export, and to find the exact profile name other commands expect.`,
	Example: `  lzr config list
  lzr config list --json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		servers, err := config.LoadServers()
		if err != nil {
			return fmt.Errorf("loading servers: %w", err)
		}

		if len(servers.Profiles) == 0 {
			fmt.Println("No profiles configured. Use 'lzr import <vless://...>' to add one.")
			return nil
		}

		jsonFlag, _ := cmd.Flags().GetBool("json")
		if jsonFlag {
			var items []map[string]interface{}
			for i, p := range servers.Profiles {
				items = append(items, map[string]interface{}{
					"index":   i,
					"name":    p.Name,
					"default": p.Default,
					"server":  fmt.Sprintf("%s:%d", p.Server.Address, p.Server.Port),
				})
			}
			out, _ := json.MarshalIndent(items, "", "  ")
			fmt.Println(string(out))
			return nil
		}

		for i, p := range servers.Profiles {
			marker := "  "
			if p.Default {
				marker = "* "
			}
			fmt.Printf("%s[%d] %s (%s:%d)\n", marker, i, core.StripControl(p.Name), core.StripControl(p.Server.Address), p.Server.Port)
		}
		return nil
	},
}

var configSwitchCmd = &cobra.Command{
	Use:   "switch <name>",
	Short: "Switch active profile",
	Long:  `Make the named proxy profile the default, so subsequent connects and exports use it. Run 'lzr config list' first to see the available profile names.`,
	Example: `  lzr config switch home
  lzr config switch "work vpn"`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		servers, err := config.LoadServers()
		if err != nil {
			return fmt.Errorf("loading servers: %w", err)
		}

		name := args[0]
		found := -1
		for i, p := range servers.Profiles {
			if p.Name == name {
				found = i
				break
			}
		}

		if found == -1 {
			return errProfileNotFound(name)
		}

		if err := servers.SetDefault(found); err != nil {
			return err
		}

		if err := config.SaveServers(servers); err != nil {
			return fmt.Errorf("saving servers: %w", err)
		}

		fmt.Printf("Switched to profile: %s\n", core.StripControl(servers.Profiles[found].Name))
		return nil
	},
}

var configEditCmd = &cobra.Command{
	Use:   "edit",
	Short: "Open configuration in editor",
	Long:  `Open the generated xray config (the xray-core JSON) in your editor, using $EDITOR (falling back to vi). Use it for manual tweaks; lazyray rewrites this file whenever you import or switch a proxy profile, so durable changes belong in the profile store.`,
	Example: `  lzr config edit
  EDITOR=nano lzr config edit`,
	RunE: func(cmd *cobra.Command, args []string) error {
		editor := os.Getenv("EDITOR")
		if editor == "" {
			editor = "vi"
		}

		configPath := config.XrayConfigPath()
		c := exec.Command(editor, configPath)
		c.Stdin = os.Stdin
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		return c.Run()
	},
}

var configDeleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Delete a profile",
	Long:  `Remove the named proxy profile from the profile store. If you delete the default profile, the first remaining profile becomes the new default. Run 'lzr config list' to confirm the exact name.`,
	Example: `  lzr config delete old-server
  lzr config delete "work vpn"`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		servers, err := config.LoadServers()
		if err != nil {
			return fmt.Errorf("loading servers: %w", err)
		}

		name := args[0]
		found := -1
		for i, p := range servers.Profiles {
			if p.Name == name {
				found = i
				break
			}
		}

		if found == -1 {
			return errProfileNotFound(name)
		}

		wasDefault := servers.Profiles[found].Default
		servers.Profiles = append(servers.Profiles[:found], servers.Profiles[found+1:]...)

		// If deleted profile was default, set the first remaining as default
		if wasDefault && len(servers.Profiles) > 0 {
			servers.Profiles[0].Default = true
		}

		if err := config.SaveServers(servers); err != nil {
			return fmt.Errorf("saving servers: %w", err)
		}

		fmt.Printf("Deleted profile: %s\n", name)
		return nil
	},
}

var (
	backupNoEncrypt       bool
	backupPassphraseFile  string
	restorePassphraseFile string
)

var configBackupCmd = &cobra.Command{
	Use:   "backup [file]",
	Short: "Backup configuration to an encrypted archive",
	Long:  `Write an archive of the profile store and the generated xray config (servers.yaml, lazyray.yaml, config.json). Archives are encrypted by default because they bundle proxy credentials; the passphrase comes from --passphrase-file, the LAZYRAY_PASSPHRASE environment variable, or an interactive prompt. Pass --no-encrypt for a plaintext tar.gz. Without a path, the archive lands in the backup directory with a timestamped name and old backups are rotated.`,
	Example: `  lzr config backup
  lzr config backup ~/lazyray-backup.tar.gz.enc
  lzr config backup --no-encrypt ~/lazyray-backup.tar.gz
  LAZYRAY_PASSPHRASE=secret lzr config backup`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := config.EnsureDirs(); err != nil {
			return err
		}

		var passphrase string
		if !backupNoEncrypt {
			var err error
			passphrase, err = resolvePassphrase(backupPassphraseFile, true)
			if errors.Is(err, errNoPassphraseSource) {
				return fmt.Errorf("backups are encrypted by default: provide --passphrase-file, set %s, run interactively, or pass --no-encrypt", passphraseEnvVar)
			}
			if err != nil {
				return err
			}
		}

		outPath := ""
		if len(args) > 0 {
			outPath = args[0]
		} else {
			ext := ".tar.gz.enc"
			if backupNoEncrypt {
				ext = ".tar.gz"
			}
			ts := time.Now().Format("20060102-150405")
			outPath = filepath.Join(config.BackupDir(), fmt.Sprintf("lazyray-backup-%s%s", ts, ext))
		}

		files := []struct {
			path string
			name string
		}{
			{config.ServersPath(), "servers.yaml"},
			{config.SettingsPath(), "lazyray.yaml"},
			{config.XrayConfigPath(), "config.json"},
		}

		var buf bytes.Buffer
		gw := gzip.NewWriter(&buf)
		tw := tar.NewWriter(gw)
		for _, file := range files {
			data, err := os.ReadFile(file.path)
			if err != nil {
				if os.IsNotExist(err) {
					continue
				}
				return fmt.Errorf("reading %s: %w", file.name, err)
			}
			hdr := &tar.Header{
				Name: file.name,
				Mode: 0600,
				Size: int64(len(data)),
			}
			if err := tw.WriteHeader(hdr); err != nil {
				return fmt.Errorf("writing tar header for %s: %w", file.name, err)
			}
			if _, err := tw.Write(data); err != nil {
				return fmt.Errorf("writing tar data for %s: %w", file.name, err)
			}
		}
		if err := tw.Close(); err != nil {
			return fmt.Errorf("finalizing archive: %w", err)
		}
		if err := gw.Close(); err != nil {
			return fmt.Errorf("finalizing archive: %w", err)
		}

		out := buf.Bytes()
		note := ""
		if !backupNoEncrypt {
			blob, err := core.EncryptData(out, passphrase)
			if err != nil {
				return fmt.Errorf("encrypting backup: %w", err)
			}
			out = []byte(blob + "\n")
			note = " (encrypted)"
		}
		// 0600 + atomic rename: never world-readable, never follows a
		// planted symlink at the destination.
		if err := fsutil.WriteFile(outPath, out, 0o600); err != nil {
			return fmt.Errorf("writing backup file: %w", err)
		}

		fmt.Printf("Backup saved to: %s%s\n", outPath, note)

		// Rotate old backups
		settings, _ := config.LoadSettings()
		if settings == nil {
			settings = config.DefaultSettings()
		}
		core.RotateBackups(settings.Backup.MaxFiles)

		return nil
	},
}

var configRestoreCmd = &cobra.Command{
	Use:   "restore <file>",
	Short: "Restore configuration from a backup archive",
	Long:  `Restore the profile store and the generated xray config from an archive made by 'lzr config backup'. Encrypted archives are detected automatically; the passphrase comes from --passphrase-file, the LAZYRAY_PASSPHRASE environment variable, or an interactive prompt. Plain tar.gz archives from older versions restore without a passphrase. Recognized members (servers.yaml, lazyray.yaml, config.json) overwrite the current files.`,
	Example: `  lzr config restore ~/lazyray-backup.tar.gz.enc
  lzr config restore ./lazyray-backup-20260101-120000.tar.gz`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := config.EnsureDirs(); err != nil {
			return err
		}

		raw, err := os.ReadFile(args[0])
		if err != nil {
			return fmt.Errorf("opening backup file: %w", err)
		}

		// Sniff a trimmed view for the encryption prefix, but feed the
		// ORIGINAL bytes to gzip when plaintext: a binary stream may
		// legitimately end in whitespace-valued bytes.
		archive := raw
		if blob := strings.TrimSpace(string(raw)); core.IsEncryptedExport(blob) {
			passphrase, perr := resolvePassphrase(restorePassphraseFile, false)
			if errors.Is(perr, errNoPassphraseSource) {
				return fmt.Errorf("this backup is encrypted: provide --passphrase-file, set %s, or run interactively", passphraseEnvVar)
			}
			if perr != nil {
				return perr
			}
			archive, err = core.DecryptData(blob, passphrase)
			if err != nil {
				return fmt.Errorf("decrypting backup: %w", err)
			}
		}

		gr, err := gzip.NewReader(bytes.NewReader(archive))
		if err != nil {
			return fmt.Errorf("reading gzip: %w", err)
		}
		defer gr.Close()

		tr := tar.NewReader(gr)

		destMap := map[string]string{
			"servers.yaml": config.ServersPath(),
			"lazyray.yaml": config.SettingsPath(),
			"config.json":  config.XrayConfigPath(),
		}

		restored := 0
		for {
			hdr, err := tr.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				return fmt.Errorf("reading tar: %w", err)
			}

			destPath, ok := destMap[hdr.Name]
			if !ok {
				continue
			}

			data, err := io.ReadAll(tr)
			if err != nil {
				return fmt.Errorf("reading %s from archive: %w", hdr.Name, err)
			}

			// Atomic rename replaces a planted symlink instead of
			// following it.
			if err := fsutil.WriteFile(destPath, data, 0o600); err != nil {
				return fmt.Errorf("writing %s: %w", hdr.Name, err)
			}

			fmt.Printf("Restored: %s\n", hdr.Name)
			restored++
		}

		if restored == 0 {
			return fmt.Errorf("no recognized files found in archive")
		}

		fmt.Printf("Restored %d file(s) from backup\n", restored)
		return nil
	},
}

var configDuplicateCmd = &cobra.Command{
	Use:   "duplicate <name>",
	Short: "Duplicate a profile",
	Long:  `Copy the named proxy profile to a new "(copy)" profile that is not the default. Use it as a starting point when you want a variant of an existing proxy server without re-importing it. Run 'lzr config list' to confirm the source name.`,
	Example: `  lzr config duplicate home
  lzr config duplicate "work vpn"`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		servers, err := config.LoadServers()
		if err != nil {
			return fmt.Errorf("loading servers: %w", err)
		}

		name := args[0]
		var source *config.Profile
		for i := range servers.Profiles {
			if servers.Profiles[i].Name == name {
				source = &servers.Profiles[i]
				break
			}
		}

		if source == nil {
			return errProfileNotFound(name)
		}

		dup := source.Clone()
		dup.Name = source.Name + " (copy)"
		dup.Default = false

		servers.Profiles = append(servers.Profiles, dup)

		if err := config.SaveServers(servers); err != nil {
			return fmt.Errorf("saving servers: %w", err)
		}

		fmt.Printf("Duplicated profile: %s → %s\n", core.StripControl(source.Name), core.StripControl(dup.Name))
		return nil
	},
}

func init() {
	configListCmd.Flags().Bool("json", false, "Output in JSON format")
	configBackupCmd.Flags().BoolVar(&backupNoEncrypt, "no-encrypt", false, "Write a plaintext archive instead of an encrypted one")
	configBackupCmd.Flags().StringVar(&backupPassphraseFile, "passphrase-file", "", "Read the encryption passphrase from the first line of this file")
	configRestoreCmd.Flags().StringVar(&restorePassphraseFile, "passphrase-file", "", "Read the decryption passphrase from the first line of this file")
	configBackupCmd.MarkFlagsMutuallyExclusive("no-encrypt", "passphrase-file")
	configCmd.AddCommand(configShowCmd, configListCmd, configSwitchCmd, configEditCmd, configDeleteCmd, configBackupCmd, configRestoreCmd, configDuplicateCmd)
	rootCmd.AddCommand(configCmd)
}
