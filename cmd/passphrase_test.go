package cmd

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// fakeTerminal redirects the passphrase prompt seams for one test.
// answers are returned in order by the fake readPassword.
func fakeTerminal(t *testing.T, isTTY bool, answers ...string) {
	t.Helper()
	origRead, origIsTTY := readPassword, stdinIsTerminal
	t.Cleanup(func() { readPassword, stdinIsTerminal = origRead, origIsTTY })
	i := 0
	readPassword = func(fd int) ([]byte, error) {
		if i >= len(answers) {
			t.Fatal("readPassword called more times than answers provided")
		}
		a := answers[i]
		i++
		return []byte(a), nil
	}
	stdinIsTerminal = func() bool { return isTTY }
}

// clearPassphraseEnv guarantees LAZYRAY_PASSPHRASE is unset and restored.
func clearPassphraseEnv(t *testing.T) {
	t.Helper()
	t.Setenv(passphraseEnvVar, "sentinel") // registers restore
	os.Unsetenv(passphraseEnvVar)
}

func TestResolvePassphrase_FileSource(t *testing.T) {
	clearPassphraseEnv(t)
	fakeTerminal(t, false)
	p := filepath.Join(t.TempDir(), "pass")
	if err := os.WriteFile(p, []byte("file-secret\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := resolvePassphrase(p, true)
	if err != nil {
		t.Fatalf("resolvePassphrase(file) error = %v", err)
	}
	if got != "file-secret" {
		t.Errorf("passphrase = %q, want file-secret (trailing newline trimmed)", got)
	}
}

func TestResolvePassphrase_FileEmpty(t *testing.T) {
	clearPassphraseEnv(t)
	fakeTerminal(t, false)
	p := filepath.Join(t.TempDir(), "pass")
	if err := os.WriteFile(p, []byte("\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := resolvePassphrase(p, false); err == nil {
		t.Fatal("empty passphrase file must be an error")
	}
}

func TestResolvePassphrase_EnvSource(t *testing.T) {
	fakeTerminal(t, false)
	t.Setenv(passphraseEnvVar, "env-secret")
	got, err := resolvePassphrase("", false)
	if err != nil {
		t.Fatalf("resolvePassphrase(env) error = %v", err)
	}
	if got != "env-secret" {
		t.Errorf("passphrase = %q, want env-secret", got)
	}
}

func TestResolvePassphrase_EnvSetButEmpty(t *testing.T) {
	fakeTerminal(t, false)
	t.Setenv(passphraseEnvVar, "")
	if _, err := resolvePassphrase("", false); err == nil {
		t.Fatal("empty env passphrase must be an error, not a fallthrough")
	}
}

func TestResolvePassphrase_NoSourceNonInteractive(t *testing.T) {
	clearPassphraseEnv(t)
	fakeTerminal(t, false)
	_, err := resolvePassphrase("", true)
	if !errors.Is(err, errNoPassphraseSource) {
		t.Fatalf("err = %v, want errNoPassphraseSource", err)
	}
}

func TestResolvePassphrase_PromptConfirmMatch(t *testing.T) {
	clearPassphraseEnv(t)
	fakeTerminal(t, true, "tty-secret", "tty-secret")
	got, err := resolvePassphrase("", true)
	if err != nil {
		t.Fatalf("resolvePassphrase(prompt) error = %v", err)
	}
	if got != "tty-secret" {
		t.Errorf("passphrase = %q, want tty-secret", got)
	}
}

func TestResolvePassphrase_PromptConfirmMismatch(t *testing.T) {
	clearPassphraseEnv(t)
	fakeTerminal(t, true, "one", "two")
	if _, err := resolvePassphrase("", true); err == nil {
		t.Fatal("mismatched confirmation must be an error")
	}
}

func TestResolvePassphrase_PromptEmpty(t *testing.T) {
	clearPassphraseEnv(t)
	fakeTerminal(t, true, "")
	if _, err := resolvePassphrase("", false); err == nil {
		t.Fatal("empty interactive passphrase must be an error")
	}
}

func TestResolvePassphrase_NoConfirmSinglePrompt(t *testing.T) {
	clearPassphraseEnv(t)
	fakeTerminal(t, true, "only-once")
	got, err := resolvePassphrase("", false)
	if err != nil {
		t.Fatalf("resolvePassphrase() error = %v", err)
	}
	if got != "only-once" {
		t.Errorf("passphrase = %q", got)
	}
}
