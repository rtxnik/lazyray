//go:build !windows

package core

import (
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"

	"github.com/rtxnik/lazyray/internal/config"
)

// processCmdline returns pid's command line. It is a package var so tests can
// substitute a deterministic cmdline without spawning real processes.
var processCmdline = func(pid int) (string, error) {
	if runtime.GOOS == "darwin" {
		out, err := exec.Command("ps", "-o", "command=", "-p", strconv.Itoa(pid)).Output()
		return string(out), err
	}
	data, err := os.ReadFile("/proc/" + strconv.Itoa(pid) + "/cmdline")
	if err != nil {
		return "", err
	}
	return strings.ReplaceAll(string(data), "\x00", " "), nil
}

// SetProcessCmdlineForTest swaps the cmdline reader for the duration of a test
// and returns a restore func. Exported so consumers in other packages (e.g.
// internal/lifecycle) can drive the shared identity check deterministically.
func SetProcessCmdlineForTest(fn func(pid int) (string, error)) (restore func()) {
	orig := processCmdline
	processCmdline = fn
	return func() { processCmdline = orig }
}

func isOurXray(pid int) bool {
	if pid <= 0 {
		return false
	}
	cmdline, err := processCmdline(pid)
	if err != nil {
		return false
	}
	return strings.Contains(cmdline, config.XrayBinaryPath())
}

func isOurTunnel(pid int) bool {
	if pid <= 0 {
		return false
	}
	cmdline, err := processCmdline(pid)
	if err != nil {
		return false
	}
	// Weak signature: lazyray tunnels are always `ssh … -N … -L …`.
	return strings.Contains(cmdline, "ssh") &&
		strings.Contains(cmdline, "-N") &&
		strings.Contains(cmdline, "-L")
}
