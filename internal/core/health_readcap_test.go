package core

import (
	"bytes"
	"io"
	"testing"
)

func TestReadCappedBodyTruncatesOversizedBody(t *testing.T) {
	body := bytes.Repeat([]byte("a"), 1<<20)
	rc := io.NopCloser(bytes.NewReader(body))

	got, err := readCappedBody(rc)
	if err != nil {
		t.Fatalf("readCappedBody returned error: %v", err)
	}
	if len(got) != maxIPCheckBytes {
		t.Fatalf("expected truncated length %d, got %d", maxIPCheckBytes, len(got))
	}
}

func TestReadCappedBodyReturnsSmallBodyWhole(t *testing.T) {
	body := []byte("203.0.113.7")
	rc := io.NopCloser(bytes.NewReader(body))

	got, err := readCappedBody(rc)
	if err != nil {
		t.Fatalf("readCappedBody returned error: %v", err)
	}
	if !bytes.Equal(got, body) {
		t.Fatalf("expected %q, got %q", body, got)
	}
}
