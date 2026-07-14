package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestImportSubscriptionRedactsEcho(t *testing.T) {
	var buf bytes.Buffer
	importSubscriptionEcho(&buf, "https://host.example/sub?token=SECRET")
	if strings.Contains(buf.String(), "SECRET") {
		t.Fatalf("subscription echo leaked token: %q", buf.String())
	}
	if !strings.Contains(buf.String(), "host.example") {
		t.Fatalf("echo missing host: %q", buf.String())
	}
}
