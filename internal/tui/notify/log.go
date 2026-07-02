package notify

// Log is a bounded, in-memory ring of Notices. It is NOT safe for concurrent
// use; it is owned by the single-threaded Bubble Tea Update loop.
type Log struct {
	ring   []Notice
	cap    int
	nextID uint64
}

// New returns an empty Log holding at most capacity notices (capacity >= 1).
func New(capacity int) *Log {
	if capacity < 1 {
		capacity = 1
	}
	return &Log{cap: capacity}
}

// Add stores n, assigning a monotonic ID and Count >= 1. If n is identical
// (Severity+Message+Source) to the current newest notice, it coalesces: the
// existing entry's Count is incremented and its Time refreshed, and that entry
// is returned with its ID unchanged. Otherwise n is appended and the oldest
// entry is evicted past capacity.
func (l *Log) Add(n Notice) Notice {
	if len(l.ring) > 0 {
		last := &l.ring[len(l.ring)-1]
		if last.Severity == n.Severity && last.Message == n.Message && last.Source == n.Source {
			last.Count++
			if !n.Time.IsZero() {
				last.Time = n.Time
			}
			return *last
		}
	}
	l.nextID++
	n.ID = l.nextID
	if n.Count < 1 {
		n.Count = 1
	}
	l.ring = append(l.ring, n)
	if len(l.ring) > l.cap {
		l.ring = l.ring[len(l.ring)-l.cap:]
	}
	return n
}

// Entries returns all stored notices, newest first.
func (l *Log) Entries() []Notice {
	out := make([]Notice, len(l.ring))
	for i, n := range l.ring {
		out[len(l.ring)-1-i] = n
	}
	return out
}

// Latest returns the most recent notice, or ok=false when the log is empty.
func (l *Log) Latest() (Notice, bool) {
	if len(l.ring) == 0 {
		return Notice{}, false
	}
	return l.ring[len(l.ring)-1], true
}
