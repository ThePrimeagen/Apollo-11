package ui

// t46 — the real DSKY panel (dsky-lab component) embedded in exec-tui,
// replacing the one-line readout. The engine drives it: verb/noun as keyed,
// PROG from the phase, registers from the flight values, PROG/RESTART lamps
// from the alarms, ALT/VEL from the landing-radar lock.

import (
	"strings"
	"testing"

	"github.com/theprimeagen/apollo-11/exec-tui/sim"
)

func TestDSKYStateMapping(t *testing.T) {
	t.Run("happy: idle boots blank and dark", func(t *testing.T) {
		_, m := newTestModel()
		st := m.dskyState()
		if st.Prog != "" || st.Verb != "" || st.R1 != "" {
			t.Fatalf("idle DSKY must be blank, got %+v", st)
		}
		if st.Lights.Prog || st.Lights.Alt || st.Lights.Vel {
			t.Fatal("no lights at boot")
		}
	})
	t.Run("happy: keyed verb/noun and monitor DELTAH reach the panel", func(t *testing.T) {
		e, m := newTestModel()
		e.StartDescent()
		for _, k := range []byte("V16N68E") {
			e.PressKey(k)
		}
		e.AdvanceAGC(100)
		st := m.dskyState()
		if st.Verb != "16" || st.Noun != "68" {
			t.Fatalf("panel must show V16 N68, got V%q N%q", st.Verb, st.Noun)
		}
		if st.Prog != "63" {
			t.Fatalf("P63 must show in PROG, got %q", st.Prog)
		}
		if !strings.HasPrefix(st.R3, "-") {
			t.Fatalf("monitor R3 must be the negative DELTAH, got %q", st.R3)
		}
		for _, r := range []string{st.R1, st.R2, st.R3} {
			if len(r) != 6 {
				t.Fatalf("registers must be sign+5 digits for the panel, got %q", r)
			}
		}
	})
	t.Run("happy: ALT and VEL burn until the landing radar locks", func(t *testing.T) {
		e, m := newTestModel()
		e.StartDescent()
		e.AdvanceAGC(100)
		st := m.dskyState()
		if !st.Lights.Alt || !st.Lights.Vel {
			t.Fatal("ALT/VEL must be lit before radar lock")
		}
		e.AcquireLandingRadar()
		st = m.dskyState()
		if st.Lights.Alt || st.Lights.Vel {
			t.Fatal("radar lock must extinguish ALT/VEL")
		}
	})
	t.Run("happy: an executive overflow lights PROG and RESTART", func(t *testing.T) {
		e, m := newTestModel()
		e.StartDescent()
		for i := 0; i < 8; i++ {
			e.ScheduleJob("HOG", 25, 1e9, false)
		}
		e.ScheduleJob("STRAW", 25, 10, false) // 1202 + restart
		st := m.dskyState()
		if !st.Lights.Prog {
			t.Fatal("the PROG lamp must light after the alarm")
		}
		if !st.Lights.Restart {
			t.Fatal("the RESTART lamp must light during the restart")
		}
		e.AdvanceAGC(5000)
		if m.dskyState().Lights.Restart {
			t.Fatal("the RESTART lamp clears once the restart is done")
		}
	})
	t.Run("happy: the alarm shows ON the DSKY as V05 N09 with the code in R1", func(t *testing.T) {
		e, m := newTestModel()
		e.StartDescent()
		for i := 0; i < 8; i++ {
			e.ScheduleJob("HOG", 25, 1e9, false)
		}
		e.ScheduleJob("STRAW", 25, 10, false)
		st := m.dskyState()
		if st.Verb != "05" || st.Noun != "09" {
			t.Fatalf("the alarm must display as V05 N09, got V%q N%q", st.Verb, st.Noun)
		}
		if st.R1 != " 01202" {
			t.Fatalf("R1 must carry the unsigned alarm code, got %q", st.R1)
		}
		e.AdvanceAGC(4000)
		st = m.dskyState()
		if st.Verb == "05" {
			t.Fatal("the code display must yield back to the flight display")
		}
		if !st.Lights.Prog {
			t.Fatal("PROG stays lit after the code display expires")
		}
	})
	t.Run("happy: digits being keyed on the panel turn dull orange", func(t *testing.T) {
		e, m := newTestModel()
		e.PressKey('V')
		e.PressKey('1')
		st := m.dskyState()
		if st.Verb != "1" || !st.Typing.Verb {
			t.Fatalf("mid-entry verb must be typed, got V%q typing=%v", st.Verb, st.Typing.Verb)
		}
		if !strings.Contains(m.View().Content, "38;5;172") {
			t.Fatal("typed digits must render dull orange on the embedded panel")
		}
	})
	t.Run("unhappy: COMP ACTY is dark on an idle machine at least sometimes", func(t *testing.T) {
		e, m := newTestModel()
		sawDark := false
		for i := 0; i < 50; i++ {
			e.AdvanceAGC(20)
			if !m.dskyState().CompActy {
				sawDark = true
			}
		}
		if !sawDark {
			t.Fatal("an idle machine must show COMP ACTY dark between housekeeping")
		}
	})
}

func TestDSKYPanelEmbedded(t *testing.T) {
	t.Run("happy: the panel renders with its labels and lights", func(t *testing.T) {
		_, m := newTestModel()
		v := m.View().Content
		for _, want := range []string{"VERB", "NOUN", "PROG", "RESTART", "ALT", "VEL", "ENTR", "CLR"} {
			if !strings.Contains(v, want) {
				t.Fatalf("embedded DSKY missing %q", want)
			}
		}
	})
	t.Run("happy: descent digits appear as seven-segment strokes", func(t *testing.T) {
		e, m := newTestModel()
		e.StartDescent()
		e.AdvanceAGC(100)
		if !strings.Contains(m.View().Content, "|_") {
			t.Fatal("running descent must render seven-segment digits on the panel")
		}
	})
	t.Run("unhappy: idle keeps the panel segment-dark", func(t *testing.T) {
		e, m := newTestModel()
		e.AdvanceAGC(100)
		if strings.Contains(m.View().Content, "|_") {
			t.Fatal("an idle DSKY must show no lit segments")
		}
	})
	t.Run("happy: the alarm code reaches the panel registers, not header text", func(t *testing.T) {
		e, m := newTestModel()
		e.StartDescent()
		for i := 0; i < 8; i++ {
			e.ScheduleJob("HOG", 25, 1e9, false)
		}
		e.ScheduleJob("STRAW", 25, 10, false)
		if !strings.Contains(m.dskyState().R1, "1202") {
			t.Fatal("the alarm code must be readable on the DSKY")
		}
		if strings.Contains(m.View().Content, "FAILREG") {
			t.Fatal("no FAILREG text may render — the DSKY carries the codes")
		}
	})
}

func TestEngineLampAccessors(t *testing.T) {
	t.Run("happy: RestartRecently true right after a bailout, false later", func(t *testing.T) {
		e := sim.New()
		for i := 0; i < 8; i++ {
			e.ScheduleJob("HOG", 25, 1e9, false)
		}
		e.ScheduleJob("STRAW", 25, 10, false)
		if !e.RestartRecently(1500) {
			t.Fatal("a fresh bailout must report a recent restart")
		}
		e.AdvanceAGC(5000)
		if e.RestartRecently(1500) {
			t.Fatal("an old restart must age out")
		}
	})
	t.Run("unhappy: a fresh engine has no recent restart and dark COMP ACTY", func(t *testing.T) {
		e := sim.New()
		if e.RestartRecently(1e9) {
			t.Fatal("no restart has ever happened")
		}
		if e.CompActy() {
			t.Fatal("nothing has run yet")
		}
	})
	t.Run("happy: an overloaded machine pins COMP ACTY on", func(t *testing.T) {
		e := sim.New()
		e.StartDescent()
		e.SetRadarBug(true)
		e.AcquireLandingRadar()
		e.AdvanceAGC(4000)
		on := 0
		for i := 0; i < 20; i++ {
			e.AdvanceAGC(50)
			if e.CompActy() {
				on++
			}
		}
		if on < 18 {
			t.Fatalf("at the knife edge COMP ACTY should be nearly always lit, got %d/20", on)
		}
	})
}
