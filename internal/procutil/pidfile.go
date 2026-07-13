package procutil

import "github.com/rtxnik/lazyray/internal/fsutil"

// WritePIDFile atomically writes runtime pid-file content with 0600 perms and
// surfaces any error. The single enforcement point for the pid-file write
// contract (atomic temp→fsync→rename via fsutil; not world-readable 0644).
func WritePIDFile(path string, data []byte) error {
	return fsutil.WriteFile(path, data, 0o600)
}
