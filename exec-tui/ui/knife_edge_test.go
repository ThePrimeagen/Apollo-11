package ui

// t33 — the knife edge must be visible without reading the log: when the
// theft has consumed the whole margin but nothing has overrun yet (the
// flight's quiet ~5 minutes before the monitor), an amber indicator says so
// and points at the controls that add the last straw.

import (
	"strings"
	"testing"
)

func TestKnifeEdgeIndicator(t *testing.T) {
	t.Run("happy: margin pinned at ~0 shows the knife-edge indicator", func(t *testing.T) {
		e, m := newTestModel()
		e.AdvanceAGC(170) // arbitrary start offset, as in interactive use
		e.StartDescent()
		e.AcquireLandingRadar()
		e.SetRadarBug(true)
		e.AdvanceAGC(10000)
		if !strings.Contains(m.View(), "knife edge") {
			t.Fatal("a zero-margin quiet regime must surface the knife-edge indicator")
		}
	})
	t.Run("unhappy: healthy descent shows no knife-edge indicator", func(t *testing.T) {
		e, m := newTestModel()
		e.StartDescent()
		e.AdvanceAGC(10000)
		if strings.Contains(m.View(), "knife edge") {
			t.Fatal("healthy descent must not claim a knife edge")
		}
	})
}
