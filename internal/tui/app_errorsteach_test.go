package tui

import (
	"errors"
	"strings"
	"testing"

	"github.com/rtxnik/lazyray/internal/clihint"
	"github.com/rtxnik/lazyray/internal/tui/commands"
	"github.com/rtxnik/lazyray/internal/tui/notify"
)

func newErrorsTeachApp() *App {
	return &App{
		keys:    commands.DefaultKeyMap(),
		notices: notify.New(10),
		width:   80,
		height:  24,
	}
}

func TestTeachHintBySource(t *testing.T) {
	cases := map[string]string{
		"":             "press [h] to diagnose",
		"lifecycle":    "press [h] to diagnose",
		"profile":      "verify the profile, then [h] to diagnose",
		"subscription": "check the subscription URL, then [h]",
		"update":       "see logs, or retry",
	}
	for src, want := range cases {
		if got := teachHint(src, "h"); got != want {
			t.Errorf("teachHint(%q) = %q, want %q", src, got, want)
		}
	}
}

func TestSetErrorGetsFallbackHint(t *testing.T) {
	a := newErrorsTeachApp()
	a.setError(errors.New("connect failed"))
	if a.activeTail == nil || a.activeTail.Hint != "press [h] to diagnose" {
		t.Fatalf("plain error should get fallback hint, got %+v", a.activeTail)
	}
}

func TestClihintHintWinsOverFallback(t *testing.T) {
	a := newErrorsTeachApp()
	a.setError(clihint.Errorf("import a profile with 'lzr import <url>'", "no profiles"))
	if a.activeTail == nil || a.activeTail.Hint != "import a profile with 'lzr import <url>'" {
		t.Fatalf("clihint hint should win, got %+v", a.activeTail)
	}
}

func TestStatusBarTailRendersInlineHint(t *testing.T) {
	a := newErrorsTeachApp()
	a.notify(notify.Notice{Severity: notify.Error, Message: "connect failed", Hint: "check the host, then [h]"})
	bar := a.renderStatusBar()
	if !strings.Contains(bar, "→ try: check the host, then [h]") {
		t.Errorf("error tail missing inline hint line:\n%s", bar)
	}
	if a.tailRows() != 2 {
		t.Errorf("tailRows = %d, want 2 for hinted error", a.tailRows())
	}
}

func TestInfoTailHasNoHintRow(t *testing.T) {
	a := newErrorsTeachApp()
	a.notify(notify.Notice{Severity: notify.Info, Message: "proxy started"})
	if a.tailRows() != 1 {
		t.Errorf("tailRows = %d, want 1 for info tail", a.tailRows())
	}
}

func TestCalcLayoutReservesHintRow(t *testing.T) {
	a := newErrorsTeachApp()
	a.notify(notify.Notice{Severity: notify.Info, Message: "x"}) // 1-row tail
	_, _, base, _, _ := a.calcLayout()
	a.notify(notify.Notice{Severity: notify.Error, Message: "boom", Hint: "press [h] to diagnose"}) // 2-row tail
	_, _, withHint, _, _ := a.calcLayout()
	if withHint != base-1 {
		t.Errorf("hinted-error layout fullHeight = %d, want %d (one row reserved)", withHint, base-1)
	}
}
