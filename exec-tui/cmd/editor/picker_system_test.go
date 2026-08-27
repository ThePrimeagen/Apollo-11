package editor

// Tests written FIRST. The 8-bit picker used to reach only the greys
// (232-255) and the 6×6×6 cube (16-231) — 240 of the 256 xterm colors.
// The 16 system colors (0-15) were unreachable, so a whole family of
// suit colors could never be painted. 's' now opens a system block,
// and between the three blocks every one of the 256 colors is pickable.

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestPickerSystemBlock(t *testing.T) {
	t.Run("happy: s switches the picker to the 16 system colors", func(t *testing.T) {
		m := newEd(t)
		m.TermW, m.TermH = 100, 40
		m = send(m, key('c'))
		if !m.PickerOpen {
			t.Fatal("c must open the 8-bit picker")
		}
		m = send(m, key('s'))
		if !m.PickerSystem {
			t.Fatal("s must switch to the system block")
		}
		if m.PickerCube {
			t.Fatal("the system block and the cube cannot both be active")
		}
		v := strings.ToLower(m.View().Content)
		if !strings.Contains(v, "system") {
			t.Fatal("the picker label must say it is on the system block")
		}
	})
	t.Run("happy: picking a system color lands exactly on 0-15", func(t *testing.T) {
		m := newEd(t)
		m = send(m, key('c'))
		m = send(m, key('s'))
		for m.PickerIdx > 0 {
			m = send(m, key('h'))
		}
		m = send(m, key('l'))
		m = send(m, key('l'))
		m = send(m, keyType(tea.KeyEnter))
		if m.PickerOpen {
			t.Fatal("enter must close the picker")
		}
		if m.Brush.FG != 2 {
			t.Fatalf("Brush.FG = %d, want system color 2", m.Brush.FG)
		}
	})
	t.Run("happy: every one of the 256 xterm colors is now reachable", func(t *testing.T) {
		reachable := map[int]bool{}
		for _, g := range Greys {
			reachable[g] = true
		}
		for red := 0; red < 6; red++ {
			for idx := 0; idx < 36; idx++ {
				reachable[cubeColor(red, idx)] = true
			}
		}
		for _, s := range SystemColors {
			reachable[s] = true
		}
		for c := 0; c < 256; c++ {
			if !reachable[c] {
				t.Fatalf("color %d is unreachable from the picker", c)
			}
		}
		if len(reachable) != 256 {
			t.Fatalf("picker reaches %d colors, want 256", len(reachable))
		}
	})
	t.Run("happy: g and the cube keys still switch blocks", func(t *testing.T) {
		m := newEd(t)
		m = send(m, key('c'))
		m = send(m, key('s'))
		m = send(m, key(']'))
		if m.PickerSystem || !m.PickerCube {
			t.Fatal("] must leave the system block for the cube")
		}
		m = send(m, key('s'))
		m = send(m, key('g'))
		if m.PickerSystem || m.PickerCube {
			t.Fatal("g must return to the greys")
		}
	})
	t.Run("unhappy: the system index clamps at both edges", func(t *testing.T) {
		m := newEd(t)
		m = send(m, key('c'))
		m = send(m, key('s'))
		for i := 0; i < 40; i++ {
			m = send(m, key('l'))
			m = send(m, key('j'))
		}
		if m.PickerIdx > 15 {
			t.Fatalf("system index ran to %d, must clamp at 15", m.PickerIdx)
		}
		for i := 0; i < 40; i++ {
			m = send(m, key('h'))
			m = send(m, key('k'))
		}
		if m.PickerIdx < 0 {
			t.Fatalf("system index ran to %d, must clamp at 0", m.PickerIdx)
		}
	})
	t.Run("unhappy: esc from the system block keeps the brush", func(t *testing.T) {
		m := newEd(t)
		m.Brush = Swatch{FG: 252, BG: -1}
		m = send(m, key('c'))
		m = send(m, key('s'))
		m = send(m, key('l'))
		m = send(m, keyType(tea.KeyEsc))
		if m.PickerOpen {
			t.Fatal("esc must close the picker")
		}
		if m.Brush != (Swatch{FG: 252, BG: -1}) {
			t.Fatalf("esc must keep the brush, got %+v", m.Brush)
		}
	})
}
