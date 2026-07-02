package refdocs

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// mdLinkRe matches inline Markdown links: [text](target). Captures target.
// Reference-style links and bare autolinks are intentionally out of scope —
// the lazyray docs use inline links exclusively.
var mdLinkRe = regexp.MustCompile(`\[[^\]]*\]\(([^)\s]+)(?:\s+"[^"]*")?\)`)

// repoRelTarget returns the repo-relative file path a Markdown link points to,
// resolved relative to the linking file's directory, and ok=true only when the
// link is a repo-relative *file* link worth existence-checking. It returns
// ok=false for external links (scheme://, mailto:, tel:), protocol-relative
// (//host), absolute paths (/x), and pure in-page anchors (#frag). Any #anchor
// suffix is stripped (anchors are not validated — only file existence is).
func repoRelTarget(fromFile, rawLink string) (string, bool) {
	link := strings.TrimSpace(rawLink)
	if link == "" || strings.HasPrefix(link, "#") {
		return "", false
	}
	if strings.HasPrefix(link, "//") || strings.HasPrefix(link, "/") {
		return "", false
	}
	if i := strings.IndexByte(link, ':'); i >= 0 {
		// has a scheme (http:, https:, mailto:, tel:, …) before any slash
		if j := strings.IndexByte(link, '/'); j == -1 || i < j {
			return "", false
		}
	}
	if k := strings.IndexByte(link, '#'); k >= 0 {
		link = link[:k]
	}
	if link == "" { // was a pure #anchor after trimming
		return "", false
	}
	return filepath.Join(filepath.Dir(fromFile), link), true
}

// danglingLinks walks root for *.md files (skipping vendor, node_modules, .git,
// .claude, dist) and returns "<mdfile> -> <link>" for every repo-relative file
// link whose target is missing.
func danglingLinks(root string) ([]string, error) {
	var problems []string
	skipDirs := map[string]bool{"vendor": true, "node_modules": true, ".git": true, ".claude": true, "dist": true}
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if skipDirs[info.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".md") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for _, m := range mdLinkRe.FindAllStringSubmatch(string(data), -1) {
			target, ok := repoRelTarget(path, m[1])
			if !ok {
				continue
			}
			if _, statErr := os.Stat(target); statErr != nil {
				rel, _ := filepath.Rel(root, path)
				problems = append(problems, rel+" -> "+m[1])
			}
		}
		return nil
	})
	return problems, err
}

func TestRepoRelTarget(t *testing.T) {
	cases := []struct {
		from, link, want string
		ok               bool
	}{
		{"docs/ARCHITECTURE.md", "https://example.com", "", false},
		{"docs/ARCHITECTURE.md", "mailto:a@b.c", "", false},
		{"docs/ARCHITECTURE.md", "#invariants", "", false},
		{"docs/ARCHITECTURE.md", "/abs/path.md", "", false},
		{"README.md", "docs/reference/configuration.md", "docs/reference/configuration.md", true},
		{"docs/ARCHITECTURE.md", "../SECURITY.md", "SECURITY.md", true},
		{"CONTRIBUTING.md", "docs/ARCHITECTURE.md#invariants--guards", "docs/ARCHITECTURE.md", true},
	}
	for _, c := range cases {
		got, ok := repoRelTarget(c.from, c.link)
		// repoRelTarget returns an OS-native path (filepath.Join); compare on a
		// slash-normalized form so the assertion holds on Windows as well.
		if ok != c.ok || (ok && filepath.ToSlash(got) != c.want) {
			t.Errorf("repoRelTarget(%q,%q) = (%q,%v), want (%q,%v)", c.from, c.link, filepath.ToSlash(got), ok, c.want, c.ok)
		}
	}
}

func TestDanglingLinksDetectsBrokenFixture(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "good.md"), []byte("see [x](good.md)"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "bad.md"), []byte("see [y](missing.md) and [z](https://ok)"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := danglingLinks(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || !strings.Contains(got[0], "bad.md -> missing.md") {
		t.Fatalf("want exactly [bad.md -> missing.md], got %v", got)
	}
}

func TestNoDanglingDocLinks(t *testing.T) {
	root := repoRoot(t)
	problems, err := danglingLinks(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(problems) > 0 {
		t.Fatalf("dangling internal Markdown links:\n  %s", strings.Join(problems, "\n  "))
	}
}
