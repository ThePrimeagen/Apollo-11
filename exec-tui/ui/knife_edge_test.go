package ui

// t33 — the knife edge must be visible without reading the log: when
// SERVICER overruns are recovering every cycle (stubs never accumulate),
// an amber indicator says so and points at the controls that add load.

import (
	"strings"
	"testing"
)

func TestKnifeEdgeIndicator(t *testing.T) {
	t.Run("happy: recovering overruns show the knife-edge indicator", func(t *testing.T) {
		e, m := newTestModel()
		e.AdvanceAGC(170) // desync the 2s boundary from the 1Hz marks
		e.StartDescent()
		e.AcquireLandingRadar()
		e.SetRadarBug(true)
		e.AdvanceAGC(10000)
		if !strings.Contains(m.View(), "knife edge") {
			t.Fatal("recovering overruns must surface a knife-edge indicator")
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
