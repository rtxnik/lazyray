package notify

import (
	"testing"
	"time"
)

func TestSeverityTag(t *testing.T) {
	cases := map[Severity]string{Info: "INFO", Success: "OK", Warning: "WARN", Error: "ERROR"}
	for sev, want := range cases {
		if got := sev.Tag(); got != want {
			t.Errorf("Severity(%d).Tag() = %q, want %q", sev, got, want)
		}
	}
}

func TestAddAssignsMonotonicID(t *testing.T) {
	l := New(10)
	a := l.Add(Notice{Severity: Info, Message: "a"})
	b := l.Add(Notice{Severity: Info, Message: "b"})
	if a.ID != 1 || b.ID != 2 {
		t.Fatalf("IDs = %d,%d, want 1,2", a.ID, b.ID)
	}
	if a.Count != 1 {
		t.Errorf("Count = %d, want 1", a.Count)
	}
}

func TestAddCoalescesConsecutiveIdentical(t *testing.T) {
	l := New(10)
	l.Add(Notice{Severity: Warning, Message: "dup", Source: "lifecycle"})
	got := l.Add(Notice{Severity: Warning, Message: "dup", Source: "lifecycle"})
	if got.Count != 2 {
		t.Fatalf("Count = %d, want 2", got.Count)
	}
	if n := len(l.Entries()); n != 1 {
		t.Fatalf("Entries len = %d, want 1 (coalesced)", n)
	}
}

func TestAddDoesNotCoalesceDifferentSource(t *testing.T) {
	l := New(10)
	l.Add(Notice{Severity: Warning, Message: "dup", Source: "a"})
	l.Add(Notice{Severity: Warning, Message: "dup", Source: "b"})
	if n := len(l.Entries()); n != 2 {
		t.Fatalf("Entries len = %d, want 2", n)
	}
}

func TestEntriesNewestFirst(t *testing.T) {
	l := New(10)
	l.Add(Notice{Severity: Info, Message: "first"})
	l.Add(Notice{Severity: Info, Message: "second"})
	e := l.Entries()
	if e[0].Message != "second" || e[1].Message != "first" {
		t.Fatalf("order = %q,%q, want second,first", e[0].Message, e[1].Message)
	}
}

func TestRingEvictsOldest(t *testing.T) {
	l := New(2)
	l.Add(Notice{Severity: Info, Message: "1"})
	l.Add(Notice{Severity: Info, Message: "2"})
	l.Add(Notice{Severity: Info, Message: "3"})
	e := l.Entries()
	if len(e) != 2 {
		t.Fatalf("len = %d, want 2", len(e))
	}
	if e[0].Message != "3" || e[1].Message != "2" {
		t.Fatalf("retained = %q,%q, want 3,2", e[0].Message, e[1].Message)
	}
}

func TestLatest(t *testing.T) {
	l := New(5)
	if _, ok := l.Latest(); ok {
		t.Fatal("Latest() ok=true on empty log")
	}
	l.Add(Notice{Severity: Error, Message: "boom", Time: time.Unix(1, 0)})
	n, ok := l.Latest()
	if !ok || n.Message != "boom" {
		t.Fatalf("Latest() = %v,%v", n, ok)
	}
}
