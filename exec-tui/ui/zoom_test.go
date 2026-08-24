package ui

// t50 — timeline zoom: the track shrinks by holding the visible window
// (~2.4s at width 140) constant while each bar covers more time.
//   default: 50ms/bar (25% narrower than the old 40ms)
//   z key:   cycles 50 → 80 (50% narrower) → 40 → 50 …

import (
	"testing"
)

func TestZoomLevels(t *testing.T) {
	t.Run("happy: default is 50ms per bar", func(t *testing.T) {
		_, m := newTestModel()
		if got := m.cellMs(); got != 50 {
			t.Fatalf("default zoom must be 50ms/bar, got %v", got)
		}
	})
	t.Run("happy: z cycles 50 -> 80 -> 40 -> 50", func(t *testing.T) {
		_, m := newTestModel()
		m = key(m, 'z')
		if got := m.cellMs(); got != 80 {
			t.Fatalf("first z must give 80ms bars, got %v", got)
		}
		m = key(m, 'z')
		if got := m.cellMs(); got != 40 {
			t.Fatalf("second z must give 40ms bars, got %v", got)
		}
		m = key(m, 'z')
		if got := m.cellMs(); got != 50 {
			t.Fatalf("third z must wrap back to 50ms bars, got %v", got)
		}
	})
	t.Run("happy: the window stays ~2.4s while the track narrows", func(t *testing.T) {
		if got := cellsFor(140, 5); got != 48 {
			t.Fatalf("50ms bars at width 140: want 48 cells, got %d", got)
		}
		if got := cellsFor(140, 8); got != 30 {
			t.Fatalf("80ms bars at width 140: want 30 cells, got %d", got)
		}
		if got := cellsFor(140, 4); got != 61 {
			t.Fatalf("40ms bars at width 140: want 61 cells, got %d", got)
		}
	})
	t.Run("unhappy: tiny widths still clamp to a usable minimum", func(t *testing.T) {
		if got := cellsFor(40, 8); got != 20 {
			t.Fatalf("cellsFor(40,8) = %d, want the 20-cell floor", got)
		}
	})
	t.Run("happy: the 2s ruler spacing follows the zoom", func(t *testing.T) {
		withColor(t)
		e, m := newTestModel()
		e.AdvanceAGC(6100)
		cols := markerCols(t, m.View(), "IDLE")
		if len(cols) >= 2 {
			if d := cols[1] - cols[0]; d != 40 {
				t.Fatalf("at 50ms/bar the ruler must sit 40 cells apart, got %d", d)
			}
		} else if len(cols) == 0 {
			t.Fatal("the ruler must be visible")
		}
	})
}
