package refdocs

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// exitConstNames parses cmd/exit.go and returns its Exit* constant identifiers,
// so a newly-added exit code that is not documented fails the test below.
func exitConstNames(t *testing.T, root string) []string {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filepath.Join(root, "cmd", "exit.go"), nil, 0)
	if err != nil {
		t.Fatalf("parsing cmd/exit.go: %v", err)
	}
	var names []string
	for _, decl := range f.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if !ok || gd.Tok != token.CONST {
			continue
		}
		for _, spec := range gd.Specs {
			vs, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			for _, name := range vs.Names {
				if strings.HasPrefix(name.Name, "Exit") {
					names = append(names, name.Name)
				}
			}
		}
	}
	return names
}

func TestExitCodesReferenceDocumentsEveryConstant(t *testing.T) {
	root := repoRoot(t)
	names := exitConstNames(t, root)
	if len(names) == 0 {
		t.Fatal("no Exit* constants found in cmd/exit.go")
	}
	data, err := os.ReadFile(filepath.Join(root, "docs", "reference", "exit-codes.md"))
	if err != nil {
		t.Fatalf("reading exit-codes.md: %v", err)
	}
	doc := string(data)
	for _, n := range names {
		if !strings.Contains(doc, n) {
			t.Errorf("exit-codes.md does not document constant %s", n)
		}
	}
}
