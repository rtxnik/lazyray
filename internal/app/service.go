// Package app is the application-service layer. It owns the multi-step business
// flows that were previously hand-sequenced and duplicated across the CLI
// (cmd/*) and the TUI (internal/tui). It sits above core/config/lifecycle and is
// never imported by them (cycle-safe). Dependencies are injected as plain
// function values (mirroring internal/doctor's Env) so flows are testable
// without a network, a spawned xray, or disk I/O.
package app

import (
	"github.com/rtxnik/lazyray/internal/config"
	"github.com/rtxnik/lazyray/internal/core"
	"github.com/rtxnik/lazyray/internal/lifecycle"
	"github.com/rtxnik/lazyray/internal/platform"
)

// Service is the application-service facade. Its seams are function values so
// tests can substitute fakes; NewService wires the real host implementations.
type Service struct {
	// Import flows.
	saveServers        func(*config.ServersConfig) error
	importSubscription func(subURL, subName string, servers *config.ServersConfig) (added, updated int, err error)

	// Connect/Disconnect flows.
	spawnDetached    func(extraArgs []string) error
	writeXrayConfig  func(p *config.Profile, s *config.Settings) error
	readState        func() (*lifecycle.State, error)
	supervisorAlive  func() bool
	signalSupervisor func(pid int) error
	reconcile        func(sp platform.SystemProxy) error
	currentProxy     func() platform.SystemProxy
}

// NewService returns a Service bound to the real config/core/lifecycle/platform
// implementations.
func NewService() *Service {
	return &Service{
		saveServers:        config.SaveServers,
		importSubscription: core.ImportSubscription,

		spawnDetached:    lifecycle.SpawnDetached,
		writeXrayConfig:  core.WriteXrayConfig,
		readState:        lifecycle.ReadState,
		supervisorAlive:  lifecycle.SupervisorAlive,
		signalSupervisor: lifecycle.SignalSupervisor,
		reconcile:        lifecycle.Reconcile,
		currentProxy:     platform.CurrentSystemProxy,
	}
}
