package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"

	"golang.org/x/term"
)

// readStdinLine reads a single line (up to the first newline) so a secret-bearing
// URL can be supplied off the command line, keeping it out of ps/proc/argv.
func readStdinLine(r io.Reader) (string, error) {
	br := bufio.NewReader(r)
	line, err := br.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}
	line = strings.TrimRight(line, "\r\n")
	if line == "" {
		return "", fmt.Errorf("no input on stdin")
	}
	return line, nil
}

// redactURL returns a safe-to-echo form of raw. Only http/https reveal the host
// (their secret lives in the dropped path/query); every other scheme is fully
// redacted because schemes like vmess://<base64> encode the whole secret in the
// host position, which must never be echoed.
func redactURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" {
		return "(redacted)"
	}
	if (u.Scheme == "http" || u.Scheme == "https") && u.Host != "" {
		return fmt.Sprintf("%s://%s/…", u.Scheme, u.Host)
	}
	return fmt.Sprintf("%s://(redacted)", u.Scheme)
}

// passphraseEnvVar is the non-interactive passphrase source for backup
// encryption and restore decryption.
const passphraseEnvVar = "LAZYRAY_PASSPHRASE"

// errNoPassphraseSource reports that no passphrase source is available in a
// non-interactive session. Callers wrap it with a context-specific hint.
var errNoPassphraseSource = errors.New("no passphrase source available")

// Test seams: production code always uses the real terminal.
var (
	readPassword    = term.ReadPassword
	stdinIsTerminal = func() bool { return term.IsTerminal(int(os.Stdin.Fd())) }
)

// resolvePassphrase returns the passphrase for backup encryption or restore
// decryption. Sources in priority order: passphraseFile (first line), the
// LAZYRAY_PASSPHRASE environment variable, an interactive hidden prompt.
// confirm=true (used when creating a backup) prompts twice and requires both
// entries to match. Interactive prompts go to stderr so stdout stays clean.
// Known limitation: a signal arriving mid-prompt can leave terminal echo off
// (standard hidden-prompt caveat).
func resolvePassphrase(passphraseFile string, confirm bool) (string, error) {
	if passphraseFile != "" {
		data, err := os.ReadFile(passphraseFile)
		if err != nil {
			return "", fmt.Errorf("reading passphrase file: %w", err)
		}
		pass := firstLine(string(data))
		if pass == "" {
			return "", fmt.Errorf("passphrase must not be empty")
		}
		return pass, nil
	}

	if pass, ok := os.LookupEnv(passphraseEnvVar); ok {
		if pass == "" {
			return "", fmt.Errorf("passphrase must not be empty")
		}
		return pass, nil
	}

	if !stdinIsTerminal() {
		return "", errNoPassphraseSource
	}

	fmt.Fprint(os.Stderr, "Passphrase: ")
	first, err := readPassword(int(os.Stdin.Fd()))
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return "", fmt.Errorf("reading passphrase: %w", err)
	}
	if len(first) == 0 {
		return "", fmt.Errorf("passphrase must not be empty")
	}
	if confirm {
		fmt.Fprint(os.Stderr, "Confirm passphrase: ")
		second, err := readPassword(int(os.Stdin.Fd()))
		fmt.Fprintln(os.Stderr)
		if err != nil {
			return "", fmt.Errorf("reading passphrase confirmation: %w", err)
		}
		if string(first) != string(second) {
			return "", fmt.Errorf("passphrases do not match")
		}
	}
	return string(first), nil
}

// firstLine returns s up to the first CR or LF.
func firstLine(s string) string {
	if i := strings.IndexAny(s, "\r\n"); i >= 0 {
		return s[:i]
	}
	return s
}
