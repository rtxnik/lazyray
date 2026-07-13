package core

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// extractFromTarGz extracts the entry whose basename case-insensitively equals
// targetName from the gzip-compressed tar at tarGzPath, writing it to dest. The
// dest file is only created once the target entry is found, so a miss leaves the
// filesystem untouched. Mirrors extractFromZip in updater.go for the .zip path.
func extractFromTarGz(tarGzPath, targetName, dest string) error {
	f, err := os.Open(tarGzPath)
	if err != nil {
		return err
	}
	defer f.Close()

	gr, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gr.Close()

	tr := tar.NewReader(gr)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if hdr.Typeflag != tar.TypeReg {
			continue
		}
		if !strings.EqualFold(filepath.Base(hdr.Name), targetName) {
			continue
		}
		out, err := os.Create(dest)
		if err != nil {
			return err
		}
		defer out.Close()
		// Bound the copy to the declared header size to avoid decompression bombs.
		if _, err := io.CopyN(out, tr, hdr.Size); err != nil && err != io.EOF {
			return err
		}
		if err := out.Sync(); err != nil {
			return err
		}
		return nil
	}

	return fmt.Errorf("%s not found in archive", targetName)
}
