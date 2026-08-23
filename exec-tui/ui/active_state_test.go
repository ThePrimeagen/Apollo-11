package ui

// t41 — "what's on right now" must always be readable: the switches carry
// their own ● ON state and the DSKY PROG digits track the phase. Reset
// clears everything.

import (
	"strings"
	"testing"
)

func TestPhaseAndStateVisibility(t *testing.T) {
	t.Run("happy: the DSKY PROG digits track 6 and a", func(t *testing.T) {
		_, m := newTestModel()
		m = key(m, 'd')
		m = key(m, '6')
		if got := m.dskyState().Prog; got != "64" {
			t.Fatalf("PROG must read 64 in P64, got %q", got)
		}
		m = key(m, 'a')
		if got := m.dskyState().Prog; got != "66" {
			t.Fatalf("PROG must read 66 in P66, got %q", got)
		}
	})
	t.Run("unhappy: reset blanks the phase and every switch state", func(t *testing.T) {
		_, m := newTestModel()
		m = key(m, 'd')
		m = key(m, 'r')
		m = key(m, 'x')
		if got := m.dskyState().Prog; got != "" {
			t.Fatalf("reset must blank PROG, got %q", got)
		}
		if strings.Contains(m.View(), "● ON") {
			t.Fatal("reset must clear every switch's ON state")
		}
	})
}
