package cmd

import (
	"errors"
	"testing"
	"time"
)

func TestRankProfiles_Order(t *testing.T) {
	results := []profileLatency{
		{name: "fail", err: errors.New("down")},
		{name: "skip", skipped: true},
		{name: "slow", latency: 200 * time.Millisecond},
		{name: "fast", latency: 50 * time.Millisecond},
	}
	rankProfiles(results)
	got := []string{results[0].name, results[1].name, results[2].name, results[3].name}
	want := []string{"fast", "slow", "skip", "fail"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("rank = %v, want %v", got, want)
		}
	}
}
