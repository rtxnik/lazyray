package platform

import (
	"strings"
	"testing"
)

func TestPSQuote(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"plain", "lazyray", "'lazyray'"},
		{"internal quote doubled", "O'Brien", "'O''Brien'"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := psQuote(tt.in); got != tt.want {
				t.Errorf("psQuote(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestPSQuoteNeutralizesInjection(t *testing.T) {
	payload := `'); Remove-Item C:\ -Recurse; ('`
	got := psQuote(payload)

	if !strings.HasPrefix(got, "'") || !strings.HasSuffix(got, "'") {
		t.Fatalf("psQuote(%q) = %q, must start and end with a single quote", payload, got)
	}

	// Every internal single quote must be doubled: stripping the outer quotes,
	// the inner content's single quotes must all come in '' pairs, so the count
	// is even and there is no lone quote that could terminate the literal early.
	inner := got[1 : len(got)-1]
	if strings.Count(inner, "'")%2 != 0 {
		t.Errorf("psQuote(%q) = %q, inner content has an unbalanced single quote", payload, got)
	}

	if want := "'" + strings.ReplaceAll(payload, "'", "''") + "'"; got != want {
		t.Errorf("psQuote(%q) = %q, want %q", payload, got, want)
	}
}
