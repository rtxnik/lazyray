package refdocs

import (
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/rtxnik/lazyray/internal/config"
)

// repoRoot walks up from this test file to the directory containing go.mod.
func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	dir := filepath.Dir(file)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found above test file")
		}
		dir = parent
	}
}

// yamlFieldNames recursively collects yaml tag base names from a struct type,
// descending into nested structs, slices, and pointers.
func yamlFieldNames(t reflect.Type, seen map[reflect.Type]bool, out map[string]bool) {
	for t.Kind() == reflect.Ptr || t.Kind() == reflect.Slice {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct || seen[t] {
		return
	}
	seen[t] = true
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		tag := f.Tag.Get("yaml")
		if tag == "" || tag == "-" {
			continue
		}
		if name := strings.SplitN(tag, ",", 2)[0]; name != "" {
			out[name] = true
		}
		yamlFieldNames(f.Type, seen, out)
	}
}

func TestConfigurationReferenceDocumentsEveryField(t *testing.T) {
	names := map[string]bool{}
	seen := map[reflect.Type]bool{}
	yamlFieldNames(reflect.TypeOf(config.Settings{}), seen, names)
	yamlFieldNames(reflect.TypeOf(config.ServersConfig{}), seen, names)

	data, err := os.ReadFile(filepath.Join(repoRoot(t), "docs", "reference", "configuration.md"))
	if err != nil {
		t.Fatalf("reading configuration.md: %v", err)
	}
	doc := string(data)

	var missing []string
	for name := range names {
		if !strings.Contains(doc, name) {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		t.Errorf("configuration.md is missing %d field(s): %v", len(missing), missing)
	}
}
