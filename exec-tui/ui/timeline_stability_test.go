package ui

// t28 — no blinking: a timeline cell, once rendered, must never change its
// content. Between two frames a row is either identical or shifted left by
// exactly one cell (with fresh content only in the rightmost cell). The old
// "most recent N buckets" pairing re-shuffled every 10ms bucket close, making
// DAP/GYRO blocks flicker between '█' and '█░' shapes.

import (
	"strings"
	"testing"
)

// rowCells extracts the cell area of a labeled timeline row from the view,
// as runes ('█'/'░' are multi-byte). At the test width of 140: 9-rune label
// column, then trackW=73 cells (the DSKY panel and core-set boxes joined to
// the right of the row are not part of the track).
func rowCells(t *testing.T, v, label string) []rune {
	t.Helper()
	const labelW, trackW = 9, 73
	for _, line := range strings.Split(v, "\n") {
		if strings.HasPrefix(line, label) {
			r := []rune(line)
			if len(r) < labelW+trackW {
				t.Fatalf("row %q too short: %q", label, line)
			}
			return r[labelW : labelW+trackW]
		}
	}
	t.Fatalf("row %q not found", label)
	return nil
}

// stableStep reports whether next is a legal successor of prev: identical,
// or shifted left by one cell with only the rightmost cell new.
func stableStep(prev, next []rune) bool {
	if string(prev) == string(next) {
		return true
	}
	if len(prev) != len(next) || len(prev) < 2 {
		return false
	}
	return string(next[:len(next)-1]) == string(prev[1:])
}

func TestTimelineNoBlinking(t *testing.T) {
	t.Run("happy: DAP row never re-pairs across 10ms frames", func(t *testing.T) {
		e, m := newTestModel()
		e.StartDescent()
		e.AdvanceAGC(2500) // fill the track
		prev := rowCells(t, m.View(), "DAP")
		shifts := 0
		for i := 0; i < 30; i++ {
			e.AdvanceAGC(10) // exactly one bucket
			next := rowCells(t, m.View(), "DAP")
			if !stableStep(prev, next) {
				t.Fatalf("frame %d re-paired the row:\nprev %q\nnext %q", i, prev, next)
			}
			if string(next) != string(prev) {
				shifts++
			}
			prev = next
		}
		if shifts == 0 {
			t.Fatal("the track never advanced — test is vacuous")
		}
	})
	t.Run("unhappy: young history is padded and just as stable", func(t *testing.T) {
		e, m := newTestModel()
		e.StartDescent()
		e.AdvanceAGC(300)
		prev := rowCells(t, m.View(), "SERVICER")
		for i := 0; i < 20; i++ {
			e.AdvanceAGC(10)
			next := rowCells(t, m.View(), "SERVICER")
			if !stableStep(prev, next) {
				t.Fatalf("young-history frame %d re-paired:\nprev %q\nnext %q", i, prev, next)
			}
			prev = next
		}
	})
}
