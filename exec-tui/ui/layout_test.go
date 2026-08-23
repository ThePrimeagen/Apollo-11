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
	t.Run("happy: the track is half the screen minus the label column", func(t *testing.T) {
		if got := trackWidth(140); got != 61 {
			t.Fatalf("trackWidth(140) = %d, want 61 (140/2 - 9)", got)
		}
		if got := trackWidth(152); got != 67 {
			t.Fatalf("trackWidth(152) = %d, want 67", got)
		}
	})
	t.Run("unhappy: tiny widths clamp to a usable minimum", func(t *testing.T) {
		if got := trackWidth(40); got != 20 {
			t.Fatalf("trackWidth(40) = %d, want the 20-cell floor", got)
		}
	})
	t.Run("happy: each cell covers 40ms and the header says so", func(t *testing.T) {
		_, m := newTestModel()
		v := m.View()
		if !strings.Contains(v, "40ms/cell") {
			t.Fatal("the track header must state 40ms per cell")
		}
		if !strings.Contains(v, "2.4s of AGC time") {
			t.Fatal("at width 140 the 61-cell track shows 2.4s of history")
		}
	})
}

func TestPoolsRowWise(t *testing.T) {
	t.Run("happy: pool titles and counts survive the move", func(t *testing.T) {
		e, m := newTestModel()
		e.StartDescent()
		e.AdvanceAGC(100)
		v := m.View()
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
		v := m.View()
		if strings.Contains(v, "READACCS: read PIPAs") || strings.Contains(v, "powered descent —") {
			t.Fatal("the event log must not render anymore")
		}
	})
	t.Run("unhappy: the knife-edge indicator still lives on the stats line", func(t *testing.T) {
		e, m := newTestModel()
		e.AdvanceAGC(170)
		e.StartDescent()
		e.AcquireLandingRadar()
		e.SetRadarBug(true)
		e.AdvanceAGC(10000)
		if !strings.Contains(m.View(), "knife edge") {
			t.Fatal("removing the log must not remove the knife-edge indicator")
		}
	})
}

func TestCompactHeight(t *testing.T) {
	t.Run("happy: the whole view fits in 35 lines", func(t *testing.T) {
		e, m := newTestModel()
		e.StartDescent()
		e.SetRadarBug(true)
		e.AdvanceAGC(5000)
		if got := len(strings.Split(m.View(), "\n")); got > 35 {
			t.Fatalf("compact layout must fit in 35 lines, got %d", got)
		}
	})
	t.Run("happy: the DSKY panel sits on the right edge", func(t *testing.T) {
		e, m := newTestModel()
		e.StartDescent()
		e.AdvanceAGC(100)
		// The VERB label lives only on the DSKY; it must start in the right
		// half of a 140-wide screen.
		for _, line := range strings.Split(m.View(), "\n") {
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
		v := m.View()
		for _, want := range []string{"SERVICER", "CORE SETS", "VAC AREAS", "PROG", "DESCENT", "RR STEAL"} {
			if !strings.Contains(v, want) {
				t.Fatalf("compact idle view missing %q", want)
			}
		}
	})
}

func TestTimelineCellGrouping(t *testing.T) {
	t.Run("happy: cells anchor to absolute 4-bucket groups (no re-pairing)", func(t *testing.T) {
		e, m := newTestModel()
		e.StartDescent()
		e.AdvanceAGC(2500)
		prev := rowCells(t, m.View(), "DAP")
		shifts := 0
		for i := 0; i < 40; i++ {
			e.AdvanceAGC(10) // one bucket: at 4 buckets/cell most frames hold still
			next := rowCells(t, m.View(), "DAP")
			if !stableStep(prev, next) {
				t.Fatalf("frame %d re-paired the row:\nprev %q\nnext %q", i, prev, next)
			}
			if string(next) != string(prev) {
				shifts++
			}
			prev = next
		}
		if shifts < 8 || shifts > 12 {
			t.Fatalf("40 buckets at 4/cell should shift ~10 times, got %d", shifts)
		}
	})
}
