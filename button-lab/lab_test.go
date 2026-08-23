package main

// Lab harness tests: a 4-row × 3-column board of buttons, vim-style focus
// movement, enter/space to press. Written before the implementation.

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func key(m labModel, r rune) labModel {
	mm, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	return mm.(labModel)
}

func TestLabBoard(t *testing.T) {
	t.Run("happy: four style rows with three buttons each", func(t *testing.T) {
		m := newLab()
		if len(m.rows) != 4 {
			t.Fatalf("want 4 style rows, got %d", len(m.rows))
		}
		for i, r := range m.rows {
			if len(r.buttons) != 3 {
				t.Fatalf("row %d: want 3 buttons, got %d", i, len(r.buttons))
			}
		}
		v := m.View()
		for _, want := range []string{"PANEL", "HALF-CELL", "PROTRUDE", "SWITCH"} {
			if !strings.Contains(v, want) {
				t.Fatalf("board must label the %s row", want)
			}
		}
	})
	t.Run("happy: exactly one button is focused at boot", func(t *testing.T) {
		m := newLab()
		n := 0
		for _, r := range m.rows {
			for _, b := range r.buttons {
				if b.Focused {
					n++
				}
			}
		}
		if n != 1 {
			t.Fatalf("want exactly 1 focused button, got %d", n)
		}
	})
}

func TestLabNavigation(t *testing.T) {
	t.Run("happy: l/h move across columns with wrap, j/k across rows", func(t *testing.T) {
		m := newLab()
		m = key(m, 'l')
		if m.col != 1 {
			t.Fatalf("l should move to column 1, got %d", m.col)
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
		m = key(m, 'j')
		if m.row != 1 {
			t.Fatalf("j should move to row 1, got %d", m.row)
		}
		m = key(m, 'k')
		m = key(m, 'k')
		if m.row != 3 {
			t.Fatalf("k from 0 wraps to the last row, got %d", m.row)
		}
	})
	t.Run("unhappy: unknown keys change nothing", func(t *testing.T) {
		m := newLab()
		m = key(m, 'z')
		if m.row != 0 || m.col != 0 {
			t.Fatal("unknown keys must not move focus")
		}
	})
}

func TestLabPress(t *testing.T) {
	t.Run("happy: enter presses the focused button in, space too", func(t *testing.T) {
		m := newLab()
		mm, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		m = mm.(labModel)
		if !m.rows[0].buttons[0].On {
			t.Fatal("enter must toggle the focused button on")
		}
		mm, _ = m.Update(tea.KeyMsg{Type: tea.KeySpace})
		m = mm.(labModel)
		if m.rows[0].buttons[0].On {
			t.Fatal("space must toggle it back off")
		}
	})
	t.Run("unhappy: pressing never affects unfocused buttons", func(t *testing.T) {
		m := newLab()
		mm, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		m = mm.(labModel)
		for ri, r := range m.rows {
			for ci, b := range r.buttons {
				if (ri != 0 || ci != 0) && b.On {
					t.Fatalf("button %d/%d flipped without focus", ri, ci)
				}
			}
		}
	})
	t.Run("happy: q quits", func(t *testing.T) {
		m := newLab()
		_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
		if cmd == nil {
			t.Fatal("q must quit")
		}
		if _, ok := cmd().(tea.QuitMsg); !ok {
			t.Fatal("q's command must be tea.Quit")
		}
	})
}
