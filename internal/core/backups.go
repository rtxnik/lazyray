package core

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/rtxnik/lazyray/internal/config"
)

// enginePrefixes are the basename stems the xray-update path writes as
// "<stem>.<timestamp>.bak" into BackupDir (see ApplyUpdate). Longest first so
// "xray.exe" wins over "xray". PruneEngineBackups operates ONLY on these — never
// on lazyray-backup-*.tar.gz[.enc] config archives.
var enginePrefixes = []string{"xray.exe", "geoip.dat", "geosite.dat", "xray"}

// PruneEngineBackups keeps the maxSets newest whole engine-backup sets (grouped
// by the shared <timestamp> token) in config.BackupDir() and removes older sets.
// Config-backup archives (lazyray-backup-*) are never counted or deleted.
// maxSets <= 0 defaults to 5.
func PruneEngineBackups(maxSets int) {
	if maxSets <= 0 {
		maxSets = 5
	}
	dir := config.BackupDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	sets := map[string][]string{}
	var stamps []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		ts, ok := engineBackupTS(e.Name())
		if !ok {
			continue
		}
		if _, seen := sets[ts]; !seen {
			stamps = append(stamps, ts)
		}
		sets[ts] = append(sets[ts], e.Name())
	}
	if len(stamps) <= maxSets {
		return
	}
	sort.Sort(sort.Reverse(sort.StringSlice(stamps))) // newest first (sortable ts)
	for _, ts := range stamps[maxSets:] {
		for _, name := range sets[ts] {
			_ = os.Remove(filepath.Join(dir, name))
		}
	}
}

// engineBackupTS extracts the <ts> from "<stem>.<ts>.bak" for a recognised
// engine stem, or ("", false). The timestamp token contains no ".", which
// rejects config archives and mis-stemmed names.
func engineBackupTS(name string) (string, bool) {
	if !strings.HasSuffix(name, ".bak") {
		return "", false
	}
	for _, stem := range enginePrefixes {
		prefix := stem + "."
		if strings.HasPrefix(name, prefix) {
			ts := strings.TrimSuffix(strings.TrimPrefix(name, prefix), ".bak")
			if ts != "" && !strings.Contains(ts, ".") {
				return ts, true
			}
		}
	}
	return "", false
}
