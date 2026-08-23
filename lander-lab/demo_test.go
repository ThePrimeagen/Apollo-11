package main

// Demo harness tests, written first: the Apollo 11 descent as a scripted
// sequence of events you step through with h/l — PDI, throttle-up, the yaw,
// radar lock, V16N68, the five alarms at their true times and altitudes,
// ATT HOLD, P66, touchdown. Alarms accumulate as persistent markers.

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func key(m demoModel, r rune) demoModel {
	mm, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	return mm.(demoModel)
}

func TestScript(t *testing.T) {
	t.Run("happy: boots at PDI, high and horizontal", func(t *testing.T) {
		m := newDemoModel()
		s := m.state()
		if s.AltFt != 49971 {
			t.Fatalf("PDI altitude must be 49,971 ft, got %v", s.AltFt)
		}
		if !strings.Contains(s.Phase, "P63") {
			t.Fatalf("PDI must be P63, got %q", s.Phase)
		}
		if len(s.Alarms) != 0 {
			t.Fatal("no alarms at ignition")
		}
	})
	t.Run("happy: the script carries all five alarms in flight order", func(t *testing.T) {
		codes := []string{}
		for _, ev := range script {
			if ev.alarm != "" {
				codes = append(codes, ev.alarm)
			}
		}
		want := []string{"1202", "1202", "1201", "1202", "1202"}
		if strings.Join(codes, ",") != strings.Join(want, ",") {
			t.Fatalf("alarm sequence must be %v, got %v", want, codes)
		}
	})
	t.Run("happy: the script ends on the surface", func(t *testing.T) {
		last := script[len(script)-1]
		if last.altFt != 0 {
			t.Fatalf("the last event must be touchdown at 0 ft, got %v", last.altFt)
		}
	})
}

func TestStepping(t *testing.T) {
	t.Run("happy: l steps forward and alarms accumulate as markers", func(t *testing.T) {
		m := newDemoModel()
		for i := 0; i < len(script)-1; i++ {
			m = key(m, 'l')
		}
		s := m.state()
		if len(s.Alarms) != 5 {
			t.Fatalf("after stepping to touchdown all 5 alarm markers persist, got %d", len(s.Alarms))
		}
		if s.AltFt != 0 {
			t.Fatalf("final step must be on the surface, got %v ft", s.AltFt)
		}
	})
	t.Run("happy: h steps back and drops markers not yet reached", func(t *testing.T) {
		m := newDemoModel()
		for i := 0; i < len(script)-1; i++ {
			m = key(m, 'l')
		}
		for i := 0; i < len(script)-1; i++ {
			m = key(m, 'h')
		}
		s := m.state()
		if s.AltFt != 49971 || len(s.Alarms) != 0 {
			t.Fatalf("stepping back to PDI must clear markers, got alt %v with %d alarms", s.AltFt, len(s.Alarms))
		}
	})
	t.Run("unhappy: stepping clamps at both ends", func(t *testing.T) {
		m := newDemoModel()
		m = key(m, 'h')
		if m.idx != 0 {
			t.Fatal("h at PDI must clamp")
		}
		for i := 0; i < len(script)+5; i++ {
			m = key(m, 'l')
		}
		if m.idx != len(script)-1 {
			t.Fatal("l at touchdown must clamp")
		}
	})
	t.Run("happy: q quits", func(t *testing.T) {
		m := newDemoModel()
		_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
		if cmd == nil {
			t.Fatal("q must quit")
		}
		if _, ok := cmd().(tea.QuitMsg); !ok {
			t.Fatal("q's command must be tea.Quit")
		}
	})
}
