package theme

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// All color in the TUI must flow from the theme package. A stray
// lipgloss.Color("#...") anywhere else re-introduces the theme-bypass bug
// E2a closed, so it is a hard failure. Test files are exempt (they assert on
// concrete hex/ANSI values); the theme package itself is the one legal home.
//
// Guard G4 (no hardcoded colors outside the theme package). Documented in
// docs/ARCHITECTURE.md (Invariants & Guards). Keep that section and this test in sync.
func TestNoHardcodedColorsOutsideThemePackage(t *testing.T) {
	const needle = `lipgloss.Color("` + "#"
	var offenders []string
	err := filepath.Walk("..", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if info.Name() == "theme" { // skip this package
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		data, rerr := os.ReadFile(path)
		if rerr != nil {
			return rerr
		}
		if strings.Contains(string(data), needle) {
			offenders = append(offenders, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk: %v", err)
	}
	if len(offenders) > 0 {
		t.Fatalf("hardcoded colors outside theme package:\n%s", strings.Join(offenders, "\n"))
	}
}
