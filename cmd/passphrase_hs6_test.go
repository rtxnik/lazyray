package cmd

import (
	"strings"
	"testing"
)

func TestReadStdinLine(t *testing.T) {
	got, err := readStdinLine(strings.NewReader("vless://tok@h:443\nsecond\n"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "vless://tok@h:443" {
		t.Fatalf("got %q", got)
	}
}

func TestReadStdinLineEmpty(t *testing.T) {
	if _, err := readStdinLine(strings.NewReader("\n")); err == nil {
		t.Fatal("empty stdin should error")
	}
}

func TestRedactURL(t *testing.T) {
	cases := map[string]string{
		"https://example.com/sub?token=abc": "https://example.com/…",
		"http://user:SECRET@host.example/x": "http://host.example/…",
		"vmess://eyJhIjoic2VjcmV0In0=":      "vmess://(redacted)",
		"vless://uuid-secret@host:443#n":    "vless://(redacted)",
		"not a url":                         "(redacted)",
		"":                                  "(redacted)",
	}
	for in, want := range cases {
		if got := redactURL(in); got != want {
			t.Errorf("redactURL(%q) = %q, want %q", in, got, want)
		}
	}
}
