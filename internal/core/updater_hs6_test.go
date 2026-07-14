package core

import (
	"bytes"
	"errors"
	"testing"
)

func TestCopyCappedRejectsOversize(t *testing.T) {
	var buf bytes.Buffer
	// cap=4; a 5-byte source is oversize (tests the bomb logic without a huge alloc).
	err := copyCapped(&buf, bytes.NewReader([]byte("12345")), 4)
	if !errors.Is(err, ErrXrayMemberTooLarge) {
		t.Fatalf("want ErrXrayMemberTooLarge, got %v", err)
	}
}

func TestCopyCappedAcceptsExactCap(t *testing.T) {
	var buf bytes.Buffer
	if err := copyCapped(&buf, bytes.NewReader([]byte("1234")), 4); err != nil {
		t.Fatalf("exact-cap member rejected: %v", err)
	}
	if buf.String() != "1234" {
		t.Fatalf("got %q", buf.String())
	}
}
