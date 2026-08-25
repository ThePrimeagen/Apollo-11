package ui

// t47 — the compact layout refactor:
//   - the busy-work timelines take HALF the screen width; each cell now
//     covers 4 history buckets (40ms) so the visible window stays ~2.4s
//   - the event log is gone from the view (the engine still records events)
//   - the core-set and VAC pools render ROW-wise beneath the timelines,
//     where the log used to live
//   - the DSKY panel sits on the complete right side
//   - net effect: the whole view fits in ~36 rows instead of ~44

import (
	"regexp"
	"strings"
	"testing"
)

var ansiPat = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func stripAnsi(s string) string { return ansiPat.ReplaceAllString(s, "") }

func TestTimelineHalfWidth(t *testing.T) {
	t.Run("happy: the default track is 48 bars of 50ms (~2.4s) at width 140", func(t *testing.T) {
		if got := cellsFor(140, 5); got != 48 {
			t.Fatalf("cellsFor(140,5) = %d, want 48", got)
		}
	})
}

func TestPoolsRowWise(t *testing.T) {
	t.Run("happy: pool titles and counts survive the move", func(t *testing.T) {
		e, m := newTestModel()
		e.StartDescent()
		e.AdvanceAGC(100)
		v := m.View().Content
		if !strings.Contains(v, "CORE SETS") || !strings.Contains(v, "VAC AREAS") {
			t.Fatal("pool titles must render")
		}
		if !strings.Contains(v, "1/8") || !strings.Contains(v, "1/5") {
			t.Fatal("pool occupancy counts must render (SERVICER holds one pair)")
		}
	})
}

func TestNoEventLog(t *testing.T) {
	t.Run("happy: the view renders no event-log lines", func(t *testing.T) {
		e, m := newTestModel()
		e.StartDescent()
		e.AdvanceAGC(4100)
		if len(e.Events()) == 0 {
			t.Fatal("the engine must still record events")
		}
		v := m.View().Content
		if strings.Contains(v, "READACCS: read PIPAs") || strings.Contains(v, "powered descent —") {
			t.Fatal("the event log must not render anymore")
		}
	})
	t.Run("unhappy: the knife edge shows as a near-zero free number, no text", func(t *testing.T) {
		e, m := newTestModel()
		e.AdvanceAGC(170)
		e.StartDescent()
		e.AcquireLandingRadar()
		e.SetRadarBug(true)
		e.AdvanceAGC(10000)
		if !e.KnifeEdge() {
			t.Fatal("the engine must still report the knife edge")
		}
		if strings.Contains(m.View().Content, "knife edge") {
			t.Fatal("no knife-edge text may render — the number tells it")
		}
	})
}

func TestCompactHeight(t *testing.T) {
	t.Run("happy: the whole view fits in 33 lines", func(t *testing.T) {
		e, m := newTestModel()
		e.StartDescent()
		e.SetRadarBug(true)
		e.AdvanceAGC(5000)
		if got := len(strings.Split(m.View().Content, "\n")); got > 33 {
			t.Fatalf("compact layout must fit in 33 lines, got %d", got)
		}
	})
	t.Run("happy: the DSKY panel sits on the right edge", func(t *testing.T) {
		e, m := newTestModel()
		e.StartDescent()
		e.AdvanceAGC(100)
		// The VERB label lives only on the DSKY; it must start in the right
		// half of a 140-wide screen.
		for _, line := range strings.Split(m.View().Content, "\n") {
			if i := strings.Index(stripAnsi(line), "VERB"); i >= 0 {
				if i < 100 {
					t.Fatalf("DSKY must hug the right edge, VERB at col %d", i)
				}
				return
			}
		}
		t.Fatal("VERB label not found")
	})
	t.Run("unhappy: phase P00 still renders the full structure", func(t *testing.T) {
		_, m := newTestModel()
		v := m.View().Content
		for _, want := range []string{"SERVICER", "CORE SETS", "VAC AREAS", "PROG", "DESCENT", "RR STEAL"} {
			if !strings.Contains(v, want) {
				t.Fatalf("compact idle view missing %q", want)
			}
		}
	})
}

func TestTimelineCellGrouping(t *testing.T) {
	t.Run("happy: cells anchor to absolute 5-bucket groups (no re-pairing)", func(t *testing.T) {
		e, m := newTestModel()
		e.StartDescent()
		e.AdvanceAGC(2500)
		prev := rowCells(t, m.View().Content, "DAP")
		shifts := 0
		for i := 0; i < 40; i++ {
			e.AdvanceAGC(10) // one bucket: at 5 buckets/cell most frames hold still
			next := rowCells(t, m.View().Content, "DAP")
			if !stableStep(prev, next) {
				t.Fatalf("frame %d re-paired the row:\nprev %q\nnext %q", i, prev, next)
			}
			if string(next) != string(prev) {
				shifts++
			}
			prev = next
		}
		if shifts < 7 || shifts > 9 {
			t.Fatalf("40 buckets at 5/cell should shift 8 times, got %d", shifts)
		}
	})
}
