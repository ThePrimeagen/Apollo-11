package main

// Lab harness tests: one row of three cockpit switches, h/l to move,
// space or enter to flick. Written before the implementation.

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func key(m labModel, r rune) labModel {
	mm, _ := m.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
	return mm.(labModel)
}

func TestLabBoard(t *testing.T) {
	t.Run("happy: three switches, exactly one focused at boot", func(t *testing.T) {
		m := newLab()
		if len(m.switches) != 3 {
			t.Fatalf("want 3 switches, got %d", len(m.switches))
		}
		n := 0
		for _, b := range m.switches {
			if b.Focused {
				n++
			}
		}
		if n != 1 {
			t.Fatalf("want exactly 1 focused switch, got %d", n)
		}
		if !strings.Contains(m.View().Content, "SWITCH") {
			t.Fatal("the board must carry its title")
		}
	})
}

func TestLabNavigation(t *testing.T) {
	t.Run("happy: l/h move across the switches with wrap", func(t *testing.T) {
		m := newLab()
		m = key(m, 'l')
		if m.col != 1 {
			t.Fatalf("l should move to switch 1, got %d", m.col)
		}
		m = key(m, 'l')
		m = key(m, 'l')
		if m.col != 0 {
			t.Fatalf("l past the edge wraps to 0, got %d", m.col)
		}
		m = key(m, 'h')
		if m.col != 2 {
			t.Fatalf("h from 0 wraps to 2, got %d", m.col)
		}
	})
	t.Run("unhappy: unknown keys change nothing", func(t *testing.T) {
		m := newLab()
		m = key(m, 'z')
		if m.col != 0 {
			t.Fatal("unknown keys must not move focus")
		}
	})
}

func TestLabPress(t *testing.T) {
	t.Run("happy: space flicks the focused switch, enter too", func(t *testing.T) {
		m := newLab()
		mm, _ := m.Update(tea.KeyPressMsg{Code: tea.KeySpace, Text: " "})
		m = mm.(labModel)
		if !m.switches[0].On {
			t.Fatal("space must flick the focused switch on")
		}
		mm, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
		m = mm.(labModel)
		if m.switches[0].On {
			t.Fatal("enter must flick it back off")
		}
	})
	t.Run("unhappy: flicking never affects unfocused switches", func(t *testing.T) {
		m := newLab()
		mm, _ := m.Update(tea.KeyPressMsg{Code: tea.KeySpace, Text: " "})
		m = mm.(labModel)
		for i, b := range m.switches {
			if i != 0 && b.On {
				t.Fatalf("switch %d flipped without focus", i)
			}
		}
	})
	t.Run("happy: q quits", func(t *testing.T) {
		m := newLab()
		_, cmd := m.Update(tea.KeyPressMsg{Code: 'q', Text: "q"})
		if cmd == nil {
			t.Fatal("q must quit")
		}
		if _, ok := cmd().(tea.QuitMsg); !ok {
			t.Fatal("q's command must be tea.Quit")
		}
	})
}
