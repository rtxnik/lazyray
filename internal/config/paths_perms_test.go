package config

import (
	"os"
	"runtime"
	"testing"
)

func TestEnsureDirs_Perms0700(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX file permissions are not honored on Windows")
	}
	t.Setenv("HOME", t.TempDir())
	if err := EnsureDirs(); err != nil {
		t.Fatalf("EnsureDirs() = %v", err)
	}
	for _, d := range []string{ConfigDir(), DataDir(), LogDir(), BackupDir()} {
		info, err := os.Stat(d)
		if err != nil {
			t.Fatalf("stat %s: %v", d, err)
		}
		if info.Mode().Perm() != 0700 {
			t.Errorf("%s perm = %v, want drwx------", d, info.Mode().Perm())
		}
	}
}
