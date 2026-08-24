package ui

// t25 — the leak must be VISIBLE: abandoned SERVICER stubs render distinctly
// in the core set / VAC boxes, and the stats line calls out leaked memory.

import (
	"strings"
	"testing"
)

func overloadedModel(t *testing.T) Model {
	t.Helper()
	e, m := newTestModel()
	e.StartDescent()
	e.AcquireLandingRadar()
	e.SetRadarBug(true)
	for _, k := range []byte("V16N68E") {
		e.PressKey(k)
	}
	e.AdvanceAGC(6500) // >=2 abandoned stubs by now
	return m
}

func TestStubVisibility(t *testing.T) {
	t.Run("happy: abandoned slots render a STUB marker", func(t *testing.T) {
		m := overloadedModel(t)
		v := m.View()
		if !strings.Contains(v, "STUB") {
			t.Fatal("abandoned SERVICER slots must render as STUB")
		}
	})
	t.Run("unhappy: healthy descent shows no stub markers", func(t *testing.T) {
		e, m := newTestModel()
		e.StartDescent()
		e.AdvanceAGC(6500)
		if strings.Contains(m.View(), "STUB") {
			t.Fatal("healthy descent must not show stub markers")
		}
	})
}

// t26 — timeline cells use shades, never vertically-cut-off blocks. A '█'
// next to a '▂' composited into boot-shaped artifacts on screen; brief work
// must render as a full-cell '░' shade instead.

func TestTimelineShades(t *testing.T) {
	t.Run("happy: brief work paints a '░' shade cell on its row", func(t *testing.T) {
		e, m := newTestModel()
		e.AdvanceAGC(500) // idle: T4RUPT/DOWNLINK run briefly, never dominate
		v := m.View()
		found := false
		for _, line := range strings.Split(v, "\n") {
			if strings.Contains(line, "DOWNLINK") && strings.Contains(line, "░") {
				found = true
			}
		}
		if !found {
			t.Fatal("briefly-running DOWNLINK must paint '░' shade cells on its row")
		}
	})
	t.Run("unhappy: no vertically-cut '▂' cells anywhere, even under load", func(t *testing.T) {
		e, m := newTestModel()
		e.StartDescent()
		e.AcquireLandingRadar()
		e.SetRadarBug(true)
		e.AdvanceAGC(6500)
		v := m.View()
		if strings.Contains(v, "▂") {
			t.Fatal("timeline must not use vertically-cut-off '▂' cells")
		}
	})
}
