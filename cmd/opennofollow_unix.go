//go:build !windows

package cmd

import (
	"os"
	"syscall"
)

// openNoFollow opens path read-only, refusing to traverse a terminal symlink
// (atomic O_NOFOLLOW — no check/use race).
func openNoFollow(path string) (*os.File, error) {
	return os.OpenFile(path, os.O_RDONLY|syscall.O_NOFOLLOW, 0)
}
