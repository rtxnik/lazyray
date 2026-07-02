//go:build windows

package cmd

import "os"

// forceKillSupervisor hard-kills the supervisor.
func forceKillSupervisor(pid int) error {
	p, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	return p.Kill()
}
