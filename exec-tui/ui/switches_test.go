package ui

// t45 — the control surface is now three cockpit switches on the bottom
// panel (the big story cards are gone):
//
//   DESCENT  (bottom left)  — flicks P63 on by keying V37E 63E on the DSKY
//   DELTAH   (next to it)   — Buzz's V16 N68 ΔH monitor
//   RR STEAL (bottom right) — the rendezvous-radar mode switch
//
// h/l move between them, space (or enter) flicks the focused one. Pause
// moved to '.'.

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/theprimeagen/apollo-11/exec-tui/sim"
)

func space(m Model) Model {
	mm, _ := m.Update(tea.KeyMsg{Type: tea.KeySpace})
	return mm.(Model)
}

func TestSwitchPanelRender(t *testing.T) {
	t.Run("happy: three labeled switches with their DSKY captions", func(t *testing.T) {
		_, m := newTestModel()
		v := m.View()
		for _, want := range []string{"DESCENT", "DELTAH", "RR STEAL"} {
			if !strings.Contains(v, want) {
				t.Fatalf("switch panel missing %q", want)
			}
		}
		for _, want := range []string{"V37E 63E", "V16 N68", "SLEW/AUTO"} {
			if !strings.Contains(v, want) {
				t.Fatalf("switch captions missing %q", want)
			}
		}
		if strings.Contains(v, "● ON") {
			t.Fatal("no switch may claim ON before anything is engaged")
		}
	})
	t.Run("happy: the big story cards are gone", func(t *testing.T) {
		_, m := newTestModel()
		v := m.View()
		if strings.Contains(v, "powered descent") || strings.Contains(v, "Buzz's monitor") {
			t.Fatal("the old card row must not render anymore")
		}
	})
	t.Run("unhappy: exactly one ON caption when only the RR switch is up", func(t *testing.T) {
		e, m := newTestModel()
		e.SetRadarBug(true)
		if got := strings.Count(m.View(), "● ON"); got != 1 {
			t.Fatalf("want exactly 1 ● ON, got %d", got)
		}
	})
}

func TestSwitchFlip(t *testing.T) {
	t.Run("happy: space on DESCENT keys V37E63E and P63 engages", func(t *testing.T) {
		e, m := newTestModel()
		m = space(m)
		if m.PendingKeys() < 7 {
			t.Fatalf("DESCENT should queue the 7 keystrokes of V37E 63E, got %d", m.PendingKeys())
		}
		e.SetWallToAGC(1.0)
		m = tick(m, 120)
		if e.Phase() != sim.P63 {
			t.Fatalf("after the keystrokes land the phase must be P63, got %v", e.Phase())
		}
	})
	t.Run("happy: space on DELTAH keys the monitor up, and V34 back down", func(t *testing.T) {
		e, m := newTestModel()
		m = key(m, 'l') // focus DELTAH
		m = space(m)
		if m.PendingKeys() < 7 {
			t.Fatalf("DELTAH should queue V16N68E, got %d pending", m.PendingKeys())
		}
		e.SetWallToAGC(1.0)
		m = tick(m, 120)
		if !e.MonitorActive() {
			t.Fatal("after V16N68E lands the monitor must refresh at 1Hz")
		}
		m = space(m) // flick it off: V34E terminates the monitor
		m = tick(m, 80)
		if e.MonitorActive() {
			t.Fatal("flicking DELTAH off must terminate the monitor via V34")
		}
	})
	t.Run("happy: space on RR STEAL flips the mode switch instantly, both ways", func(t *testing.T) {
		e, m := newTestModel()
		m = key(m, 'h') // wrap to RR STEAL
		m = space(m)
		if !e.RadarBug() {
			t.Fatal("RR STEAL must put the switch in SLEW/AUTO (theft on)")
		}
		if m.PendingKeys() != 0 {
			t.Fatal("the mode switch is a panel switch, not DSKY keystrokes")
		}
		m = space(m)
		if e.RadarBug() {
			t.Fatal("flicking again returns the switch to LGC (theft off)")
		}
	})
	t.Run("unhappy: DESCENT re-flick while running or mid-keying queues nothing", func(t *testing.T) {
		e, m := newTestModel()
		m = space(m)
		n := m.PendingKeys()
		m = space(m)
		if m.PendingKeys() != n {
			t.Fatalf("double flick must not re-queue keystrokes: %d -> %d", n, m.PendingKeys())
		}
		e.SetWallToAGC(1.0)
		m = tick(m, 120)
		m = space(m)
		if m.PendingKeys() != 0 {
			t.Fatal("flicking a running DESCENT must be a no-op")
		}
	})
	t.Run("unhappy: space in typing mode is not a switch flip", func(t *testing.T) {
		e, m := newTestModel()
		m = key(m, 't')
		m = space(m)
		if m.PendingKeys() != 0 || e.RadarBug() {
			t.Fatal("typing mode must swallow space without flipping switches")
		}
	})
}

func TestPauseMovedToDot(t *testing.T) {
	t.Run("happy: '.' pauses and resumes", func(t *testing.T) {
		e, m := newTestModel()
		m = key(m, '.')
		if !m.Paused() {
			t.Fatal("'.' must pause")
		}
		before := e.AGCTimeMs()
		m = tick(m, 10)
		if e.AGCTimeMs() != before {
			t.Fatal("paused ticks must not advance AGC time")
		}
		m = key(m, '.')
		if m.Paused() {
			t.Fatal("'.' again must resume")
		}
	})
	t.Run("unhappy: space no longer pauses", func(t *testing.T) {
		_, m := newTestModel()
		m = space(m)
		if m.Paused() {
			t.Fatal("space is a switch flip now, not pause")
		}
	})
}
