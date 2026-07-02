package app

import (
	"time"

	"github.com/rtxnik/lazyray/internal/config"
)

// disconnectGrace/disconnectPoll reproduce the TUI's disconnectSession timing
// verbatim (7s graceful window, 50ms poll) so teardown behaviour is unchanged.
const (
	disconnectGrace = 7 * time.Second
	disconnectPoll  = 50 * time.Millisecond
)

// Connect brings the resident supervisor up by re-execing the detached daemon
// with the caller's owner/proxy arguments. It is the single home for the spawn
// step both shells performed inline; each shell keeps its own surrounding
// reconcile/settings/wait/output because those differ per shell.
func (s *Service) Connect(extraArgs []string) error {
	return s.spawnDetached(extraArgs)
}

// WriteActiveConfig regenerates and writes the xray config for a profile that is
// becoming (or staying) active. It returns the raw error so each shell keeps its
// own wrapping; the TUI called core.WriteXrayConfig inline at three activate
// sites (profile switch, edit, routing change).
func (s *Service) WriteActiveConfig(profile *config.Profile, settings *config.Settings) error {
	return s.writeXrayConfig(profile, settings)
}

// Disconnect tears the session down gracefully: signal the supervisor by PID,
// wait up to disconnectGrace for it to release the lock, then reconcile any
// dangling routing/state. It reproduces the TUI's disconnectSession exactly;
// hard-kill escalation stays a CLI-shell concern and is intentionally NOT
// performed here.
func (s *Service) Disconnect() error {
	sp := s.currentProxy()
	if st, _ := s.readState(); st != nil && st.SupervisorPID > 0 {
		_ = s.signalSupervisor(st.SupervisorPID)
	}
	deadline := time.Now().Add(disconnectGrace)
	for time.Now().Before(deadline) && s.supervisorAlive() {
		time.Sleep(disconnectPoll)
	}
	return s.reconcile(sp)
}
