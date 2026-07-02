package sparkline

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestRenderWidthZero(t *testing.T) {
	if got := Render([]float64{1, 2, 3}, 0); got != "" {
		t.Fatalf("Render width 0 = %q, want empty", got)
	}
}

func TestRenderEmptyIsBaseline(t *testing.T) {
	got := Render(nil, 5)
	if want := strings.Repeat("▁", 5); got != want {
		t.Fatalf("Render empty = %q, want %q", got, want)
	}
}

func TestRenderWidthInRunes(t *testing.T) {
	got := Render([]float64{1, 5, 2, 8, 3, 7, 4}, 5)
	if n := utf8.RuneCountInString(got); n != 5 {
		t.Fatalf("rune count = %d, want 5", n)
	}
}

func TestRenderFlatSeriesIsMidLine(t *testing.T) {
	got := Render([]float64{3, 3, 3}, 3)
	if want := strings.Repeat(string(levels[len(levels)/2]), 3); got != want {
		t.Fatalf("flat render = %q, want %q", got, want)
	}
}

func TestRenderRampLowToHigh(t *testing.T) {
	got := []rune(Render([]float64{0, 1, 2, 3, 4, 5, 6, 7}, 8))
	if got[0] != levels[0] {
		t.Errorf("first bar = %q, want lowest %q", got[0], levels[0])
	}
	if got[len(got)-1] != levels[len(levels)-1] {
		t.Errorf("last bar = %q, want highest %q", got[len(got)-1], levels[len(levels)-1])
	}
}

func TestRenderRightAlignsWhenFewerThanWidth(t *testing.T) {
	got := []rune(Render([]float64{5, 6}, 5))
	for i := 0; i < 3; i++ {
		if got[i] != ' ' {
			t.Errorf("rune %d = %q, want leading space", i, got[i])
		}
	}
	if got[4] == ' ' {
		t.Errorf("newest sample must sit at the right edge, got space")
	}
}

func TestRenderTakesTailWhenMoreThanWidth(t *testing.T) {
	// 0..9, width 3 -> last three samples 7,8,9 -> all should be near the top.
	got := []rune(Render([]float64{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}, 3))
	if len(got) != 3 {
		t.Fatalf("len = %d, want 3", len(got))
	}
	// min within the tail is 7, max 9 -> first bar lowest, last highest.
	if got[0] != levels[0] || got[2] != levels[len(levels)-1] {
		t.Errorf("tail render = %q, want ramp over last 3 samples", string(got))
	}
}

func TestRenderOnlyBlockRunes(t *testing.T) {
	got := Render([]float64{1, 9, 3, 7, 5}, 5)
	set := map[rune]bool{}
	for _, r := range levels {
		set[r] = true
	}
	for _, r := range got {
		if r != ' ' && !set[r] {
			t.Errorf("unexpected rune %q in %q", r, got)
		}
	}
}
