package main

// Demo harness tests, written before the implementation. The demo replays
// the descent on a lone DSKY:
//
//   d — the crew keys V37E 63E; P63 engages; V06 N63 shows velocity /
//       altitude-rate / altitude ticking once per second; the LM's ALT+VEL
//       radar lights burn until the landing radar locks, then extinguish
//   n — Buzz keys V16N68E; the monitor refreshes R1 slant range, R2
//       time-to-go, R3 DELTAH at 1Hz with COMP ACTY pulsing
//   a — executive overflow: PROG lamp lights, RESTART flashes, the
//       unprotected monitor is dropped and the display reverts to V06 N63
//   r — RSET clears the caution lamps
//   0-9 — press the matching keypad key, the way typing mode does in the
//       Executive sim; digits being keyed turn dull orange

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/theprimeagen/apollo-11/dsky-lab/dsky"
)

func TestDemoBoot(t *testing.T) {
	t.Run("happy: boots to a blank, dark DSKY", func(t *testing.T) {
		d := newDemo()
		s := d.state()
		if s.Prog != "" || s.Verb != "" || s.R3 != "" {
			t.Fatalf("boot must be blank, got %+v", s)
		}
		if s.Lights.Prog || s.Lights.Alt {
			t.Fatal("no lights at boot")
		}
	})
	t.Run("unhappy: unknown keys change nothing", func(t *testing.T) {
		d := newDemo()
		d.press('z')
		d.advance(2000)
		if s := d.state(); s.Prog != "" || s.Verb != "" {
			t.Fatal("unknown keys must be no-ops")
		}
	})
}

func TestDemoDescent(t *testing.T) {
	t.Run("happy: d keys V37E63E and P63 engages with live N63 values", func(t *testing.T) {
		d := newDemo()
		d.press('d')
		d.advance(400)
		if s := d.state(); s.Verb != "37" {
			t.Fatalf("the keystrokes must appear as typed, verb = %q", s.Verb)
		}
		d.advance(3000)
		s := d.state()
		if s.Prog != "63" || s.Verb != "06" || s.Noun != "63" {
			t.Fatalf("P63 must engage with V06 N63, got P%s V%s N%s", s.Prog, s.Verb, s.Noun)
		}
		r3a := s.R3
		d.advance(2000)
		if d.state().R3 == r3a {
			t.Fatal("the altitude in R3 must tick as the descent runs")
		}
		if !d.state().Lights.Alt || !d.state().Lights.Vel {
			t.Fatal("ALT and VEL burn until the landing radar locks")
		}
		d.advance(8000)
		if d.state().Lights.Alt || d.state().Lights.Vel {
			t.Fatal("radar lock must extinguish ALT and VEL")
		}
	})
	t.Run("unhappy: d twice does not restart the descent", func(t *testing.T) {
		d := newDemo()
		d.press('d')
		d.advance(4000)
		alt := d.state().R3
		d.press('d')
		d.advance(1000)
		if s := d.state(); s.Verb == "37" {
			t.Fatal("a running descent must ignore a second d")
		}
		if d.state().R3 == alt {
			t.Fatal("the descent must keep running")
		}
	})
}

func TestDemoMonitor(t *testing.T) {
	ready := func() *demo {
		d := newDemo()
		d.press('d')
		d.advance(12000) // descent running, radar locked
		return d
	}
	t.Run("happy: n keys V16N68E and the monitor refreshes at 1Hz", func(t *testing.T) {
		d := ready()
		d.press('n')
		d.advance(3000)
		s := d.state()
		if s.Verb != "16" || s.Noun != "68" {
			t.Fatalf("the monitor must show V16 N68, got V%s N%s", s.Verb, s.Noun)
		}
		if !strings.HasPrefix(s.R3, "-") {
			t.Fatalf("R3 must show a negative DELTAH, got %q", s.R3)
		}
		before := s
		d.advance(1000)
		after := d.state()
		if before.R1 == after.R1 && before.R2 == after.R2 && before.R3 == after.R3 {
			t.Fatal("the monitor must refresh its registers every second")
		}
	})
	t.Run("happy: COMP ACTY pulses with each refresh", func(t *testing.T) {
		d := ready()
		d.press('n')
		d.advance(2000)
		sawOn, sawOff := false, false
		for i := 0; i < 20; i++ {
			d.advance(100)
			if d.state().CompActy {
				sawOn = true
			} else {
				sawOff = true
			}
		}
		if !sawOn || !sawOff {
			t.Fatalf("COMP ACTY must pulse (on=%v off=%v)", sawOn, sawOff)
		}
	})
	t.Run("unhappy: n before the descent does nothing", func(t *testing.T) {
		d := newDemo()
		d.press('n')
		d.advance(3000)
		if s := d.state(); s.Verb != "" {
			t.Fatal("the monitor needs a running computer")
		}
	})
}

func TestDemoAlarm(t *testing.T) {
	overloaded := func() *demo {
		d := newDemo()
		d.press('d')
		d.advance(12000)
		d.press('n')
		d.advance(3000)
		return d
	}
	t.Run("happy: a lights PROG, drops the monitor, reverts to V06 N63", func(t *testing.T) {
		d := overloaded()
		d.press('a')
		d.advance(500)
		s := d.state()
		if !s.Lights.Prog {
			t.Fatal("the PROG caution lamp must light")
		}
		if s.Verb != "06" || s.Noun != "63" {
			t.Fatalf("the restart must revert the display to V06 N63, got V%s N%s", s.Verb, s.Noun)
		}
		d.advance(3000)
		if !d.state().Lights.Prog {
			t.Fatal("PROG stays lit until RSET")
		}
		if d.state().Lights.Restart {
			t.Fatal("the RESTART flash must clear on its own")
		}
	})
	t.Run("happy: r is RSET — the caution lamps clear", func(t *testing.T) {
		d := overloaded()
		d.press('a')
		d.advance(500)
		d.press('r')
		if d.state().Lights.Prog {
			t.Fatal("RSET must clear the PROG lamp")
		}
	})
	t.Run("unhappy: a before the descent is a no-op", func(t *testing.T) {
		d := newDemo()
		d.press('a')
		d.advance(500)
		if d.state().Lights.Prog {
			t.Fatal("no alarm without a running computer")
		}
	})
}

func TestDemoKeypad(t *testing.T) {
	t.Run("happy: pressing 1 depresses the 1 key like the program", func(t *testing.T) {
		d := newDemo()
		d.press('1')
		s := d.state()
		if s.Pressed != '1' {
			t.Fatalf("pressing 1 must hold the 1 key, got %q", s.Pressed)
		}
		out := dsky.Render(s, true)
		if !strings.Contains(out, "48;5;172") {
			t.Fatal("the 1 key must light dull orange while held")
		}
	})
	t.Run("happy: pressing 2 depresses the 2 key like the program", func(t *testing.T) {
		d := newDemo()
		d.press('2')
		if d.state().Pressed != '2' {
			t.Fatalf("pressing 2 must hold the 2 key, got %q", d.state().Pressed)
		}
		d.press('1')
		if d.state().Pressed != '1' {
			t.Fatal("a new press must move the highlight to that key")
		}
	})
	t.Run("happy: V then 1 types an orange digit onto VERB", func(t *testing.T) {
		d := newDemo()
		d.press('v')
		d.press('1')
		s := d.state()
		if s.Verb != "1" {
			t.Fatalf("V then 1 must type a 1, got verb %q", s.Verb)
		}
		if !s.Typing.Verb {
			t.Fatal("the verb must be marked as being typed")
		}
		out := dsky.Render(s, true)
		if !strings.Contains(out, "38;5;172") {
			t.Fatal("the typed digit must render dull orange")
		}
	})
	t.Run("happy: the demo view shows the keypad ready to press", func(t *testing.T) {
		m := model{d: newDemo()}
		v := m.View().Content
		for _, want := range []string{"ENTR", "CLR", "1", "2", "VERB", "NOUN"} {
			if !strings.Contains(v, want) {
				t.Fatalf("demo view missing keypad %q", want)
			}
		}
	})
	t.Run("happy: the house keyboard 1/2 reach the keypad through Update", func(t *testing.T) {
		m := model{d: newDemo()}
		mm, _ := m.Update(tea.KeyPressMsg{Code: '1', Text: "1"})
		m = mm.(model)
		if m.d.state().Pressed != '1' {
			t.Fatal("the demo scene must press 1 when the 1 key is hit")
		}
		mm, _ = m.Update(tea.KeyPressMsg{Code: '2', Text: "2"})
		m = mm.(model)
		if m.d.state().Pressed != '2' {
			t.Fatal("the demo scene must press 2 when the 2 key is hit")
		}
	})
	t.Run("unhappy: a digit with no verb/noun entry lights the key but writes nothing", func(t *testing.T) {
		d := newDemo()
		d.press('1')
		s := d.state()
		if s.Verb != "" || s.Noun != "" || s.Prog != "" {
			t.Fatalf("a lone digit must not write the display, got %+v", s)
		}
		if s.Pressed != '1' {
			t.Fatal("the key must still depress so you can see the press")
		}
		d.advance(500)
		if d.state().Pressed != 0 {
			t.Fatal("the key must release after the hold")
		}
	})
}
