package sim

// t27 — the UI needs a monotonic count of closed history buckets so it can
// anchor its 2-buckets-per-cell pairing to ABSOLUTE bucket parity. Without
// it, pairs re-shuffle every 10ms and timeline blocks flicker.

import "testing"

func TestBucketsClosed(t *testing.T) {
	t.Run("happy: counts one bucket per 10ms of AGC time", func(t *testing.T) {
		e := New()
		if got := e.BucketsClosed(); got != 0 {
			t.Fatalf("fresh engine should have 0 closed buckets, got %d", got)
		}
		e.AdvanceAGC(100)
		if got := e.BucketsClosed(); got != 10 {
			t.Fatalf("100ms should close 10 buckets, got %d", got)
		}
		e.AdvanceAGC(15)
		if got := e.BucketsClosed(); got != 11 {
			t.Fatalf("115ms should close 11 buckets, got %d", got)
		}
	})
	t.Run("unhappy: keeps counting past the ring capacity", func(t *testing.T) {
		e := New()
		e.AdvanceAGC(12500) // 1250 buckets > historySize (1200)
		got := e.BucketsClosed()
		if got != 1250 {
			t.Fatalf("count must be monotonic past the ring size, got %d", got)
		}
		if n := len(e.History(2000)); n > got {
			t.Fatalf("History cannot return more than closed count: %d > %d", n, got)
		}
	})
}
