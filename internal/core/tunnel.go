package core

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/rtxnik/lazyray/internal/config"
	"github.com/rtxnik/lazyray/internal/fsutil"
	"github.com/rtxnik/lazyray/internal/procutil"
)

// TunnelStatus represents SSH tunnel state.
type TunnelStatus struct {
	Name      string
	Connected bool
	PID       int
	LocalPort int
	PanelURL  string
}

// TunnelManager manages SSH tunnels.
type TunnelManager struct {
	mu      sync.Mutex
	tunnels map[string]*tunnel
}

type tunnel struct {
	cmd       *exec.Cmd
	localPort int
	profile   *config.Profile
}

// newSSHCmd builds the tunnel command with a detached session so the tunnel
// survives after the CLI process exits (cross-session recovery). buildSSHArgs
// produces a plain `ssh -N -L …` with no child-spawning options, so the leader
// pid is the whole tunnel.
func newSSHCmd(args []string) *exec.Cmd {
	cmd := exec.Command("ssh", args...)
	cmd.SysProcAttr = detachedProcAttr()
	return cmd
}

// startSSHProcess launches the tunnel child process. Seam for tests.
var startSSHProcess = func(args []string) (*exec.Cmd, error) {
	cmd := newSSHCmd(args)
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return cmd, nil
}

// SetStartSSHProcessForTest swaps the ssh spawn and returns a restore func.
func SetStartSSHProcessForTest(fn func(args []string) (*exec.Cmd, error)) (restore func()) {
	prev := startSSHProcess
	startSSHProcess = fn
	return func() { startSSHProcess = prev }
}

// NewTunnelManager creates a new tunnel manager.
func NewTunnelManager() *TunnelManager {
	return &TunnelManager{
		tunnels: make(map[string]*tunnel),
	}
}

// Connect establishes an SSH tunnel to a profile's panel.
func (tm *TunnelManager) Connect(profile *config.Profile) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if t, exists := tm.tunnels[profile.Name]; exists && t.cmd.Process != nil {
		return fmt.Errorf("tunnel to %s is already connected", profile.Name)
	}

	// Check if a persistent tunnel is already running from a previous CLI session
	if pid := readTunnelPID(profile.Name); pid > 0 {
		if isTunnelProcessAlive(pid) {
			return fmt.Errorf("tunnel to %s is already connected (PID %d from previous session)", profile.Name, pid)
		}
		removeTunnelPID(profile.Name)
	}

	if profile.SSH.Host == "" {
		return fmt.Errorf("no SSH configuration for profile %s", profile.Name)
	}
	if err := ValidateSSHTarget(profile.SSH.User, profile.SSH.Host); err != nil {
		return err
	}

	// Host-key trust gate: never spawn ssh for an untrusted or changed host.
	pinned, err := ParseHostKeys(profile.SSH.HostKeys)
	if err != nil {
		return err
	}
	if len(pinned) == 0 {
		captured, err := CaptureHostKeys(profile.SSH.Host, profile.SSH.Port)
		if err != nil {
			return fmt.Errorf("cannot reach %s to verify its identity: %w", StripControl(profile.SSH.Host), err)
		}
		return &ErrHostKeyUnknown{Host: profile.SSH.Host, Port: profile.SSH.Port, Captured: captured}
	}
	if err := verifyPinnedHostKey(profile.SSH.Host, profile.SSH.Port, pinned); err != nil {
		return err
	}

	if err := config.EnsureDirs(); err != nil {
		return fmt.Errorf("preparing data dir: %w", err)
	}
	knownHostsPath := config.TunnelKnownHostsPath(profile.Name)
	if err := fsutil.WriteFile(knownHostsPath, DeriveKnownHosts(profile.SSH.Host, profile.SSH.Port, pinned), 0o600); err != nil {
		return fmt.Errorf("writing known_hosts: %w", err)
	}

	localPort, err := findFreePort()
	if err != nil {
		return fmt.Errorf("finding free port: %w", err)
	}

	cmd, err := startSSHProcess(buildSSHArgs(profile, localPort, knownHostsPath, hostKeyAlgorithmsFor(pinned)))
	if err != nil {
		return fmt.Errorf("starting SSH tunnel: %w", err)
	}

	tm.tunnels[profile.Name] = &tunnel{
		cmd:       cmd,
		localPort: localPort,
		profile:   profile,
	}

	// Persist PID and port for cross-session recovery
	if err := writeTunnelPID(profile.Name, cmd.Process.Pid, localPort); err != nil {
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
		delete(tm.tunnels, profile.Name)
		return fmt.Errorf("recording tunnel pid: %w", err)
	}

	return nil
}

// Disconnect closes an SSH tunnel.
func (tm *TunnelManager) Disconnect(name string) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	t, exists := tm.tunnels[name]
	if !exists {
		// Try to close a persistent tunnel from PID file
		if pid := readTunnelPID(name); pid > 0 {
			if isTunnelProcessAlive(pid) && IsOurTunnel(pid) {
				if proc, err := os.FindProcess(pid); err == nil {
					_ = proc.Kill()
				}
			}
			removeTunnelPID(name)
			return nil
		}
		return fmt.Errorf("no tunnel for %s", name)
	}

	if t.cmd.Process != nil {
		if err := t.cmd.Process.Kill(); err != nil {
			return fmt.Errorf("killing tunnel process: %w", err)
		}
		_, _ = t.cmd.Process.Wait()
	}

	removeTunnelPID(name)
	delete(tm.tunnels, name)
	return nil
}

// DisconnectAll closes all SSH tunnels (in-memory and persistent).
func (tm *TunnelManager) DisconnectAll() {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	for name, t := range tm.tunnels {
		if t.cmd.Process != nil {
			_ = t.cmd.Process.Kill()
			_, _ = t.cmd.Process.Wait()
		}
		removeTunnelPID(name)
		delete(tm.tunnels, name)
	}

	// Also close any orphan tunnels from PID files
	CloseAllPersistentTunnels()
}

// Status returns the status of all tunnels for given profiles.
// Checks both in-memory tunnels and persistent PID files.
func (tm *TunnelManager) Status(profiles []config.Profile) []TunnelStatus {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	var statuses []TunnelStatus
	for _, p := range profiles {
		if p.SSH.Host == "" {
			continue
		}
		status := TunnelStatus{
			Name: p.Name,
		}
		if t, exists := tm.tunnels[p.Name]; exists && t.cmd.Process != nil {
			status.Connected = true
			status.PID = t.cmd.Process.Pid
			status.LocalPort = t.localPort
			status.PanelURL = fmt.Sprintf("https://127.0.0.1:%d%s", t.localPort, p.SSH.Panel.Path)
		} else if pid, port := readTunnelPIDAndPort(p.Name); pid > 0 && isTunnelProcessAlive(pid) && IsOurTunnel(pid) {
			// Found our own persistent tunnel from a previous session
			status.Connected = true
			status.PID = pid
			status.LocalPort = port
			if port > 0 {
				status.PanelURL = fmt.Sprintf("https://127.0.0.1:%d%s", port, p.SSH.Panel.Path)
			}
		} else if pid > 0 {
			// Stale PID file — clean up
			removeTunnelPID(p.Name)
		}
		statuses = append(statuses, status)
	}
	return statuses
}

// CloseAllPersistentTunnels kills all SSH tunnel processes found in PID files.
func CloseAllPersistentTunnels() {
	matches, err := filepath.Glob(config.TunnelPIDGlob())
	if err != nil {
		return
	}
	for _, path := range matches {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		parts := strings.SplitN(strings.TrimSpace(string(data)), "\n", 2)
		pid, err := strconv.Atoi(parts[0])
		if err != nil || pid <= 0 {
			_ = os.Remove(path)
			continue
		}
		if isTunnelProcessAlive(pid) && IsOurTunnel(pid) {
			if proc, err := os.FindProcess(pid); err == nil {
				_ = proc.Kill()
			}
		}
		_ = os.Remove(path)
	}
}

// Tunnel PID file helpers.
// The PID file stores: line 1 = PID, line 2 = local port.

func writeTunnelPID(name string, pid, localPort int) error {
	if err := config.EnsureDirs(); err != nil {
		return err
	}
	data := fmt.Sprintf("%d\n%d", pid, localPort)
	return procutil.WritePIDFile(config.TunnelPIDPath(name), []byte(data))
}

func readTunnelPID(name string) int {
	pid, _ := readTunnelPIDAndPort(name)
	return pid
}

func readTunnelPIDAndPort(name string) (int, int) {
	data, err := os.ReadFile(config.TunnelPIDPath(name))
	if err != nil {
		return 0, 0
	}
	lines := strings.SplitN(strings.TrimSpace(string(data)), "\n", 2)
	pid, err := strconv.Atoi(strings.TrimSpace(lines[0]))
	if err != nil {
		return 0, 0
	}
	port := 0
	if len(lines) > 1 {
		port, _ = strconv.Atoi(strings.TrimSpace(lines[1]))
	}
	return pid, port
}

func removeTunnelPID(name string) {
	_ = os.Remove(config.TunnelPIDPath(name))
}

func isTunnelProcessAlive(pid int) bool {
	return isProcessAlive(pid)
}

func findFreePort() (int, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	port := ln.Addr().(*net.TCPAddr).Port
	ln.Close()
	return port, nil
}

// buildSSHArgs assembles the tunnel argv. Host-key verification is enforced
// by ssh itself against the derived per-profile known_hosts — an independent
// second layer behind the pre-flight dial, so a network change between the
// two still cannot bypass pinning. "--" plus ValidateSSHTarget close the
// option-injection hole (leading-'-' User/Host parsing as ssh options).
func buildSSHArgs(profile *config.Profile, localPort int, knownHostsPath string, algos []string) []string {
	forward := fmt.Sprintf("%d:127.0.0.1:%d", localPort, profile.SSH.Panel.Port)
	return []string{
		"-L", forward,
		"-p", strconv.Itoa(profile.SSH.Port),
		"-i", profile.SSH.KeyPath,
		"-o", "StrictHostKeyChecking=yes",
		"-o", "UserKnownHostsFile=" + knownHostsPath,
		"-o", "GlobalKnownHostsFile=/dev/null",
		"-o", "HostKeyAlgorithms=" + strings.Join(algos, ","),
		"-o", "ServerAliveInterval=30",
		"-o", "ServerAliveCountMax=3",
		"-N",
		"--",
		fmt.Sprintf("%s@%s", profile.SSH.User, profile.SSH.Host),
	}
}
