// cmd/exit_test.go
package cmd

import (
	"errors"
	"testing"
)

func TestExitCodeFor(t *testing.T) {
	cases := []struct {
		err  error
		want int
	}{
		{nil, 0},
		{errors.New("boom"), 1},
		{&ExitError{Code: ExitNotRunning, Err: errors.New("x")}, ExitNotRunning},
		{&ExitError{Code: ExitConfig, Err: errors.New("x")}, ExitConfig},
	}
	for _, c := range cases {
		if got := exitCodeFor(c.err); got != c.want {
			t.Errorf("exitCodeFor(%v) = %d, want %d", c.err, got, c.want)
		}
	}
}
