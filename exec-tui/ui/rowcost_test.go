package ui

// t51 — every timeline row carries a fixed-width parenthetical showing the
// AGC milliseconds that item consumed over the trailing 2-second cycle,
// space-padded with an explicit unit so the layout NEVER shifts:
// `SERVICER (1372ms)`. The old cycles/restarts counter line is gone.

import (
	"regexp"
	"strconv"
	"strings"
	"testing"
)

var costRE = regexp.MustCompile(`^([A-Z0-9 ]{9})\(( *\d+)ms\) `)

// rowCost extracts the parenthetical ms value for a labeled row.
func rowCost(t *testing.T, v, label string) int {
	t.Helper()
	for _, line := range strings.Split(v, "\n") {
		p := stripAnsi(line)
		m := costRE.FindStringSubmatch(p)
		if m == nil || !strings.HasPrefix(strings.TrimSpace(m[1]), label) {
			continue
		}
		n, err := strconv.Atoi(strings.TrimSpace(m[2]))
		if err != nil {
			t.Fatalf("bad cost %q on row %q", m[2], label)
		}
		return n
	}
	t.Fatalf("row %q with a (NNNN) cost not found", label)
	return -1
}

func TestRowCosts(t *testing.T) {
	t.Run("happy: every row shows a zero-filled 4-digit ms cost", func(t *testing.T) {
		e, m := newTestModel()
		e.AdvanceAGC(4100)
		v := m.View()
		for _, label := range []string{"SERVICER", "MONITOR", "CHARIN", "DAP", "RR STEAL", "IDLE"} {
			_ = rowCost(t, v, label)
		}
	})
	t.Run("happy: an idle machine charges almost everything to IDLE", func(t *testing.T) {
		e, m := newTestModel()
		e.AdvanceAGC(4100)
		v := m.View()
		if got := rowCost(t, v, "IDLE"); got < 1900 {
			t.Fatalf("idle should hold ~2000ms of the 2s window, got %d", got)
		}
		if got := rowCost(t, v, "MONITOR"); got != 0 {
			t.Fatalf("an off monitor must read (   0ms), got %d", got)
		}
	})
	t.Run("happy: the monitor at 1Hz reads ~60ms per 2s cycle", func(t *testing.T) {
		e, m := newTestModel()
		e.StartDescent()
		for _, k := range []byte("V16N68E") {
			e.PressKey(k)
		}
		e.AdvanceAGC(6100)
		got := rowCost(t, m.View(), "MONITOR")
		if got < 40 || got > 90 {
			t.Fatalf("V16N68 at 1Hz should need ~60ms per cycle, got %d", got)
		}
	})
	t.Run("happy: descent workloads read plausibly", func(t *testing.T) {
		e, m := newTestModel()
		e.StartDescent()
		e.SetRadarBug(true)
		e.AdvanceAGC(6100)
		v := m.View()
		if got := rowCost(t, v, "SERVICER"); got < 1000 {
			t.Fatalf("SERVICER should consume >1000ms per cycle, got %d", got)
		}
		if got := rowCost(t, v, "DAP"); got < 180 || got > 300 {
			t.Fatalf("the DAP should consume ~240ms per cycle, got %d", got)
		}
		if got := rowCost(t, v, "RR STEAL"); got < 250 || got > 350 {
			t.Fatalf("the theft should consume ~300ms per cycle, got %d", got)
		}
	})
	t.Run("unhappy: the counters line is gone", func(t *testing.T) {
		e, m := newTestModel()
		e.StartDescent()
		e.AdvanceAGC(4100)
		v := m.View()
		for _, gone := range []string{"cycles ", "restarts ", "copies "} {
			if strings.Contains(v, gone) {
				t.Fatalf("the counters line must be gone, found %q", gone)
			}
		}
	})
	t.Run("unhappy: the label column is fixed-width — costs never shift the track", func(t *testing.T) {
		e, m := newTestModel()
		e.StartDescent()
		e.AdvanceAGC(2100)
		v := m.View()
		start := -1
		for _, line := range strings.Split(v, "\n") {
			p := stripAnsi(line)
			if m := costRE.FindStringSubmatch(p); m != nil {
				if start == -1 {
					start = len(m[0])
				} else if len(m[0]) != start {
					t.Fatalf("label columns differ in width: %d vs %d", start, len(m[0]))
				}
			}
		}
		if start != 18 {
			t.Fatalf("label column must be exactly 18 cells (9 label + 8 cost + space), got %d", start)
		}
	})
}
