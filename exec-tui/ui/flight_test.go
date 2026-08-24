package ui

// t52 — the historical recreation: press 'f' and the descent flies itself.
// The flight plan fires the real engine actions at the real mission moments
// (theft on from PDI, radar lock at T+274, Buzz's V16N68 at T+304 and again
// at T+346 after the restart sheds it, P64 at T+506, ATT HOLD at T+603, P66
// at T+615) — and the alarms are NOT scripted: they emerge from the engine's
// own arithmetic, landing on the flight's timing. The lander panel renders
// in the center gap (when the terminal is wide enough), fed by the engine.

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/theprimeagen/apollo-11/exec-tui/sim"
)

// newWideTestModel gives the layout room for the center lander panel.
func newWideTestModel() (*sim.Engine, Model) {
	e := sim.New()
	m := NewModel(e)
	mm, _ := m.Update(tea.WindowSizeMsg{Width: 170, Height: 46})
	return e, mm.(Model)
}

// flyTo drives the UI frame loop until the engine clock reaches missionSec.
func flyTo(t *testing.T, e *sim.Engine, m Model, missionSec float64) Model {
	t.Helper()
	for i := 0; i < 200000 && e.AGCTimeMs() < missionSec*1000; i++ {
		mm, _ := m.Update(FrameMsg{})
		m = mm.(Model)
	}
	if e.AGCTimeMs() < missionSec*1000 {
		t.Fatalf("frame loop stalled at %.1fs", e.AGCTimeMs()/1000)
	}
	return m
}

func TestFlightPlan(t *testing.T) {
	t.Run("happy: f flies the whole arc — theft, radar, monitor, P64, P66", func(t *testing.T) {
		e, m := newWideTestModel()
		m = key(m, 'f')
		if e.Phase() != sim.P63 || !e.RadarBug() {
			t.Fatal("PDI must start P63 with the RR theft already active")
		}
		if e.LandingRadarAcquired() {
			t.Fatal("the landing radar must not be locked at ignition")
		}
		m = flyTo(t, e, m, 280)
		if !e.LandingRadarAcquired() {
			t.Fatal("radar lock must arrive on its own by T+280")
		}
		if len(e.Alarms()) != 0 {
			t.Fatal("no alarms may fire before the monitor goes up")
		}
		m = flyTo(t, e, m, 340)
		if len(e.Alarms()) < 1 {
			t.Fatal("the first alarm must emerge by T+340")
		}
		first := e.Alarms()[0].AGCTimeMs / 1000
		if first < 306 || first > 335 {
			t.Fatalf("the first alarm must land near the flight's T+316, got T+%.0f", first)
		}
		m = flyTo(t, e, m, 380)
		if len(e.Alarms()) < 2 {
			t.Fatal("re-keying the monitor must draw the second alarm by T+380")
		}
		m = flyTo(t, e, m, 510)
		if e.Phase() != sim.P64 {
			t.Fatalf("high gate at T+506 must enter P64, got %v", e.Phase())
		}
		m = flyTo(t, e, m, 640)
		if e.Phase() != sim.P66 {
			t.Fatalf("ATT HOLD/P66 must engage by T+640, got %v", e.Phase())
		}
		if len(e.Alarms()) < 3 {
			t.Fatalf("P64 must add alarms, got %d total", len(e.Alarms()))
		}
	})
	t.Run("unhappy: f is a no-op when a flight is already running", func(t *testing.T) {
		e, m := newWideTestModel()
		m = key(m, 'f')
		m = flyTo(t, e, m, 5)
		before := e.AGCTimeMs()
		m = key(m, 'f')
		if e.AGCTimeMs() != before || e.Phase() != sim.P63 {
			t.Fatal("a second f must not restart the flight")
		}
	})
	t.Run("happy: f speeds time up for the replay, and ] clamps higher now", func(t *testing.T) {
		e, m := newWideTestModel()
		m = key(m, 'f')
		if e.WallToAGC() < 4 {
			t.Fatalf("the replay must compress time, got %vx", e.WallToAGC())
		}
		for i := 0; i < 12; i++ {
			m = key(m, ']')
		}
		if e.WallToAGC() > 16 {
			t.Fatalf("the speed cap is 16x, got %v", e.WallToAGC())
		}
	})
}

func TestLanderPanel(t *testing.T) {
	t.Run("happy: the flight shows the lander in the center gap", func(t *testing.T) {
		e, m := newWideTestModel()
		m = key(m, 'f')
		m = flyTo(t, e, m, 3)
		v := m.View()
		if !strings.Contains(v, "ft/s") {
			t.Fatal("the lander telemetry must render during the flight")
		}
		if !strings.Contains(stripAnsi(v), "▼") {
			t.Fatal("the touchdown countdown must render during the flight")
		}
	})
	t.Run("happy: the lander carries the ENGINE's real alarms as markers", func(t *testing.T) {
		e, m := newWideTestModel()
		m = key(m, 'f')
		m = flyTo(t, e, m, 340)
		if len(e.Alarms()) == 0 {
			t.Fatal("precondition: an alarm must have fired")
		}
		v := stripAnsi(m.View())
		if !strings.Contains(v, "◄ 120") {
			t.Fatal("the engine's alarm must appear as a marker on the descent")
		}
	})
	t.Run("unhappy: no lander outside a flight, and none on narrow screens", func(t *testing.T) {
		_, m := newWideTestModel()
		if strings.Contains(m.View(), "ft/s") {
			t.Fatal("no lander panel before a flight starts")
		}
		e2 := sim.New()
		m2 := NewModel(e2)
		mm, _ := m2.Update(tea.WindowSizeMsg{Width: 120, Height: 46})
		m2 = mm.(Model)
		m2 = key(m2, 'f')
		m2 = flyTo(t, e2, m2, 3)
		if strings.Contains(m2.View(), "ft/s") {
			t.Fatal("narrow terminals must skip the lander panel instead of wrapping")
		}
	})
}
