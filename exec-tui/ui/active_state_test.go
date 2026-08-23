package ui

// t41 — "what's on right now" must always be readable: the switches carry
// their own ● ON state, and the header phase badge tracks P63/P64/P66.
// Reset clears everything.

import (
	"strings"
	"testing"
)

func TestPhaseAndStateVisibility(t *testing.T) {
	t.Run("happy: the header phase badge tracks 6 and a", func(t *testing.T) {
		_, m := newTestModel()
		m = key(m, 'd')
		m = key(m, '6')
		if !strings.Contains(m.View(), "P64") {
			t.Fatal("header must show P64")
		}
		m = key(m, 'a')
		if !strings.Contains(m.View(), "P66") {
			t.Fatal("header must show P66")
		}
	})
	t.Run("unhappy: reset clears the phase and every switch state", func(t *testing.T) {
		_, m := newTestModel()
		m = key(m, 'd')
		m = key(m, 'r')
		m = key(m, 'x')
		v := m.View()
		if !strings.Contains(v, "P00") {
			t.Fatal("reset must return the phase badge to P00")
		}
		if strings.Contains(v, "● ON") {
			t.Fatal("reset must clear every switch's ON state")
		}
	})
}
