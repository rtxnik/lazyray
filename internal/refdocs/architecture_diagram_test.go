package refdocs

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
)

const modulePath = "github.com/rtxnik/lazyray"

// liveEdges parses every non-test .go file under <root>/internal and
// <root>/cmd and returns the set of package import edges, keyed "from|to".
// Packages under internal/ are labeled by their path relative to internal/
// (e.g. "tui/theme"); the cmd shell is the single node "cmd". main.go (a
// 14-line stub) and tools/ (build-time generator) stay out on purpose.
func liveEdges(t *testing.T, root string) map[string]bool {
	t.Helper()
	internalDir := filepath.Join(root, "internal")
	edges := map[string]bool{}
	fset := token.NewFileSet()
	addFileEdges := func(path, from string) error {
		f, perr := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if perr != nil {
			return perr
		}
		for _, imp := range f.Imports {
			p, _ := strconv.Unquote(imp.Path.Value)
			prefix := modulePath + "/internal/"
			if !strings.HasPrefix(p, prefix) {
				continue
			}
			to := strings.TrimPrefix(p, prefix)
			if to == from || to == "" {
				continue
			}
			edges[from+"|"+to] = true
		}
		return nil
	}
	err := filepath.Walk(internalDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		from, rerr := filepath.Rel(internalDir, filepath.Dir(path))
		if rerr != nil {
			return rerr
		}
		return addFileEdges(path, filepath.ToSlash(from))
	})
	if err != nil {
		t.Fatal(err)
	}
	cmdEntries, err := os.ReadDir(filepath.Join(root, "cmd"))
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range cmdEntries {
		name := e.Name()
		if e.IsDir() {
			t.Fatalf("cmd/ gained a subdirectory (%s); liveEdges assumes a flat cmd package - extend the walker and the diagram", name)
		}
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		if err := addFileEdges(filepath.Join(root, "cmd", name), "cmd"); err != nil {
			t.Fatal(err)
		}
	}
	return edges
}

var mermaidEdgeRe = regexp.MustCompile(`(?m)^\s*([A-Za-z0-9_]+)\s*-->\s*([A-Za-z0-9_]+)\s*$`)
var mermaidNodeRe = regexp.MustCompile(`(?m)^\s*([A-Za-z0-9_]+)\[([^\]]+)\]\s*$`)

// diagramEdges extracts the internal package edges declared in the single
// ```mermaid block of docs/ARCHITECTURE.md, keyed "fromLabel|toLabel".
func diagramEdges(t *testing.T, root string) map[string]bool {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, "docs", "ARCHITECTURE.md"))
	if err != nil {
		t.Fatal(err)
	}
	block := extractMermaid(t, string(data))
	id2label := map[string]string{}
	for _, m := range mermaidNodeRe.FindAllStringSubmatch(block, -1) {
		id2label[m[1]] = strings.TrimSpace(m[2])
	}
	edges := map[string]bool{}
	for _, m := range mermaidEdgeRe.FindAllStringSubmatch(block, -1) {
		from, okF := id2label[m[1]]
		to, okT := id2label[m[2]]
		if !okF || !okT {
			t.Fatalf("mermaid edge references undeclared node: %s --> %s", m[1], m[2])
		}
		edges[from+"|"+to] = true
	}
	return edges
}

func extractMermaid(t *testing.T, md string) string {
	t.Helper()
	start := strings.Index(md, "```mermaid")
	if start < 0 {
		t.Fatal("no ```mermaid block in docs/ARCHITECTURE.md")
	}
	rest := md[start+len("```mermaid"):]
	end := strings.Index(rest, "```")
	if end < 0 {
		t.Fatal("unterminated ```mermaid block")
	}
	return rest[:end]
}

func TestArchitectureDiagramMatchesImports(t *testing.T) {
	root := repoRoot(t)
	want := liveEdges(t, root)   // labels: internal/-relative paths + "cmd"
	got := diagramEdges(t, root) // labels: the node [label] text

	var missing, extra []string
	for e := range want {
		if !got[e] {
			missing = append(missing, e)
		}
	}
	for e := range got {
		if !want[e] {
			extra = append(extra, e)
		}
	}
	sort.Strings(missing)
	sort.Strings(extra)
	if len(missing) > 0 || len(extra) > 0 {
		t.Fatalf("ARCHITECTURE.md C4 diagram is out of sync with internal/ + cmd imports.\n"+
			"  missing edges (in code, not in diagram): %v\n"+
			"  extra edges (in diagram, not in code):   %v\n"+
			"  → update the ```mermaid block in docs/ARCHITECTURE.md", missing, extra)
	}
}
