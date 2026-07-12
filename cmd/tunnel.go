package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/rtxnik/lazyray/internal/clihint"
	"github.com/rtxnik/lazyray/internal/config"
	"github.com/rtxnik/lazyray/internal/core"
	"github.com/spf13/cobra"
)

var tunnelManager = core.NewTunnelManager()

// readTrustLine reads one line of interactive confirmation. Seam for tests.
var readTrustLine = func() (string, error) {
	return bufio.NewReader(os.Stdin).ReadString('\n')
}

var tunnelCmd = &cobra.Command{
	Use:     "tunnel [name]",
	Aliases: []string{"ssh-tunnel"},
	Short:   "Manage SSH tunnels to server panels",
	Long: `Open or inspect an SSH tunnel to a profile's management panel.

This is an SSH tunnel to the server's admin panel — separate from the proxy itself: it does not route your traffic through xray-core and does not change the active proxy profile. With no argument it lists SSH-capable profiles and their tunnel state; with a profile name it opens a persistent tunnel to that profile's panel. Tear tunnels down with 'lzr tunnel close'.`,
	Example: "  lzr tunnel\n  lzr tunnel ru\n  lzr tunnel close",
	Args:    cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return tunnelStatus()
		}

		servers, err := config.LoadServers()
		if err != nil {
			return fmt.Errorf("loading servers: %w", err)
		}

		return tunnelConnectByName(servers, strings.ToLower(args[0]))
	},
}

var tunnelCloseCmd = &cobra.Command{
	Use:     "close",
	Short:   "Close all SSH tunnels",
	Long:    "Close every open SSH tunnel that 'lzr tunnel' started. SSH tunnels persist after the command that opened them exits, so use this to tear them all down.",
	Example: "  lzr tunnel close",
	RunE: func(cmd *cobra.Command, args []string) error {
		tunnelManager.DisconnectAll()
		fmt.Println("All tunnels closed")
		return nil
	},
}

var tunnelTrustFingerprints []string

var tunnelTrustCmd = &cobra.Command{
	Use:   "trust <name>",
	Short: "Pin or re-pin a profile's SSH host key",
	Long: `Capture the SSH host key of a profile's server, show its SHA256 fingerprint, and pin it into the profile after explicit confirmation.

Verify the fingerprint out-of-band before confirming (on the server: ssh-keygen -lf /etc/ssh/ssh_host_*.pub). With --fingerprint the command is non-interactive: only captured keys whose fingerprints match the provided values are pinned, and a value that matches nothing is an error. Re-running the command replaces a previous pin (shown as "old") after the same confirmation. Pinning via a dedicated known_hosts disables OpenSSH's automatic host-key rotation, so re-run this command when the server legitimately rotates its keys.`,
	Example: "  lzr tunnel trust ru\n  lzr tunnel trust ru --fingerprint SHA256:mVN1EX9nGiimZzXFqXTZHrpx5RCasCMEEyBGavrfBFo",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		servers, err := config.LoadServers()
		if err != nil {
			return fmt.Errorf("loading servers: %w", err)
		}
		return tunnelTrust(servers, strings.ToLower(args[0]), tunnelTrustFingerprints)
	},
}

func tunnelTrust(servers *config.ServersConfig, target string, fingerprints []string) error {
	p := findTunnelProfile(servers, target)
	if p == nil {
		return errProfileNotFound(target)
	}
	if p.SSH.Host == "" {
		return fmt.Errorf("no SSH configuration for profile %s", core.StripControl(p.Name))
	}
	if err := core.ValidateSSHTarget(p.SSH.User, p.SSH.Host); err != nil {
		return err
	}
	if len(fingerprints) == 0 && !stdinIsTerminal() {
		return clihint.Errorf(
			"pass --fingerprint SHA256:... for non-interactive pinning",
			"refusing to pin without confirmation")
	}
	captured, err := core.CaptureHostKeys(p.SSH.Host, p.SSH.Port)
	if err != nil {
		return fmt.Errorf("cannot reach %s to capture its host key: %w", p.SSH.Host, err)
	}

	if len(fingerprints) > 0 {
		matched, err := matchFingerprints(captured, fingerprints)
		if err != nil {
			return err
		}
		pinHostKeys(p, matched)
	} else {
		if pinned, err := core.ParseHostKeys(p.SSH.HostKeys); err == nil && len(pinned) > 0 {
			printHostKeyFingerprints(os.Stderr, "Currently pinned (old)", pinned)
		}
		printHostKeyFingerprints(os.Stderr, "Live capture (new)", captured)
		fmt.Fprintln(os.Stderr, "Verify out-of-band on the server: ssh-keygen -lf /etc/ssh/ssh_host_*.pub")
		fmt.Fprint(os.Stderr, "Pin these keys? [y/N]: ")
		line, err := readTrustLine()
		if err != nil {
			return fmt.Errorf("reading confirmation: %w", err)
		}
		switch strings.ToLower(strings.TrimSpace(line)) {
		case "y", "yes":
		default:
			return fmt.Errorf("aborted; nothing pinned")
		}
		pinHostKeys(p, captured)
	}

	if err := config.SaveServers(servers); err != nil {
		return fmt.Errorf("saving trusted host key: %w", err)
	}
	fmt.Printf("Pinned %d host key(s) for %s\n", len(p.SSH.HostKeys), core.StripControl(p.Name))
	return nil
}

// matchFingerprints returns exactly the captured keys whose fingerprints are
// listed in wanted. A wanted value matching no captured key is an error —
// nothing gets pinned on a partial verification failure. Repeated wanted
// values are deduplicated so a fingerprint passed twice pins one key once.
func matchFingerprints(captured []core.HostKey, wanted []string) ([]core.HostKey, error) {
	byFP := make(map[string]core.HostKey, len(captured))
	for _, k := range captured {
		fp, err := k.Fingerprint()
		if err != nil {
			return nil, err
		}
		byFP[fp] = k
	}
	var matched []core.HostKey
	added := make(map[string]bool, len(wanted))
	for _, w := range wanted {
		k, ok := byFP[w]
		if !ok {
			return nil, fmt.Errorf("fingerprint %s does not match any captured host key; nothing pinned", w)
		}
		if !added[w] {
			added[w] = true
			matched = append(matched, k)
		}
	}
	return matched, nil
}

// findTunnelProfile resolves a profile by name: exact (case-insensitive)
// matches always win; prefix matching is a second pass only. A trust
// decision must bind to the profile the user actually named.
func findTunnelProfile(servers *config.ServersConfig, target string) *config.Profile {
	for i := range servers.Profiles {
		if strings.EqualFold(servers.Profiles[i].Name, target) {
			return &servers.Profiles[i]
		}
	}
	for i := range servers.Profiles {
		if matchesShortName(servers.Profiles[i].Name, target) {
			return &servers.Profiles[i]
		}
	}
	return nil
}

func tunnelConnectByName(servers *config.ServersConfig, target string) error {
	p := findTunnelProfile(servers, target)
	if p == nil {
		return errProfileNotFound(target)
	}

	err := tunnelManager.Connect(p)
	var unknown *core.ErrHostKeyUnknown
	if errors.As(err, &unknown) {
		err = trustAndRetry(servers, p, unknown)
	}
	var changed *core.ErrHostKeyChanged
	if errors.As(err, &changed) {
		printHostKeyFingerprints(os.Stderr, "Pinned (old)", changed.Pinned)
		printHostKeyFingerprints(os.Stderr, "Live (new)", changed.Captured)
		return clihint.Errorf(
			"if the change is expected, re-pin with 'lzr tunnel trust "+core.StripControl(p.Name)+"'",
			"refusing to connect: host key for %s changed (possible MITM)", changed.Host)
	}
	if err != nil {
		return err
	}

	statuses := tunnelManager.Status(servers.Profiles)
	for _, s := range statuses {
		if s.Name == p.Name && s.Connected {
			fmt.Printf("Connected to %s (PID %d)\n", core.StripControl(s.Name), s.PID)
			fmt.Printf("  Panel: %s\n", s.PanelURL)
			fmt.Println("  Tunnel will persist after this command exits")
			fmt.Println("  Close with: lzr tunnel close")
			return nil
		}
	}
	return nil
}

// trustAndRetry runs the first-connect TOFU confirmation, pins on consent,
// and retries the connect. Non-interactive sessions refuse with instructions
// instead of pinning blind.
func trustAndRetry(servers *config.ServersConfig, p *config.Profile, unknown *core.ErrHostKeyUnknown) error {
	if !stdinIsTerminal() {
		return clihint.Errorf(
			"run 'lzr tunnel trust "+core.StripControl(p.Name)+"' interactively (or with --fingerprint) first",
			"host %s is not trusted yet", unknown.Host)
	}
	fmt.Fprintf(os.Stderr, "First connection to %s (%s).\n",
		core.StripControl(p.Name), net.JoinHostPort(unknown.Host, strconv.Itoa(unknown.Port)))
	printHostKeyFingerprints(os.Stderr, "Host key fingerprints", unknown.Captured)
	fmt.Fprintln(os.Stderr, "Verify out-of-band on the server: ssh-keygen -lf /etc/ssh/ssh_host_*.pub")
	fmt.Fprint(os.Stderr, "Trust this host? [y/N]: ")
	line, err := readTrustLine()
	if err != nil {
		return fmt.Errorf("reading confirmation: %w", err)
	}
	switch strings.ToLower(strings.TrimSpace(line)) {
	case "y", "yes":
	default:
		return fmt.Errorf("host not trusted; tunnel aborted")
	}
	pinHostKeys(p, unknown.Captured)
	if err := config.SaveServers(servers); err != nil {
		return fmt.Errorf("saving trusted host key: %w", err)
	}
	return tunnelManager.Connect(p)
}

// pinHostKeys stores keys into the profile in the persistent line form.
func pinHostKeys(p *config.Profile, keys []core.HostKey) {
	lines := make([]string, 0, len(keys))
	for _, k := range keys {
		lines = append(lines, k.String())
	}
	p.SSH.HostKeys = lines
}

// printHostKeyFingerprints renders one "type SHA256:..." line per key.
func printHostKeyFingerprints(w io.Writer, label string, keys []core.HostKey) {
	fmt.Fprintf(w, "%s:\n", label)
	if len(keys) == 0 {
		fmt.Fprintln(w, "  (none)")
		return
	}
	for _, k := range keys {
		fp, err := k.Fingerprint()
		if err != nil {
			fp = "(invalid: " + err.Error() + ")"
		}
		fmt.Fprintf(w, "  %s %s\n", k.Type, fp)
	}
}

func tunnelStatus() error {
	servers, err := config.LoadServers()
	if err != nil {
		return fmt.Errorf("loading servers: %w", err)
	}

	statuses := tunnelManager.Status(servers.Profiles)
	if len(statuses) == 0 {
		fmt.Println("No SSH-capable profiles configured")
		return nil
	}

	for _, s := range statuses {
		state := "disconnected"
		if s.Connected {
			state = fmt.Sprintf("connected (PID %d) → %s", s.PID, s.PanelURL)
		}
		fmt.Printf("  %s: %s\n", core.StripControl(s.Name), state)
	}
	return nil
}

func matchesShortName(profileName, target string) bool {
	// Allow matching "al" to "Alpha→Beta Cascade" etc.
	lower := strings.ToLower(profileName)
	return strings.HasPrefix(lower, target)
}

func init() {
	tunnelTrustCmd.Flags().StringArrayVar(&tunnelTrustFingerprints, "fingerprint", nil,
		"pin only captured keys matching this SHA256 fingerprint (repeatable; enables non-interactive pinning)")
	tunnelCmd.AddCommand(tunnelCloseCmd)
	tunnelCmd.AddCommand(tunnelTrustCmd)
	rootCmd.AddCommand(tunnelCmd)
}
