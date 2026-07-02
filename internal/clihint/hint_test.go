package clihint

import (
	"bytes"
	"errors"
	"fmt"
	"testing"
)

func TestErrorMessageAndUnwrap(t *testing.T) {
	sentinel := errors.New("disk full")
	e := Errorf("free some space", "writing config: %w", sentinel)
	if e.Error() != "writing config: disk full" {
		t.Errorf("Error() = %q", e.Error())
	}
	if e.Msg != "writing config: disk full" {
		t.Errorf("Msg = %q", e.Msg)
	}
	if e.Hint != "free some space" {
		t.Errorf("Hint = %q", e.Hint)
	}
	if !errors.Is(e, sentinel) {
		t.Errorf("errors.Is(e, sentinel) = false, want true (%%w not honored)")
	}
}

func TestRenderHinted(t *testing.T) {
	var buf bytes.Buffer
	Render(&buf, Errorf("import a profile with 'lzr import <url>'", "no profiles configured"))
	want := "Error: no profiles configured\n  → try: import a profile with 'lzr import <url>'\n"
	if buf.String() != want {
		t.Errorf("Render = %q, want %q", buf.String(), want)
	}
}

func TestRenderNoHint(t *testing.T) {
	var buf bytes.Buffer
	Render(&buf, &Error{Msg: "something broke"})
	if buf.String() != "Error: something broke\n" {
		t.Errorf("Render = %q", buf.String())
	}
}

func TestRenderPlainError(t *testing.T) {
	var buf bytes.Buffer
	Render(&buf, fmt.Errorf("raw failure"))
	if buf.String() != "Error: raw failure\n" {
		t.Errorf("Render = %q", buf.String())
	}
}

func TestRenderFindsWrappedHint(t *testing.T) {
	wrapped := fmt.Errorf("context: %w", Errorf("try harder", "inner failure"))
	var buf bytes.Buffer
	Render(&buf, wrapped)
	want := "Error: inner failure\n  → try: try harder\n"
	if buf.String() != want {
		t.Errorf("Render = %q, want %q", buf.String(), want)
	}
}

func TestRenderNil(t *testing.T) {
	var buf bytes.Buffer
	Render(&buf, nil)
	if buf.Len() != 0 {
		t.Errorf("Render(nil) wrote %q, want empty", buf.String())
	}
}
