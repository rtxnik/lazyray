package commands

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"reflect"
	"testing"
)

// Guard 1: KeyMap fields and Registry IDs are an exact bijection. A new binding
// without a Command (or a Command without a binding) fails here.
//
// Guard G1 (KeyMap<->Registry bijection). Documented in docs/ARCHITECTURE.md
// (Invariants & Guards). Keep that section and this test in sync.
func TestRegistryMatchesKeyMap(t *testing.T) {
	reg := New(DefaultKeyMap())

	regIDs := map[string]bool{}
	for _, c := range reg {
		if regIDs[c.ID] {
			t.Fatalf("duplicate registry ID %q", c.ID)
		}
		regIDs[c.ID] = true
	}

	fields := map[string]bool{}
	tp := reflect.TypeOf(KeyMap{})
	for i := 0; i < tp.NumField(); i++ {
		fields[tp.Field(i).Name] = true
	}

	for name := range fields {
		if !regIDs[name] {
			t.Errorf("KeyMap field %q has no registry Command", name)
		}
	}
	for id := range regIDs {
		if !fields[id] {
			t.Errorf("registry Command %q has no KeyMap field", id)
		}
	}
}

// Guard 2: every a.keys.<Field> dispatched in app.go is a registered command,
// and every registered command is dispatched. Closes the orphan/missing-handler
// drift class that the metadata-only registry (Option A) would otherwise leave.
//
// Guard G2 (dispatch completeness). Documented in docs/ARCHITECTURE.md
// (Invariants & Guards). Keep that section and this test in sync.
func TestDispatchReferencesMatchRegistry(t *testing.T) {
	src, err := os.ReadFile("../app.go")
	if err != nil {
		t.Fatalf("read app.go: %v", err)
	}
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "app.go", src, 0)
	if err != nil {
		t.Fatalf("parse app.go: %v", err)
	}

	// Collect <recv>.keys.<Field> selector references (receiver name agnostic).
	used := map[string]bool{}
	ast.Inspect(f, func(n ast.Node) bool {
		outer, ok := n.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		inner, ok := outer.X.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		if inner.Sel.Name == "keys" {
			used[outer.Sel.Name] = true
		}
		return true
	})

	reg := New(DefaultKeyMap())
	regIDs := map[string]bool{}
	for _, c := range reg {
		regIDs[c.ID] = true
		if !used[c.ID] {
			t.Errorf("command %q is registered but never dispatched (no .keys.%s in app.go)", c.ID, c.ID)
		}
	}
	for name := range used {
		if !regIDs[name] {
			t.Errorf("app.go dispatches .keys.%s but it is not in the registry", name)
		}
	}
}
