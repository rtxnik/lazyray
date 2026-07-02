// Command gen-docs generates reference documentation for the lzr CLI from the
// cobra command tree exposed by cmd.RootCmd(): troff man pages and/or a
// Markdown command reference. Output is reproducible (no auto-gen timestamps).
package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/rtxnik/lazyray/cmd"
	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

type options struct {
	format  string
	manOut  string
	mdOut   string
	keysOut string
}

func main() {
	var opts options
	flag.StringVar(&opts.format, "format", "man", "output format: man | markdown | all")
	flag.StringVar(&opts.manOut, "out", "man/man1", "output directory for man pages")
	flag.StringVar(&opts.mdOut, "md-out", "docs/reference/cli", "output directory for the Markdown command reference")
	flag.StringVar(&opts.keysOut, "keys-out", "docs/reference/keybindings.md", "output file for the keybindings reference")
	flag.Parse()

	if err := run(opts); err != nil {
		fmt.Fprintln(os.Stderr, "gen-docs:", err)
		os.Exit(1)
	}
}

func run(opts options) error {
	switch opts.format {
	case "man":
		return genMan(opts.manOut)
	case "markdown":
		if err := genMarkdownCLI(opts.mdOut); err != nil {
			return err
		}
		return writeKeybindings(opts.keysOut)
	case "all":
		if err := genMan(opts.manOut); err != nil {
			return err
		}
		if err := genMarkdownCLI(opts.mdOut); err != nil {
			return err
		}
		return writeKeybindings(opts.keysOut)
	default:
		return fmt.Errorf("unknown --format %q (want man|markdown|all)", opts.format)
	}
}

// preparedRoot returns the root command with the cobra-added completion/help
// subcommands stripped and the auto-gen tag disabled on every command (the flag
// is per-command, not inherited), so generated output is reproducible.
func preparedRoot() *cobra.Command {
	root := cmd.RootCmd()
	for _, name := range []string{"completion", "help"} {
		if c, _, err := root.Find([]string{name}); err == nil && c != root {
			root.RemoveCommand(c)
		}
	}
	disableAutoGenTag(root)
	return root
}

func disableAutoGenTag(c *cobra.Command) {
	c.DisableAutoGenTag = true
	for _, child := range c.Commands() {
		disableAutoGenTag(child)
	}
}

func genMan(out string) error {
	if err := os.MkdirAll(out, 0o755); err != nil {
		return fmt.Errorf("creating output dir %q: %w", out, err)
	}
	root := preparedRoot()
	header := &doc.GenManHeader{
		Title:   "LZR",
		Section: "1",
		Source:  "lazyray",
		Manual:  "lazyray Manual",
		Date:    timePtr(time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)),
	}
	if err := doc.GenManTree(root, header, out); err != nil {
		return fmt.Errorf("generating man tree: %w", err)
	}
	return nil
}

func genMarkdownCLI(out string) error {
	if err := os.MkdirAll(out, 0o755); err != nil {
		return fmt.Errorf("creating output dir %q: %w", out, err)
	}
	root := preparedRoot()
	if err := doc.GenMarkdownTree(root, out); err != nil {
		return fmt.Errorf("generating markdown tree: %w", err)
	}
	return nil
}

func timePtr(t time.Time) *time.Time { return &t }
