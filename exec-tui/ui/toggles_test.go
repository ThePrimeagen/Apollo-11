package ui

// t43 — the simplified story interface: three toggle cards that carry the
// whole Apollo 11 overload narrative, selected with h/l (vim-style) and
// engaged with Enter.
//
//   1. SERVICER    — the descent guidance cycle (keys V37E 63E on the DSKY)
//   2. RR SLEW/AUTO — the rendezvous-radar mode switch (the 15% theft)
//   3. V16 N68     — Buzz's DELTAH monitor (the last ~3% straw)

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/theprimeagen/apollo-11/exec-tui/sim"
)

func enter(m Model) Model {
	mm, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	return mm.(Model)
}

func TestToggleCardsRender(t *testing.T) {
	t.Run("happy: all three cards render with explanations", func(t *testing.T) {
		_, m := newTestModel()
		v := m.View()
		for _, want := range []string{"SERVICER", "RR SLEW/AUTO", "V16 N68"} {
			if !strings.Contains(v, want) {
				t.Fatalf("toggle cards missing %q", want)
			}
		}
		// one-line explanations the video leans on
		for _, want := range []string{"every 2", "6,400", "DELTAH"} {
			if !strings.Contains(v, want) {
				t.Fatalf("card explanations missing %q", want)
			}
		}
	})
	t.Run("happy: cards show OFF initially and ON once engaged", func(t *testing.T) {
		e, m := newTestModel()
		if v := m.View(); strings.Count(v, "● ON") != 0 {
			t.Fatal("no card may claim ON before anything is engaged")
		}
		e.SetRadarBug(true)
		if v := m.View(); strings.Count(v, "● ON") != 1 {
			t.Fatal("the RR card must show ON while the mode switch is in SLEW/AUTO")
		}
	})
}

func TestToggleSelection(t *testing.T) {
	t.Run("happy: l moves right, h moves left, both wrap", func(t *testing.T) {
		_, m := newTestModel()
		if m.Selected() != 0 {
			t.Fatalf("selection starts on SERVICER, got %d", m.Selected())
		}
		m = key(m, 'l')
		if m.Selected() != 1 {
			t.Fatalf("l should select the RR card, got %d", m.Selected())
		}
		m = key(m, 'l')
		m = key(m, 'l')
		if m.Selected() != 0 {
			t.Fatalf("l past the last card wraps to the first, got %d", m.Selected())
		}
		m = key(m, 'h')
		if m.Selected() != 2 {
			t.Fatalf("h from the first card wraps to the last, got %d", m.Selected())
		}
	})
	t.Run("happy: the selected card is visibly marked", func(t *testing.T) {
		_, m := newTestModel()
		if !strings.Contains(m.View(), "▸") {
			t.Fatal("the selected card must carry a visible marker")
		}
	})
	t.Run("unhappy: h/l in typing mode are DSKY input, not selection", func(t *testing.T) {
		_, m := newTestModel()
		m = key(m, 't')
		before := m.Selected()
		m = key(m, 'l')
		if m.Selected() != before {
			t.Fatal("typing mode must not move the card selection")
		}
	})
}

func TestToggleEngage(t *testing.T) {
	t.Run("happy: Enter on SERVICER types V37E63E and the descent starts", func(t *testing.T) {
		e, m := newTestModel()
		m = enter(m)
		if m.PendingKeys() < 7 {
			t.Fatalf("engaging SERVICER should queue the 7 keystrokes of V37E 63E, got %d", m.PendingKeys())
		}
		e.SetWallToAGC(1.0)
		m = tick(m, 120)
		if e.Phase() != sim.P63 {
			t.Fatalf("after the keystrokes land the phase must be P63, got %v", e.Phase())
		}
	})
	t.Run("happy: Enter on RR SLEW/AUTO flips the mode switch instantly, and back", func(t *testing.T) {
		e, m := newTestModel()
		m = key(m, 'l')
		m = enter(m)
		if !e.RadarBug() {
			t.Fatal("engaging the RR card must put the switch in SLEW/AUTO (theft on)")
		}
		if m.PendingKeys() != 0 {
			t.Fatal("the mode switch is a panel switch, not DSKY keystrokes")
		}
		m = enter(m)
		if e.RadarBug() {
			t.Fatal("engaging again returns the switch to LGC (theft off)")
		}
	})
	t.Run("happy: Enter on V16 N68 types the monitor request", func(t *testing.T) {
		e, m := newTestModel()
		m = key(m, 'h') // wrap to the V16N68 card
		m = enter(m)
		if m.PendingKeys() < 7 {
			t.Fatalf("engaging V16N68 should queue its 7 keystrokes, got %d", m.PendingKeys())
		}
		e.SetWallToAGC(1.0)
		m = tick(m, 120)
		if !e.MonitorActive() {
			t.Fatal("after V16N68E lands the monitor must refresh at 1Hz")
		}
	})
	t.Run("unhappy: Enter on an already-running SERVICER queues nothing", func(t *testing.T) {
		e, m := newTestModel()
		e.StartDescent()
		m = enter(m)
		if m.PendingKeys() != 0 {
			t.Fatalf("re-engaging a running SERVICER must be a no-op, got %d pending keys", m.PendingKeys())
		}
	})
	t.Run("unhappy: Enter with keystrokes already in flight queues nothing more", func(t *testing.T) {
		_, m := newTestModel()
		m = enter(m)
		n := m.PendingKeys()
		m = enter(m)
		if m.PendingKeys() != n {
			t.Fatalf("double engage must not re-queue keystrokes: %d -> %d", n, m.PendingKeys())
		}
	})
}
