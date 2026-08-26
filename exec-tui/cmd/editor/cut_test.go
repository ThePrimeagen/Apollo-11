package editor

// Tests written FIRST. x is cut/pick-up: on outline, delete the cell under
// the cursor and make that glyph + color the active brush. On fg/bg, x
// strips that color only and leaves the ASCII. Empty cells must not invent
// a glyph. x must not paste, scroll, or move the cursor.

import (
	"testing"

	tea "charm.land/bubbletea/v2"
)

func cutEd(t *testing.T) Model {
	t.Helper()
	return New(blankTestAtlas(), "")
}

func cutSend(m Model, msg tea.Msg) Model {
	got, _ := m.Update(msg)
	return got.(Model)
}

func cutKey(r rune) tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: r, Text: string(r)}
}

// stamp paints ch in the current brush via one-shot insert, which honors
// PaintCh directly. P resets PaintCh from SymIdx, so tests that need a
// specific glyph cannot rely on P alone.
func stamp(m Model, ch rune) Model {
	m = cutSend(m, cutKey('i'))
	return cutSend(m, cutKey(ch))
}

func TestCutPickupX(t *testing.T) {
	t.Run("happy: x deletes the cell and sets current glyph and color to what was there", func(t *testing.T) {
		m := cutEd(t)
		m.CursorR, m.CursorC = 1, 2
		m.PalIdx = -1
		m.Brush = Swatch{FG: 178, BG: 94}
		m = stamp(m, '▀')
		painted := m.Current().At(1, 2)
		if painted.Transparent() || painted.Ch != '▀' {
			t.Fatalf("need a painted cell to cut, got %+v", painted)
		}
		if painted.FG != 178 || painted.BG != 94 {
			t.Fatalf("need a distinctive color on the cell, got fg %d bg %d", painted.FG, painted.BG)
		}

		// Change the brush so pickup is observable.
		m.PaintCh = '█'
		m.Brush = Swatch{FG: 252, BG: -1}
		m.PalIdx = 1

		m = cutSend(m, cutKey('x'))

		got := m.Current().At(1, 2)
		if !got.Transparent() {
			t.Fatalf("x must delete the cell, got %+v", got)
		}
		if got.Ch != ' ' && got.Ch != 0 {
			t.Fatalf("deleted cell must be blank, got %q", string(got.Ch))
		}
		if m.PaintCh != '▀' {
			t.Fatalf("x must set current character to the deleted glyph, got %q", string(m.PaintCh))
		}
		if m.Brush.FG != 178 || m.Brush.BG != 94 {
			t.Fatalf("x must set current color to the deleted swatch, got %+v", m.Brush)
		}
		if m.color() != (Swatch{FG: 178, BG: 94}) {
			t.Fatalf("active paint color must be the cut swatch, got %+v", m.color())
		}
		if m.CursorR != 1 || m.CursorC != 2 {
			t.Fatalf("x must leave the cursor on the cut cell, got (%d,%d)", m.CursorR, m.CursorC)
		}
		if len(m.RecentGlyphs) > 0 && m.RecentGlyphs[0] != '▀' {
			t.Fatalf("picking up a glyph should count as using it, recent[0]=%q", string(m.RecentGlyphs[0]))
		}
		if len(m.RecentColors) > 0 && m.RecentColors[0] != (Swatch{FG: 178, BG: 94}) {
			t.Fatalf("picking up a color should count as using it, recent[0]=%+v", m.RecentColors[0])
		}
	})

	t.Run("happy: after x, P pastes the picked-up glyph and color", func(t *testing.T) {
		m := cutEd(t)
		m.CursorR, m.CursorC = 0, 0
		m.PalIdx = -1
		m.Brush = Swatch{FG: 208, BG: 52}
		m = stamp(m, '▄')
		m.PaintCh = '█'
		m.Brush = Swatch{FG: 252, BG: -1}
		m = cutSend(m, cutKey('x'))
		m.CursorC = 1
		m = cutSend(m, cutKey('P'))
		c := m.Current().At(0, 1)
		if c.Ch != '▄' {
			t.Fatalf("P after x must paste the cut glyph, got %q", string(c.Ch))
		}
		if c.FG != 208 || c.BG != 52 {
			t.Fatalf("P after x must paste the cut color, got fg %d bg %d", c.FG, c.BG)
		}
		if !m.Current().At(0, 0).Transparent() {
			t.Fatal("the cut cell must stay empty; x is not a move/paste")
		}
	})

	t.Run("unhappy: x on an empty cell does not crash or invent a glyph", func(t *testing.T) {
		m := cutEd(t)
		m.CursorR, m.CursorC = 0, 0
		if !m.Current().At(0, 0).Transparent() {
			t.Fatal("test needs an empty cell")
		}
		beforeCh := m.PaintCh
		beforeBrush := m.Brush
		beforePal := m.PalIdx
		m = cutSend(m, cutKey('x'))
		if !m.Current().At(0, 0).Transparent() {
			t.Fatal("x on empty must leave the cell empty")
		}
		if m.PaintCh != beforeCh {
			t.Fatalf("x on empty must keep the current brush glyph, got %q want %q", string(m.PaintCh), string(beforeCh))
		}
		if m.PaintCh == 0 || m.PaintCh == ' ' {
			t.Fatal("x on empty must not invent a space/zero glyph as the brush")
		}
		if m.Brush != beforeBrush || m.PalIdx != beforePal {
			t.Fatalf("x on empty must keep the current color, brush %+v pal %d", m.Brush, m.PalIdx)
		}
	})

	t.Run("unhappy: x at the canvas edge deletes that cell and does not move", func(t *testing.T) {
		m := cutEd(t)
		sp := m.Current()
		m.CursorR, m.CursorC = sp.Height-1, sp.Width-1
		m.PalIdx = -1
		m.Brush = Swatch{FG: 245, BG: -1}
		m = stamp(m, '▓')
		m.PaintCh = '█'
		m.Brush = Swatch{FG: 252, BG: -1}
		m = cutSend(m, cutKey('x'))
		if !m.Current().At(sp.Height-1, sp.Width-1).Transparent() {
			t.Fatal("x at the far corner must delete that cell")
		}
		if m.PaintCh != '▓' {
			t.Fatalf("x at the edge must still pick up the glyph, got %q", string(m.PaintCh))
		}
		if m.CursorR != sp.Height-1 || m.CursorC != sp.Width-1 {
			t.Fatalf("x must not wrap or scroll the cursor, got (%d,%d)", m.CursorR, m.CursorC)
		}
	})

	t.Run("unhappy: x does not paste onto other cells or change the window", func(t *testing.T) {
		m := cutEd(t)
		m.Win = WinCanvas
		m.CursorR, m.CursorC = 2, 3
		m.PalIdx = -1
		m.Brush = Swatch{FG: 255, BG: -1}
		m = stamp(m, '▌')
		neighbor := m.Current().At(2, 4)
		other := m.Current().At(0, 0)
		m = cutSend(m, cutKey('x'))
		if m.Current().At(2, 4) != neighbor {
			t.Fatal("x must not paste or stamp the neighbor")
		}
		if m.Current().At(0, 0) != other {
			t.Fatal("x must not mutate cells away from the cursor")
		}
		if m.Win != WinCanvas {
			t.Fatalf("x must not change window focus, got %v", m.Win)
		}
		if m.CursorR != 2 || m.CursorC != 3 {
			t.Fatalf("x must not scroll or walk the cursor, got (%d,%d)", m.CursorR, m.CursorC)
		}
	})

	t.Run("unhappy: x off the canvas does not cut or change the brush", func(t *testing.T) {
		m := cutEd(t)
		m.CursorR, m.CursorC = 0, 0
		m.PalIdx = -1
		m.Brush = Swatch{FG: 178, BG: 94}
		m = stamp(m, '▀')
		m.Win = WinPalette
		beforeCell := m.Current().At(0, 0)
		beforeCh := m.PaintCh
		beforeBrush := m.Brush
		m = cutSend(m, cutKey('x'))
		if m.Current().At(0, 0) != beforeCell {
			t.Fatal("x in the palette window must not delete canvas cells")
		}
		if m.PaintCh != beforeCh || m.Brush != beforeBrush {
			t.Fatal("x off the canvas must not pick up a brush")
		}
	})

	t.Run("unhappy: d still deletes without stealing the brush", func(t *testing.T) {
		m := cutEd(t)
		m.CursorR, m.CursorC = 0, 0
		m.PalIdx = -1
		m.Brush = Swatch{FG: 24, BG: 232}
		m = stamp(m, '░')
		m.PaintCh = '█'
		m.Brush = Swatch{FG: 252, BG: -1}
		m = cutSend(m, cutKey('d'))
		if !m.Current().At(0, 0).Transparent() {
			t.Fatal("d must still clear the cell")
		}
		if m.PaintCh != '█' {
			t.Fatalf("d must not pick up the deleted glyph, got %q", string(m.PaintCh))
		}
		if m.Brush != (Swatch{FG: 252, BG: -1}) {
			t.Fatalf("d must not pick up the deleted color, got %+v", m.Brush)
		}
	})
}
