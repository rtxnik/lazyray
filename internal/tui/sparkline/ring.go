// Package sparkline renders a compact block-unicode trend line from a numeric
// series and provides a fixed-capacity ring buffer to hold the samples. It has
// no dependency on the TUI, so it is unit-testable in isolation.
package sparkline

// Ring is a fixed-capacity FIFO of float64 samples. Once full, each Push
// overwrites the oldest sample. The zero value is unusable; build with NewRing.
type Ring struct {
	buf  []float64
	next int // index of the next write
	size int // number of valid samples (<= cap)
}

// NewRing returns a ring that retains the most recent `capacity` samples.
// A capacity below 1 is clamped to 1.
func NewRing(capacity int) *Ring {
	if capacity < 1 {
		capacity = 1
	}
	return &Ring{buf: make([]float64, capacity)}
}

// Push appends v, overwriting the oldest sample when the ring is full.
func (r *Ring) Push(v float64) {
	r.buf[r.next] = v
	r.next = (r.next + 1) % len(r.buf)
	if r.size < len(r.buf) {
		r.size++
	}
}

// Len returns the number of valid samples currently held.
func (r *Ring) Len() int { return r.size }

// Values returns the held samples in chronological order (oldest first,
// newest last) as a fresh copy.
func (r *Ring) Values() []float64 {
	out := make([]float64, r.size)
	start := 0
	if r.size == len(r.buf) {
		start = r.next // when full, the oldest sample sits at next
	}
	for i := 0; i < r.size; i++ {
		out[i] = r.buf[(start+i)%len(r.buf)]
	}
	return out
}

// Last returns the most recent sample, or 0 if empty.
func (r *Ring) Last() float64 {
	if r.size == 0 {
		return 0
	}
	return r.buf[(r.next-1+len(r.buf))%len(r.buf)]
}

// Min returns the smallest held sample, or 0 if empty.
func (r *Ring) Min() float64 {
	if r.size == 0 {
		return 0
	}
	vals := r.Values()
	m := vals[0]
	for _, v := range vals[1:] {
		if v < m {
			m = v
		}
	}
	return m
}

// Max returns the largest held sample, or 0 if empty.
func (r *Ring) Max() float64 {
	if r.size == 0 {
		return 0
	}
	vals := r.Values()
	m := vals[0]
	for _, v := range vals[1:] {
		if v > m {
			m = v
		}
	}
	return m
}
