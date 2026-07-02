package sparkline

import (
	"reflect"
	"testing"
)

func TestRingEmpty(t *testing.T) {
	r := NewRing(4)
	if r.Len() != 0 {
		t.Fatalf("Len = %d, want 0", r.Len())
	}
	if got := r.Values(); len(got) != 0 {
		t.Fatalf("Values = %v, want empty", got)
	}
	if r.Last() != 0 || r.Min() != 0 || r.Max() != 0 {
		t.Fatalf("empty-ring accessors must be 0")
	}
}

func TestRingPushWithinCapacity(t *testing.T) {
	r := NewRing(4)
	r.Push(1)
	r.Push(2)
	r.Push(3)
	if got, want := r.Values(), []float64{1, 2, 3}; !reflect.DeepEqual(got, want) {
		t.Fatalf("Values = %v, want %v", got, want)
	}
	if r.Last() != 3 {
		t.Fatalf("Last = %v, want 3", r.Last())
	}
	if r.Len() != 3 {
		t.Fatalf("Len = %d, want 3", r.Len())
	}
}

func TestRingWrapKeepsNewestInOrder(t *testing.T) {
	r := NewRing(3)
	for _, v := range []float64{1, 2, 3, 4, 5} {
		r.Push(v)
	}
	if got, want := r.Values(), []float64{3, 4, 5}; !reflect.DeepEqual(got, want) {
		t.Fatalf("Values = %v, want %v (oldest dropped, chronological)", got, want)
	}
	if r.Last() != 5 {
		t.Fatalf("Last = %v, want 5", r.Last())
	}
	if r.Min() != 3 || r.Max() != 5 {
		t.Fatalf("Min/Max = %v/%v, want 3/5", r.Min(), r.Max())
	}
}

func TestRingMinMaxNonMonotonic(t *testing.T) {
	r := NewRing(4)
	// Unsorted samples: the extrema are neither the first nor the last
	// element, so Min/Max must scan the whole series rather than trust
	// the seed value.
	for _, v := range []float64{5, 2, 8, 1} {
		r.Push(v)
	}
	if r.Min() != 1 {
		t.Fatalf("Min = %v, want 1", r.Min())
	}
	if r.Max() != 8 {
		t.Fatalf("Max = %v, want 8", r.Max())
	}
}

func TestNewRingClampsCapacity(t *testing.T) {
	r := NewRing(0)
	r.Push(7)
	if got, want := r.Values(), []float64{7}; !reflect.DeepEqual(got, want) {
		t.Fatalf("Values = %v, want %v", got, want)
	}
}
