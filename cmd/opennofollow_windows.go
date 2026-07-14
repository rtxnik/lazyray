//go:build windows

package cmd

import "os"

// openNoFollow opens path read-only. Windows lacks O_NOFOLLOW; fall back to an
// Lstat guard (the POSIX symlink threat model does not apply the same way).
func openNoFollow(path string) (*os.File, error) {
	fi, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		return nil, os.ErrInvalid
	}
	return os.Open(path)
}
