package main

// Demo harness tests, written first: dskypad casts the DSKY panel as a
// live keypad — the terminal's keys go straight to the component. The
// pad boots on PROG 63 with blank VERB/NOUN and dark registers. v and
// n open two-digit entries, digits fill them, enter (or e) commits:
// V16 N68 wakes the descent monitor — the registers fill and the
// ALT/VEL lamps light — any other commit puts the monitor back to
// sleep. c or backspace cancels an entry, r extinguishes the lamps,
// q and ctrl+c close the pad. The view is the panel centered in the
// window plus one hint line, always exactly window-height lines. The
// pad is event-driven: no ticks, no clocks.

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func press(m model, msg tea.Msg) model {
	mm, _ := m.Update(msg)
	return mm.(model)
}

func runeKey(r rune) tea.KeyPressMsg { return tea.KeyPressMsg{Code: r, Text: string(r)} }

func enterKey() tea.KeyPressMsg { return tea.KeyPressMsg{Code: tea.KeyEnter} }

func backspaceKey() tea.KeyPressMsg { return tea.KeyPressMsg{Code: tea.KeyBackspace} }

func typeKeys(m model, keys string) model {
	for _, r := range keys {
		m = press(m, runeKey(r))
	}
	return m
}

func TestDskypad(t *testing.T) {
	t.Run("happy: the pad boots on PROG 63 — panel and key hints on view, monitor dark", func(t *testing.T) {
		m := newModel()
		v := m.View().Content
		for _, want := range []string{"VERB", "NOUN", "PROG", "verb", "noun", "entr", "clr", "rset", "quit"} {
			if !strings.Contains(v, want) {
				t.Fatalf("the opening view is missing %q", want)
			}
		}
		if m.panel.State.Prog != "63" {
			t.Fatalf("the pad boots on PROG %q, want 63", m.panel.State.Prog)
		}
		if m.panel.State.Verb != "" || m.panel.State.R1 != "" {
			t.Fatal("the pad boots with a blank verb and dark registers")
		}
	})
	t.Run("happy: typing v16n68 and enter wakes the descent monitor", func(t *testing.T) {
		m := typeKeys(newModel(), "v16n68")
		if m.panel.State.R1 != "" {
			t.Fatal("the monitor must wait for ENTR")
		}
		m = press(m, enterKey())
		st := m.panel.State
		if st.Verb != "16" || st.Noun != "68" {
			t.Fatalf("committed %q/%q, want 16/68", st.Verb, st.Noun)
		}
		if st.R1 != "+01405" || st.R2 != "+00335" || st.R3 != "-02900" {
			t.Fatalf("the registers must fill on V16 N68, got %q %q %q", st.R1, st.R2, st.R3)
		}
		if !st.Lights.Alt || !st.Lights.Vel {
			t.Fatal("the ALT/VEL lamps must light with the monitor")
		}
	})
	t.Run("happy: the e key commits too — the keypad's own ENTR", func(t *testing.T) {
		m := typeKeys(newModel(), "v16n68e")
		if m.panel.State.R1 != "+01405" {
			t.Fatalf("e must commit like enter, registers got %q", m.panel.State.R1)
		}
	})
	t.Run("happy: a non-monitor commit puts the monitor back to sleep", func(t *testing.T) {
		m := typeKeys(newModel(), "v16n68e")
		m = typeKeys(m, "v21n33")
		m = press(m, enterKey())
		st := m.panel.State
		if st.Verb != "21" || st.Noun != "33" {
			t.Fatalf("committed %q/%q, want 21/33", st.Verb, st.Noun)
		}
		if st.R1 != "" || st.R2 != "" || st.R3 != "" {
			t.Fatalf("the registers must go dark off-monitor, got %q %q %q", st.R1, st.R2, st.R3)
		}
		if st.Lights.Alt || st.Lights.Vel {
			t.Fatal("the ALT/VEL lamps must go out with the monitor")
		}
	})
	t.Run("happy: r extinguishes the caution lamps", func(t *testing.T) {
		m := typeKeys(newModel(), "v16n68e")
		if !m.panel.State.Lights.Alt {
			t.Fatal("test premise: the monitor must light the lamps")
		}
		m = press(m, runeKey('r'))
		if m.panel.State.Lights.Alt || m.panel.State.Lights.Vel {
			t.Fatal("RSET must wipe the lamps")
		}
		if m.panel.State.R1 != "+01405" {
			t.Fatal("RSET must leave the registers burning")
		}
	})
	t.Run("happy: c and backspace both cancel an entry back to the old value", func(t *testing.T) {
		m := typeKeys(newModel(), "v16n68e")
		m = typeKeys(m, "n4")
		if m.panel.State.Noun != "4 " {
			t.Fatalf("test premise: mid-entry noun %q, want '4 '", m.panel.State.Noun)
		}
		m = press(m, runeKey('c'))
		if m.panel.State.Noun != "68" {
			t.Fatalf("c must bring back the old noun, got %q", m.panel.State.Noun)
		}
		m = typeKeys(m, "v4")
		m = press(m, backspaceKey())
		if m.panel.State.Verb != "16" {
			t.Fatalf("backspace must bring back the old verb, got %q", m.panel.State.Verb)
		}
	})
	t.Run("happy: the view fills the window exactly across resizes", func(t *testing.T) {
		m := newModel()
		if got := len(strings.Split(m.View().Content, "\n")); got != defaultH {
			t.Fatalf("boot view has %d lines, want %d", got, defaultH)
		}
		m = press(m, tea.WindowSizeMsg{Width: 40, Height: 20})
		if got := len(strings.Split(m.View().Content, "\n")); got != 20 {
			t.Fatalf("view has %d lines for a 20-line window", got)
		}
		m = press(m, tea.WindowSizeMsg{Width: 100, Height: 34})
		if got := len(strings.Split(m.View().Content, "\n")); got != 34 {
			t.Fatalf("view has %d lines for a 34-line window", got)
		}
	})
	t.Run("happy: the pad is event-driven — Init schedules no clock", func(t *testing.T) {
		if newModel().Init() != nil {
			t.Fatal("the pad has nothing to animate; Init must be nil")
		}
	})
	t.Run("unhappy: keys the DSKY does not have leave the pad untouched", func(t *testing.T) {
		m := typeKeys(newModel(), "v16n68e")
		before := m.View().Content
		m = typeKeys(m, "xz+ #")
		if m.View().Content != before {
			t.Fatal("unmapped keys must change nothing")
		}
	})
	t.Run("unhappy: enter with nothing open leaves the pad quiet", func(t *testing.T) {
		m := newModel()
		before := m.View().Content
		m = press(m, enterKey())
		if m.View().Content != before {
			t.Fatal("an idle ENTR must change nothing")
		}
		if m.panel.State.R1 != "" {
			t.Fatal("an idle ENTR must not wake the monitor")
		}
	})
	t.Run("unhappy: q and ctrl+c close the pad", func(t *testing.T) {
		for _, msg := range []tea.Msg{
			runeKey('q'),
			tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl},
		} {
			_, cmd := newModel().Update(msg)
			if cmd == nil {
				t.Fatalf("%v must quit", msg)
			}
			if _, ok := cmd().(tea.QuitMsg); !ok {
				t.Fatalf("%v must issue tea.Quit", msg)
			}
		}
	})
}
