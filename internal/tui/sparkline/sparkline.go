package sparkline

import "strings"

// levels are the eight block-unicode bar glyphs, lowest to highest.
var levels = []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

// Render draws the last `width` samples as a block-unicode line of exactly
// `width` runes. Fewer samples than width are right-aligned with leading spaces
// (newest at the right). With no samples it returns a calm baseline of low bars;
// width < 1 returns "". A flat series (min == max) renders as a constant
// mid-level line rather than misleading full or empty bars.
func Render(values []float64, width int) string {
	if width < 1 {
		return ""
	}
	if len(values) > width {
		values = values[len(values)-width:]
	}
	if len(values) == 0 {
		return strings.Repeat(string(levels[0]), width)
	}

	min, max := values[0], values[0]
	for _, v := range values[1:] {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}

	var b strings.Builder
	for i := 0; i < width-len(values); i++ {
		b.WriteRune(' ') // left-pad so newest sits at the right edge
	}
	span := max - min
	for _, v := range values {
		var idx int
		if span == 0 {
			idx = len(levels) / 2
		} else {
			idx = int((v - min) / span * float64(len(levels)-1))
			if idx < 0 {
				idx = 0
			}
			if idx > len(levels)-1 {
				idx = len(levels) - 1
			}
		}
		b.WriteRune(levels[idx])
	}
	return b.String()
}
