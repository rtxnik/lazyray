// Package fsutil provides filesystem helpers shared across lazyray. It depends
// only on the standard library so any package (config, core, lifecycle, ...) can
// import it without creating an import cycle.
package fsutil

import (
	"os"
	"path/filepath"
)

// WriteFile atomically writes data to path with the given permissions.
//
// It writes to a temporary file in the same directory, fsyncs it, then renames
// it over the destination. A crash, SIGKILL, or full disk mid-write therefore
// leaves the previous contents of path intact — never a truncated or empty
// file. The temp file is created with perm directly (not the 0600 default), and
// the directory entry is fsynced so the rename survives a power loss where the
// platform supports it.
//
// This is the single atomic-write primitive for the project. Callers that
// persist user config or runtime state MUST use it instead of os.WriteFile.
func WriteFile(path string, data []byte, perm os.FileMode) (err error) {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	// On any error before a successful rename, drop the temp file. After a
	// successful rename tmpName no longer exists, so Remove is a harmless no-op.
	defer func() {
		if err != nil {
			_ = tmp.Close()
			_ = os.Remove(tmpName)
		}
	}()

	if err = tmp.Chmod(perm); err != nil {
		return err
	}
	if _, err = tmp.Write(data); err != nil {
		return err
	}
	if err = tmp.Sync(); err != nil {
		return err
	}
	if err = tmp.Close(); err != nil {
		return err
	}
	if err = os.Rename(tmpName, path); err != nil {
		return err
	}
	// Best-effort: fsync the directory so the rename itself is durable. Not all
	// platforms/filesystems support directory fsync, so ignore any error.
	if d, derr := os.Open(dir); derr == nil {
		_ = d.Sync()
		_ = d.Close()
	}
	return nil
}
