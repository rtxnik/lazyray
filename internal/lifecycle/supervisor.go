// internal/lifecycle/supervisor.go
package lifecycle

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"time"

	"github.com/rtxnik/lazyray/internal/config"
	"github.com/rtxnik/lazyray/internal/core"
	"github.com/rtxnik/lazyray/internal/platform"
)

// Supervise-loop tuning (package vars so tests can override them deterministically).
var (
	// healthyRunThreshold: an xray incarnation that survives this long is treated
	// as healthy, so a later crash starts a fresh restart budget instead of
	// counting toward the crash-loop limit.
	healthyRunThreshold = 30 * time.Second
	// restartBackoff paces respawns after a rapid (sub-threshold) crash so a
	// broken config cannot busy-loop. Not applied after a healthy run.
	restartBackoff = 2 * time.Second
	// timeNow is the clock seam (overridden in tests for deterministic uptime).
	timeNow = time.Now
	// writeState persists the restarted-pid update. It is a seam (var, not a
	// direct WriteState call) so a test can force a persistence failure and assert
	// the in-memory State remains the load-bearing source of truth. Only the
	// restart tail routes through it; the initial Run write stays a direct
	// WriteState (its failure is already a fatal StagedError{state}).
	writeState = WriteState
)

// Supervisor owns one xray session: process, routing, watchdog, and teardown.
type Supervisor struct {
	Owner    Owner
	Profile  *config.Profile
	Settings *config.Settings
	Proxy    ProxyMode

	cmd *exec.Cmd

	// Seams (default to real implementations in Run when nil).
	sysproxy      platform.SystemProxy
	startXray     func() (pid int, err error)
	superviseXray func(ctx context.Context, st *State) error
	killXray      func(pid int) error
}

// Run blocks until the context is canceled or a termination signal arrives,
// then performs an ordered teardown. It is the body of `lzr __run`.
func (s *Supervisor) Run(ctx context.Context) error {
	if s.sysproxy == nil {
		s.sysproxy = platform.CurrentSystemProxy()
	}
	if s.startXray == nil {
		s.startXray = s.startXrayReal
	}
	if s.superviseXray == nil {
		s.superviseXray = s.superviseXrayReal
	}
	if s.killXray == nil {
		s.killXray = DefaultKill
	}

	lock, err := AcquireLock()
	if err != nil {
		return &StagedError{Stage: "lock", Err: err} // ErrLocked when another supervisor is live
	}
	defer func() { _ = lock.Release() }()

	// Clean any leftovers from a crashed predecessor (we hold the lock now, so
	// the previous supervisor is definitely dead). guardedKill verifies the PID
	// still belongs to our xray before signalling, guarding against reuse.
	if old, _ := ReadState(); old != nil {
		_ = Teardown(old, s.sysproxy, guardedKill)
	}

	routing, err := ApplyRouting(s.sysproxy, s.Settings, s.Proxy)
	if err != nil {
		return &StagedError{Stage: "routing", Err: err}
	}

	pid, err := s.startXray()
	if err != nil {
		_ = RevertRouting(s.sysproxy, routing)
		return &StagedError{Stage: "start", Err: err}
	}

	st := &State{
		Owner:         s.Owner,
		SupervisorPID: os.Getpid(),
		XrayPID:       pid,
		StartedAt:     time.Now().UTC(),
		SocksPort:     s.Settings.Local.SocksPort,
		HTTPPort:      s.Settings.Local.HTTPPort,
		Routing:       routing,
		ActiveProfile: func() string {
			if s.Profile != nil {
				return s.Profile.Name
			}
			return ""
		}(),
	}
	if err := WriteState(st); err != nil {
		_ = s.killXray(pid)
		_ = RevertRouting(s.sysproxy, routing)
		return &StagedError{Stage: "state", Err: err}
	}

	// Startup succeeded — clear any stale failure record from a previous attempt.
	_ = ClearStartupError()

	sigCtx, stop := signal.NotifyContext(ctx, terminateSignals()...)
	defer stop()

	_ = s.superviseXray(sigCtx, st)

	// Teardown: revert routing → kill xray group → remove state.
	return Teardown(st, s.sysproxy, s.killXray)
}

// startXrayReal regenerates the config and launches xray in its own process
// group (so GracefulKill(-pid) reaps grandchildren).
func (s *Supervisor) startXrayReal() (int, error) {
	if err := core.WriteXrayConfig(s.Profile, s.Settings); err != nil {
		return 0, err
	}
	cmd := exec.Command(config.XrayBinaryPath(), "run", "-c", config.XrayConfigPath())
	cmd.SysProcAttr = xrayProcAttr()
	if err := cmd.Start(); err != nil {
		return 0, err
	}
	s.cmd = cmd
	return cmd.Process.Pid, nil
}

// superviseXrayReal waits on xray; if it exits unexpectedly and auto-restart is
// enabled, it respawns, keeping st.XrayPID (the pid Teardown and Reconcile use)
// in sync. Only rapid (sub-threshold) crashes count toward maxRetries and are
// paced by restartBackoff; a run that survives healthyRunThreshold resets the
// streak and restarts for free. On ctx cancellation it terminates the child
// itself (it owns s.cmd), using Wait completion as the death oracle.
func (s *Supervisor) superviseXrayReal(ctx context.Context, st *State) error {
	const maxRetries = 3
	retries := 0
	startedAt := timeNow()
	for {
		cmd := s.cmd
		exited := make(chan struct{})
		go func() {
			if cmd != nil {
				_ = cmd.Wait() // sole reaper; no zombies
			}
			close(exited)
		}()

		select {
		case <-ctx.Done():
			s.terminateChild(st.XrayPID, exited)
			return nil
		case <-exited:
			if ctx.Err() != nil {
				return nil // shutdown won the ctx.Done/exited race — never respawn
			}
			uptime := timeNow().Sub(startedAt) // read the clock once per exit
			if uptime >= healthyRunThreshold {
				retries = 0 // healthy run that crashed → fresh restart budget
			}
			if !s.Settings.Xray.AutoRestart || retries >= maxRetries {
				if s.Settings.Notifications.Enabled {
					_ = platform.Current().Notify("lazyray", "xray exited and was not restarted")
				}
				return nil
			}
			if uptime < healthyRunThreshold { // rapid crash → pace respawn and count it
				timer := time.NewTimer(restartBackoff)
				select {
				case <-ctx.Done():
					timer.Stop()
					return nil // no live child between exit and respawn
				case <-timer.C:
				}
				retries++ // only rapid crashes accumulate; a healthy restart is free
			}
			newPID, err := s.startXray()
			if err != nil {
				if s.Settings.Notifications.Enabled {
					_ = platform.Current().Notify("lazyray", "xray crashed and could not be restarted")
				}
				return nil
			}
			st.XrayPID = newPID // in-memory: Teardown (Run:102) sees the current pid
			if werr := writeState(st); werr != nil {
				// In-memory st already holds newPID, so graceful Teardown stays
				// correct and Reconcile's guardedKill (IsOurXray) cannot wrong-kill a
				// stale on-disk pid. Surface the divergence instead of swallowing it,
				// but keep supervising — a transient persistence failure must not drop
				// a working proxy. Residual (WriteState fail + hard supervisor death +
				// a later Reconcile) is at worst an orphaned xray, never a wrong-kill.
				if lf := openSupervisorLog(); lf != nil {
					_, _ = fmt.Fprintf(lf, "%s supervisor: state persistence failed after xray restart (pid %d); on-disk record is stale: %v\n",
						timeNow().UTC().Format(time.RFC3339), newPID, werr)
					_ = lf.Close()
				}
			}
			startedAt = timeNow()
		}
	}
}
