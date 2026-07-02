package panels

import (
	"strings"
	"testing"

	"github.com/rtxnik/lazyray/internal/core"
)

func joinBadge(s *StatusPanel) string {
	var b strings.Builder
	for _, ln := range s.badgeLines() {
		b.WriteString(ln.text)
		b.WriteString("\n")
	}
	return b.String()
}

func TestStoppedWithErrorTeachesReasonAndDoctor(t *testing.T) {
	s := NewStatusPanel()
	s.Status = &core.XrayStatus{Running: false}
	s.StoppedReason = "start: bind :8080 already in use"
	s.DoctorKey = "h"
	out := joinBadge(&s)
	for _, want := range []string{"exited with error", "last: start: bind :8080 already in use", "press [h] to diagnose"} {
		if !strings.Contains(out, want) {
			t.Errorf("stopped-error badge missing %q\n%s", want, out)
		}
	}
}

func TestStoppedCleanTeachesConnect(t *testing.T) {
	s := NewStatusPanel()
	s.Status = &core.XrayStatus{Running: false}
	s.StoppedReason = ""
	s.ConnectKey = "enter"
	out := joinBadge(&s)
	if !strings.Contains(out, "press [enter] on a profile to connect") {
		t.Errorf("clean-stop badge missing connect hint\n%s", out)
	}
	if strings.Contains(out, "exited with error") {
		t.Errorf("clean stop must not claim an error\n%s", out)
	}
}

func TestRunningBadgeUnchanged(t *testing.T) {
	s := NewStatusPanel()
	s.Status = &core.XrayStatus{Running: true}
	if out := joinBadge(&s); !strings.Contains(out, "Connected") {
		t.Errorf("running badge should show Connected\n%s", out)
	}
}
