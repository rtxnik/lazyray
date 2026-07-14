// Package execsafe resolves external helper binaries to trustworthy absolute
// paths, closing a PATH-hijack gap without hard-coding platform locations.
package execsafe

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// SecureLookPath resolves name to an absolute path via exec.LookPath (which, on
// modern Go, already refuses a relative/CWD match), canonicalises it with
// EvalSymlinks, and rejects the result if the real binary or any ancestor
// directory up to the root is world-writable WITHOUT the sticky bit.
// World-writability is the PATH-hijack lever (a shared dir any local user can
// write, planted ahead of the system dirs). A sticky world-writable directory
// (the /tmp pattern, mode 01777) is treated as acceptable — a deliberate
// tradeoff: the sticky bit restricts rename/delete to entry owners (not new-file
// creation), but such directories are not normally ahead of the system dirs in
// PATH. Group-writable system prefixes (e.g. an admin-group Homebrew bin) are
// never rejected, and ownership is not checked. On Windows the POSIX model does
// not apply and the LookPath result is returned unchanged.
func SecureLookPath(name string) (string, error) {
	p, err := exec.LookPath(name)
	if err != nil {
		return "", err
	}
	if runtime.GOOS == "windows" {
		return p, nil
	}
	real, err := filepath.EvalSymlinks(p)
	if err != nil {
		return "", err
	}
	// Check BOTH the original lookup location and the resolved real path: a
	// symlink planted in a world-writable PATH directory can point at a binary in
	// a tight directory, so resolving first and checking only the target would
	// miss the world-writable directory where the interposition happened. Start
	// the original walk at the parent directory — the leaf p may itself be a
	// symlink whose own mode is meaningless (always lrwxrwxrwx).
	if err := refuseWorldWritable(name, filepath.Dir(p)); err != nil {
		return "", err
	}
	if err := refuseWorldWritable(name, real); err != nil {
		return "", err
	}
	return real, nil
}

// refuseWorldWritable returns an error if start, or any ancestor directory up to
// the filesystem root, is a real (non-symlink) path that is world-writable
// without the sticky bit. A symlinked component's own mode is meaningless
// (always lrwxrwxrwx) — on usr-merge distributions /bin, /sbin, and /lib are
// symlinks into /usr, so judging them by Lstat would falsely reject a root-owned
// binary. Such components are skipped here; their resolved targets are validated
// by the caller's separate walk over the fully symlink-resolved real path.
func refuseWorldWritable(name, start string) error {
	for cur := start; ; {
		fi, err := os.Lstat(cur)
		if err != nil {
			return err
		}
		if fi.Mode()&os.ModeSymlink == 0 && fi.Mode().Perm()&0o002 != 0 && fi.Mode()&os.ModeSticky == 0 {
			return fmt.Errorf("%q resolves through a world-writable path (%s)", name, cur)
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			return nil
		}
		cur = parent
	}
}
