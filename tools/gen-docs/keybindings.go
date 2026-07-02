package main

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/rtxnik/lazyray/internal/tui/commands"
)

// writeKeybindings renders the keybindings reference from the command registry
// built on the built-in defaults (BaseKeyMap), so output never depends on a
// local keys.yaml.
func writeKeybindings(path string) error {
	reg := commands.New(commands.BaseKeyMap())

	launch := map[string]bool{}
	for _, c := range reg.Launchable() {
		launch[c.ID] = true
	}

	var b strings.Builder
	b.WriteString("# Keybindings Reference\n\n")
	b.WriteString("Default keybindings for the lazyray TUI. Override any of them in ")
	b.WriteString("`keys.yaml` in your config directory (see the configuration reference). ")
	b.WriteString("The **Palette** column marks commands launchable from the command palette (`:`).\n\n")

	for _, g := range reg.ByCategory() {
		if len(g.Commands) == 0 {
			continue
		}
		b.WriteString("## " + g.Title + "\n\n")
		b.WriteString("| Key | Action | Scope | Palette |\n")
		b.WriteString("|-----|--------|-------|----------|\n")
		for _, c := range g.Commands {
			palette := ""
			if launch[c.ID] {
				palette = "yes"
			}
			fmt.Fprintf(&b, "| `%s` | %s | %s | %s |\n",
				commands.KeyDisplay(c.Binding), c.Title, c.Scope.String(), palette)
		}
		b.WriteString("\n")
	}

	b.WriteString("## Rebinding keys\n\n")
	b.WriteString("Set any of these fields in `keys.yaml` to override the default binding:\n\n")
	for _, name := range keysConfigFields() {
		b.WriteString("- `" + name + "`\n")
	}
	b.WriteString("\n`health` is a backward-compatible alias for `doctor`.\n")

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

// keysConfigFields returns the yaml field names of commands.KeysConfig in
// declaration order, so the rebinding list cannot drift from the struct.
func keysConfigFields() []string {
	t := reflect.TypeOf(commands.KeysConfig{})
	var out []string
	for i := 0; i < t.NumField(); i++ {
		name := strings.SplitN(t.Field(i).Tag.Get("yaml"), ",", 2)[0]
		if name != "" && name != "health" { // health documented separately as an alias
			out = append(out, name)
		}
	}
	return out
}
