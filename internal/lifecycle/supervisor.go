// internal/lifecycle/supervisor.go
package lifecycle

import (
	"context"
	"os"
	"os/exec"
	"os/signal"
	"time"

	"github.com/rtxnik/lazyray/internal/config"
	"github.com/rtxnik/lazyray/internal/core"
	"github.com/rtxnik/lazyray/internal/platform"
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
	superviseXray func(ctx context.Context, pid int) error
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

	_ = s.superviseXray(sigCtx, pid)

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
// enabled, it respawns up to maxRetries. On ctx cancellation it terminates the
// child itself (it owns s.cmd), using the Wait completion as the death oracle so
// the kill and the reaper never operate on the same pid from two goroutines.
func (s *Supervisor) superviseXrayReal(ctx context.Context, pid int) error {
	const maxRetries = 3
	retries := 0
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
			s.terminateChild(pid, exited)
			return nil
		case <-exited:
			if !s.Settings.Xray.AutoRestart || retries >= maxRetries {
				if s.Settings.Notifications.Enabled {
					_ = platform.Current().Notify("lazyray", "xray exited and was not restarted")
				}
				return nil
			}
			retries++
			newPID, err := s.startXray()
			if err != nil {
				if s.Settings.Notifications.Enabled {
					_ = platform.Current().Notify("lazyray", "xray crashed and could not be restarted")
				}
				return nil
			}
			if st, _ := ReadState(); st != nil {
				st.XrayPID = newPID
				_ = WriteState(st)
			}
			pid = newPID
		}
	}
}
