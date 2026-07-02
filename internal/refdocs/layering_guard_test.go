package refdocs

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// plantedOrchestration bypasses the service layer two ways: an aliased
// lifecycle import spawning the supervisor, and a direct call to the
// Service-owned core.WriteXrayConfig. The guard must catch both.
const plantedOrchestration = `package cmd

import (
	"github.com/rtxnik/lazyray/internal/core"
	lc "github.com/rtxnik/lazyray/internal/lifecycle"
)

func planted() error {
	if err := core.WriteXrayConfig(nil, nil); err != nil {
		return err
	}
	return lc.SpawnDetached(nil)
}
`

// plantedDotImport makes symbol use unauditable; the scanner must flag the
// import itself.
const plantedDotImport = `package cmd

import . "github.com/rtxnik/lazyray/internal/lifecycle"

var _ = SupervisorAlive
`

// plantedSubpackage puts a Service-owned core call in a tui subpackage --
// the recursive core-denylist scan must see it.
const plantedSubpackage = `package modals

import "github.com/rtxnik/lazyray/internal/core"

func plantedModal() error {
	return core.WriteXrayConfig(nil, nil)
}
`

func writePlanted(t *testing.T, root, rel, src string) {
	t.Helper()
	abs := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(abs, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestLayeringGuardDetectsPlantedViolations(t *testing.T) {
	root := t.TempDir()
	writePlanted(t, root, "cmd/planted.go", plantedOrchestration)

	lcRefs, err := pkgSelectorRefs(root, []string{"cmd"}, modulePath+"/internal/lifecycle")
	if err != nil {
		t.Fatal(err)
	}
	violations, stale := refViolations(lcRefs, nil)
	if len(stale) != 0 {
		t.Fatalf("unexpected stale entries: %v", stale)
	}
	if len(violations) != 1 || violations[0] != "cmd/planted.go:12 SpawnDetached" {
		t.Fatalf("aliased lifecycle violation not detected, got %v", violations)
	}

	coreRefs, err := pkgSelectorRefs(root, []string{"cmd"}, modulePath+"/internal/core")
	if err != nil {
		t.Fatal(err)
	}
	coreViolations, _ := refViolations(coreRefs, nil)
	if len(coreViolations) != 1 || coreViolations[0] != "cmd/planted.go:9 WriteXrayConfig" {
		t.Fatalf("core bypass not detected, got %v", coreViolations)
	}
}

func TestLayeringGuardRejectsDotImport(t *testing.T) {
	root := t.TempDir()
	writePlanted(t, root, "cmd/dot.go", plantedDotImport)

	refs, err := pkgSelectorRefs(root, []string{"cmd"}, modulePath+"/internal/lifecycle")
	if err != nil {
		t.Fatal(err)
	}
	violations, _ := refViolations(refs, nil)
	if len(violations) != 1 || violations[0] != "cmd/dot.go:3 ." {
		t.Fatalf("dot-import not flagged, got %v", violations)
	}
}

func TestCoreDenylistSeesTuiSubpackages(t *testing.T) {
	root := t.TempDir()
	writePlanted(t, root, "internal/tui/modals/planted.go", plantedSubpackage)
	if err := os.MkdirAll(filepath.Join(root, "cmd"), 0o755); err != nil {
		t.Fatal(err)
	}
	refs, err := pkgSelectorRefs(root, shellCoreScanDirs(t, root), modulePath+"/internal/core")
	if err != nil {
		t.Fatal(err)
	}
	violations, _ := refViolations(refs, nil)
	if len(violations) != 1 || violations[0] != "internal/tui/modals/planted.go:6 WriteXrayConfig" {
		t.Fatalf("subpackage core bypass not detected, got %v", violations)
	}
}

func TestRefViolationsReportsStaleAllowlistEntries(t *testing.T) {
	allow := map[string]map[string]bool{
		"cmd/planted.go": {"ReadState": true},
	}
	violations, stale := refViolations(nil, allow)
	if len(violations) != 0 {
		t.Fatalf("unexpected violations: %v", violations)
	}
	if len(stale) != 1 || stale[0] != "cmd/planted.go ReadState" {
		t.Fatalf("stale allowlist entry not reported, got %v", stale)
	}
}

func TestRefViolationsAcceptsAllowlistedRefs(t *testing.T) {
	refs := []selRef{{file: "cmd/planted.go", line: 12, symbol: "SpawnDetached"}}
	allow := map[string]map[string]bool{
		"cmd/planted.go": {"SpawnDetached": true},
	}
	violations, stale := refViolations(refs, allow)
	if len(violations) != 0 {
		t.Fatalf("allowlisted ref reported as violation: %v", violations)
	}
	if len(stale) != 0 {
		t.Fatalf("used allowlist entry reported stale: %v", stale)
	}
}

// selRef is one selector reference <pkg>.<Symbol> found in a shell source
// file, or the pseudo-reference "." for a dot-import of the target package.
type selRef struct {
	file   string // slash-normalized path relative to the scanned root
	line   int
	symbol string
}

// pkgSelectorRefs parses every non-test .go file directly inside the given
// directories (relative to root; non-recursive -- each dir is one Go package)
// and returns every selector reference to the package with the given module
// import path. The file's actual import name is resolved, so aliased imports
// are caught; a dot-import is returned as symbol "." because its uses cannot
// be audited syntactically. Limitation (accepted): method calls on values of
// the target package's types (e.g. sup.Run()) are not package selectors and
// are not reported -- constructing the value is the audited gate.
func pkgSelectorRefs(root string, dirs []string, importPath string) ([]selRef, error) {
	defaultName := importPath[strings.LastIndex(importPath, "/")+1:]
	fset := token.NewFileSet()
	var refs []selRef
	for _, dir := range dirs {
		entries, err := os.ReadDir(filepath.Join(root, filepath.FromSlash(dir)))
		if err != nil {
			return nil, err
		}
		for _, e := range entries {
			name := e.Name()
			if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
				continue
			}
			path := filepath.Join(root, filepath.FromSlash(dir), name)
			f, err := parser.ParseFile(fset, path, nil, 0)
			if err != nil {
				return nil, err
			}
			imp := findImport(f, importPath)
			if imp == nil {
				continue
			}
			rel := dir + "/" + name
			local := defaultName
			if imp.Name != nil {
				local = imp.Name.Name
			}
			switch local {
			case ".":
				refs = append(refs, selRef{file: rel, line: fset.Position(imp.Pos()).Line, symbol: "."})
				continue
			case "_":
				continue
			}
			ast.Inspect(f, func(n ast.Node) bool {
				sel, ok := n.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				if id, ok := sel.X.(*ast.Ident); ok && id.Name == local {
					refs = append(refs, selRef{file: rel, line: fset.Position(sel.Pos()).Line, symbol: sel.Sel.Name})
				}
				return true
			})
		}
	}
	return refs, nil
}

// findImport returns f's import spec for importPath, or nil.
func findImport(f *ast.File, importPath string) *ast.ImportSpec {
	for _, imp := range f.Imports {
		if p, _ := strconv.Unquote(imp.Path.Value); p == importPath {
			return imp
		}
	}
	return nil
}

// refViolations classifies refs against a per-file symbol allowlist. It
// returns disallowed references (violations, "file:line Symbol") and
// allowlist entries that matched nothing (stale, "file Symbol" -- prune them
// so the pin stays an exact statement of reality), both sorted.
func refViolations(refs []selRef, allow map[string]map[string]bool) (violations, stale []string) {
	used := map[string]map[string]bool{}
	for _, r := range refs {
		if !allow[r.file][r.symbol] {
			violations = append(violations, fmt.Sprintf("%s:%d %s", r.file, r.line, r.symbol))
		}
		if used[r.file] == nil {
			used[r.file] = map[string]bool{}
		}
		used[r.file][r.symbol] = true
	}
	for file, syms := range allow {
		for sym := range syms {
			if !used[file][sym] {
				stale = append(stale, file+" "+sym)
			}
		}
	}
	sort.Strings(violations)
	sort.Strings(stale)
	return violations, stale
}

// --- G6: the shell/service layering boundary, pinned ---

// shellDirs are the two shell packages (non-recursive package dirs).
var shellDirs = []string{"cmd", "internal/tui"}

// sanctionedLifecycleImporters is the closed set of packages that may import
// internal/lifecycle at all (labels as in the architecture diagram). status
// and doctor assemble read-only snapshots, app is the orchestration home,
// cmd and tui are the shells whose direct use shellLifecycleAllow pins
// per file.
var sanctionedLifecycleImporters = map[string]bool{
	"app":    true,
	"cmd":    true,
	"doctor": true,
	"status": true,
	"tui":    true,
}

// shellLifecycleAllow pins, per shell file, every lifecycle symbol that may
// be referenced directly. Everything else must go through internal/app.
// Extending this map is a deliberate, reviewed act: add a justification
// comment with the new entry.
var shellLifecycleAllow = map[string]map[string]bool{
	// __run hosts the resident supervisor itself -- the one sanctioned deep
	// consumer of lifecycle in cmd.
	"cmd/run.go": symbolSet("Supervisor", "Owner", "OwnerDaemon", "ProxyDefault", "ProxyForceOff", "ProxyForceOn", "StagedError", "WriteStartupError"),
	// start/stop/restart keep idempotency checks, crash self-heal and the
	// CLI-only graceful-signal path (Service.Disconnect deliberately omits
	// the hard-kill escalation these commands pair with it).
	"cmd/start.go":   symbolSet("OwnerDaemon", "ReadState", "Reconcile", "SupervisorAlive"),
	"cmd/stop.go":    symbolSet("ReadState", "Reconcile", "SignalSupervisor", "SupervisorAlive"),
	"cmd/restart.go": symbolSet("OwnerDaemon", "ReadState", "Reconcile", "SignalSupervisor", "SupervisorAlive"),
	// read-only probe-context construction for connectivity tests.
	"cmd/test.go": symbolSet("ProbeContextFor"),
	// the TUI reads liveness/state for gating and rendering; every mutation
	// goes through the app service.
	"internal/tui/app.go": symbolSet("OwnerTUI", "ProbeContextFor", "ReadStartupError", "ReadState", "SupervisorAlive"),
}

// serviceOwnedCoreSymbols are the internal/core entry points wrapped by
// internal/app.Service (WriteActiveConfig, ImportSubscription). Shells call
// the service, never these.
var serviceOwnedCoreSymbols = map[string]bool{
	"WriteXrayConfig":    true,
	"ImportSubscription": true,
}

func symbolSet(names ...string) map[string]bool {
	s := make(map[string]bool, len(names))
	for _, n := range names {
		s[n] = true
	}
	return s
}

func TestOnlySanctionedPackagesImportLifecycle(t *testing.T) {
	root := repoRoot(t)
	edges := liveEdges(t, root)
	seen := map[string]bool{}
	var wrong []string
	for e := range edges {
		ft := strings.SplitN(e, "|", 2)
		if ft[1] != "lifecycle" {
			continue
		}
		seen[ft[0]] = true
		if !sanctionedLifecycleImporters[ft[0]] {
			wrong = append(wrong, ft[0])
		}
	}
	var stale []string
	for p := range sanctionedLifecycleImporters {
		if !seen[p] {
			stale = append(stale, p)
		}
	}
	sort.Strings(wrong)
	sort.Strings(stale)
	if len(wrong) > 0 || len(stale) > 0 {
		t.Fatalf("lifecycle importer set drifted.\n"+
			"  unsanctioned importers: %v\n"+
			"  sanctioned but no longer importing (prune): %v\n"+
			"  -> supervisor orchestration belongs in internal/app.Service; see docs/ARCHITECTURE.md (G6)", wrong, stale)
	}
}

func TestShellLifecycleUseIsAllowlisted(t *testing.T) {
	root := repoRoot(t)
	refs, err := pkgSelectorRefs(root, shellDirs, modulePath+"/internal/lifecycle")
	if err != nil {
		t.Fatal(err)
	}
	violations, stale := refViolations(refs, shellLifecycleAllow)
	if len(violations) > 0 || len(stale) > 0 {
		t.Fatalf("shell use of internal/lifecycle drifted from the G6 allowlist.\n"+
			"  disallowed references: %v\n"+
			"  stale allowlist entries (prune): %v\n"+
			"  -> multi-step supervisor orchestration belongs in internal/app.Service; "+
			"a deliberate direct use needs a justified entry in shellLifecycleAllow", violations, stale)
	}
}

// shellCoreScanDirs returns shellDirs plus every internal/tui subpackage
// directory (relative, slash paths). The core denylist has no importer-set
// backstop (core is legitimately imported everywhere), so it must see the
// whole shell tree; the lifecycle allowlist keeps shellDirs because any
// subpackage lifecycle import already trips the importer-set pin.
func shellCoreScanDirs(t *testing.T, root string) []string {
	t.Helper()
	dirs := append([]string{}, shellDirs...)
	tuiDir := filepath.Join(root, "internal", "tui")
	err := filepath.Walk(tuiDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() || path == tuiDir {
			return nil
		}
		rel, rerr := filepath.Rel(root, path)
		if rerr != nil {
			return rerr
		}
		dirs = append(dirs, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return dirs
}

func TestShellsDoNotBypassServiceOwnedFlows(t *testing.T) {
	root := repoRoot(t)
	refs, err := pkgSelectorRefs(root, shellCoreScanDirs(t, root), modulePath+"/internal/core")
	if err != nil {
		t.Fatal(err)
	}
	var violations []string
	for _, r := range refs {
		if serviceOwnedCoreSymbols[r.symbol] || r.symbol == "." {
			violations = append(violations, fmt.Sprintf("%s:%d core.%s", r.file, r.line, r.symbol))
		}
	}
	sort.Strings(violations)
	if len(violations) > 0 {
		t.Fatalf("shells bypass a Service-owned flow:\n  %s\n"+
			"  -> call the internal/app.Service method instead (WriteActiveConfig / ImportSubscription)",
			strings.Join(violations, "\n  "))
	}
}
