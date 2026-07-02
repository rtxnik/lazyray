package cmd

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// helpContractAllowlist holds command paths exempt from the help contract: the
// root TUI launcher (documented in the README), cobra's generated built-ins,
// and the hidden supervisor command have no user-facing help to write.
var helpContractAllowlist = map[string]bool{
	"lzr":                       true,
	"lzr help":                  true,
	"lzr completion":            true,
	"lzr completion bash":       true,
	"lzr completion fish":       true,
	"lzr completion powershell": true,
	"lzr completion zsh":        true,
	"lzr __run":                 true,
}

// TestHelpContract enforces, per command: a non-empty Long for every command; a
// non-empty Example for every runnable command; and a usage string on every
// flag.
func TestHelpContract(t *testing.T) {
	var walk func(c *cobra.Command)
	walk = func(c *cobra.Command) {
		path := c.CommandPath()
		if !c.Hidden && !helpContractAllowlist[path] {
			t.Run(strings.ReplaceAll(path, " ", "_"), func(t *testing.T) {
				if strings.TrimSpace(c.Long) == "" {
					t.Errorf("%q: Long is empty (every command needs a Long)", path)
				}
				if c.Runnable() && strings.TrimSpace(c.Example) == "" {
					t.Errorf("%q: runnable command has no Example", path)
				}
				c.Flags().VisitAll(func(f *pflag.Flag) {
					if strings.TrimSpace(f.Usage) == "" {
						t.Errorf("%q: flag --%s has no usage string", path, f.Name)
					}
				})
			})
		}
		for _, sub := range c.Commands() {
			walk(sub)
		}
	}
	walk(RootCmd())
}
