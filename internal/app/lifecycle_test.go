package app

import (
	"errors"
	"testing"

	"github.com/rtxnik/lazyray/internal/config"
	"github.com/rtxnik/lazyray/internal/lifecycle"
	"github.com/rtxnik/lazyray/internal/platform"
)

func TestConnect_DelegatesArgs(t *testing.T) {
	var got []string
	svc := &Service{spawnDetached: func(extra []string) error { got = extra; return nil }}
	if err := svc.Connect([]string{"--owner", "tui"}); err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	if len(got) != 2 || got[0] != "--owner" || got[1] != "tui" {
		t.Errorf("spawn args = %v, want [--owner tui]", got)
	}
}

func TestConnect_ErrorPropagates(t *testing.T) {
	sentinel := errors.New("spawn boom")
	svc := &Service{spawnDetached: func([]string) error { return sentinel }}
	if err := svc.Connect(nil); !errors.Is(err, sentinel) {
		t.Fatalf("err = %v, want sentinel", err)
	}
}

func TestWriteActiveConfig_DelegatesRaw(t *testing.T) {
	sentinel := errors.New("gen boom")
	var gotP *config.Profile
	var gotS *config.Settings
	svc := &Service{writeXrayConfig: func(p *config.Profile, s *config.Settings) error {
		gotP, gotS = p, s
		return sentinel
	}}
	p := &config.Profile{Name: "x"}
	s := &config.Settings{}
	if err := svc.WriteActiveConfig(p, s); !errors.Is(err, sentinel) {
		t.Fatalf("err = %v, want raw sentinel", err)
	}
	if gotP != p || gotS != s {
		t.Error("WriteActiveConfig must pass profile+settings through unchanged")
	}
}

func TestDisconnect_SignalsPidThenReconciles(t *testing.T) {
	var signalledPID int
	reconcileCalled := false
	alive := true
	svc := &Service{
		currentProxy:     func() platform.SystemProxy { return nil },
		readState:        func() (*lifecycle.State, error) { return &lifecycle.State{SupervisorPID: 4321}, nil },
		signalSupervisor: func(pid int) error { signalledPID = pid; alive = false; return nil },
		supervisorAlive:  func() bool { return alive },
		reconcile:        func(platform.SystemProxy) error { reconcileCalled = true; return nil },
	}
	if err := svc.Disconnect(); err != nil {
		t.Fatalf("Disconnect() error = %v", err)
	}
	if signalledPID != 4321 {
		t.Errorf("signalled pid = %d, want 4321", signalledPID)
	}
	if !reconcileCalled {
		t.Error("reconcile must be called")
	}
}

func TestDisconnect_NoStateSkipsSignal(t *testing.T) {
	signalCalled := false
	reconcileCalled := false
	svc := &Service{
		currentProxy:     func() platform.SystemProxy { return nil },
		readState:        func() (*lifecycle.State, error) { return nil, nil },
		signalSupervisor: func(int) error { signalCalled = true; return nil },
		supervisorAlive:  func() bool { return false },
		reconcile:        func(platform.SystemProxy) error { reconcileCalled = true; return nil },
	}
	if err := svc.Disconnect(); err != nil {
		t.Fatalf("Disconnect() error = %v", err)
	}
	if signalCalled {
		t.Error("no signal when state is nil")
	}
	if !reconcileCalled {
		t.Error("reconcile still runs when state is nil")
	}
}

func TestDisconnect_PidZeroSkipsSignal(t *testing.T) {
	signalCalled := false
	svc := &Service{
		currentProxy:     func() platform.SystemProxy { return nil },
		readState:        func() (*lifecycle.State, error) { return &lifecycle.State{SupervisorPID: 0}, nil },
		signalSupervisor: func(int) error { signalCalled = true; return nil },
		supervisorAlive:  func() bool { return false },
		reconcile:        func(platform.SystemProxy) error { return nil },
	}
	if err := svc.Disconnect(); err != nil {
		t.Fatalf("Disconnect() error = %v", err)
	}
	if signalCalled {
		t.Error("no signal when SupervisorPID <= 0")
	}
}

func TestDisconnect_PollsWhileAlive(t *testing.T) {
	calls := 0
	svc := &Service{
		currentProxy:     func() platform.SystemProxy { return nil },
		readState:        func() (*lifecycle.State, error) { return &lifecycle.State{SupervisorPID: 10}, nil },
		signalSupervisor: func(int) error { return nil },
		supervisorAlive:  func() bool { calls++; return calls < 3 }, // alive twice, then down
		reconcile:        func(platform.SystemProxy) error { return nil },
	}
	if err := svc.Disconnect(); err != nil {
		t.Fatalf("Disconnect() error = %v", err)
	}
	if calls < 3 {
		t.Errorf("supervisorAlive polled %d times, want >= 3", calls)
	}
}

func TestDisconnect_ReconcileErrorPropagates(t *testing.T) {
	sentinel := errors.New("reconcile boom")
	svc := &Service{
		currentProxy:     func() platform.SystemProxy { return nil },
		readState:        func() (*lifecycle.State, error) { return nil, nil },
		signalSupervisor: func(int) error { return nil },
		supervisorAlive:  func() bool { return false },
		reconcile:        func(platform.SystemProxy) error { return sentinel },
	}
	if err := svc.Disconnect(); !errors.Is(err, sentinel) {
		t.Fatalf("err = %v, want sentinel", err)
	}
}
