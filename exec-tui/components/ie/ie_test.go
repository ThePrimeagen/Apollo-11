package ie

// Tests written FIRST: the ie component is the old Internet Explorer
// logo as a fixed 14×7 terminal card — the bold blue lowercase e with
// the golden swoosh orbiting it, the swoosh crossing in front of the
// e on the lower left and flicking off the top right. The card is
// drawn in half-cell pixels (a terminal cell is one pixel wide and
// two pixels tall), so the 14×14-pixel logo reads square on a real
// terminal. Art is the card alone, always 14×7, wearing only the IE
// blue and the two golds on half-block glyphs. Logo is the screenplay
// component: Start pins the stage, Render centers the card on a
// stage-sized sprite, Update moves nothing — the logo is a still —
// and Stop empties the stage. A stage too small for the card sits the
// show out without panicking.

import (
	"testing"

	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
	"github.com/theprimeagen/apollo-11/exec-tui/screenplay"
)

// The compile-time pin: the logo plays as a screenplay component.
var _ screenplay.Component = (*Logo)(nil)

// cardGlyphs is every glyph a half-cell pixel card may wear.
var cardGlyphs = map[rune]bool{' ': true, '▀': true, '▄': true, '█': true}

// cardInks is every ink the card may wear on either channel.
var cardInks = map[int]bool{-1: true, BlueInk: true, GoldInk: true, GoldFade: true}

func opaqueCells(sp sprite.Sprite) int {
	n := 0
	for r := 0; r < sp.Height; r++ {
		for c := 0; c < sp.Width; c++ {
			if !sp.At(r, c).Transparent() {
				n++
			}
		}
	}
	return n
}

func TestArt(t *testing.T) {
	t.Run("happy: the card is exactly 14×7 in half-block pixels, the IE inks and nothing else", func(t *testing.T) {
		a := Art()
		if a.Width != Cols || a.Height != Rows || Cols != 14 || Rows != 7 {
			t.Fatalf("card is %dx%d with Cols %d Rows %d, want the fixed 14x7", a.Width, a.Height, Cols, Rows)
		}
		if err := a.Validate(); err != nil {
			t.Fatalf("the card must be a valid sprite: %v", err)
		}
		for r := 0; r < a.Height; r++ {
			for c := 0; c < a.Width; c++ {
				cell := a.At(r, c)
				if !cardGlyphs[cell.Ch] {
					t.Fatalf("cell (%d,%d) wears glyph %q — half-cell pixels are blocks only", r, c, cell.Ch)
				}
				if !cardInks[cell.FG] || !cardInks[cell.BG] {
					t.Fatalf("cell (%d,%d) wears fg %d bg %d — the card knows only the IE blue and the golds", r, c, cell.FG, cell.BG)
				}
			}
		}
	})
	t.Run("happy: the blue e under the golden swoosh — crossbar, left limb, tail, taper", func(t *testing.T) {
		a := Art()
		if got := a.At(3, 6); got.Ch != '█' || got.FG != BlueInk {
			t.Fatalf("the crossbar's heart (3,6) holds %q fg %d, want a full blue block", got.Ch, got.FG)
		}
		if got := a.At(3, 0); got.Ch != '█' || got.FG != GoldInk {
			t.Fatalf("the swoosh's left limb (3,0) holds %q fg %d, want a full gold block", got.Ch, got.FG)
		}
		if got := a.At(0, 8); got.Ch != '▀' || got.FG != GoldInk || got.BG != BlueInk {
			t.Fatalf("the tail crossing the bowl (0,8) holds %q fg %d bg %d, want gold over blue", got.Ch, got.FG, got.BG)
		}
		if got := a.At(1, 13); got.FG != GoldFade {
			t.Fatalf("the tail's tip (1,13) wears fg %d, want the faded gold taper", got.FG)
		}
	})
	t.Run("happy: the counter is a hole and the corners breathe", func(t *testing.T) {
		a := Art()
		for _, c := range []int{5, 6, 7} {
			if !a.At(2, c).Transparent() {
				t.Fatalf("the e's counter at (2,%d) is painted — the hole makes the letter", c)
			}
		}
		for _, corner := range [][2]int{{0, 0}, {0, Cols - 1}, {Rows - 1, 0}, {Rows - 1, Cols - 1}} {
			if !a.At(corner[0], corner[1]).Transparent() {
				t.Fatalf("corner (%d,%d) is painted — the stage must show through", corner[0], corner[1])
			}
		}
		n := opaqueCells(a)
		if n == 0 || n >= Cols*Rows {
			t.Fatalf("the card lights %d of %d cells — a logo, not a slab and not a ghost", n, Cols*Rows)
		}
	})
	t.Run("unhappy: two cards never share cells — painting one must not bleed into the next", func(t *testing.T) {
		a, b := Art(), Art()
		a.Set(3, 6, sprite.Cell{Ch: '█', FG: GoldInk, BG: -1})
		if got := b.At(3, 6); got.FG != BlueInk {
			t.Fatalf("the second card's crossbar turned fg %d — Art must hand out fresh cells", got.FG)
		}
	})
}

func TestLogoComponent(t *testing.T) {
	t.Run("happy: the card sits centered on a big stage, nothing else painted", func(t *testing.T) {
		l := New()
		l.Start(80, 19)
		sp := l.Render()
		if sp.Width != 80 || sp.Height != 19 {
			t.Fatalf("stage %dx%d, want 80x19", sp.Width, sp.Height)
		}
		top, left := (19-Rows)/2, (80-Cols)/2
		if got := sp.At(top+3, left+6); got.Ch != '█' || got.FG != BlueInk {
			t.Fatalf("the centered crossbar (%d,%d) holds %q fg %d, want the full blue block", top+3, left+6, got.Ch, got.FG)
		}
		if got := sp.At(top, left+8); got.FG != GoldInk || got.BG != BlueInk {
			t.Fatalf("the centered tail (%d,%d) wears fg %d bg %d, want gold over blue", top, left+8, got.FG, got.BG)
		}
		if opaqueCells(sp) != opaqueCells(Art()) {
			t.Fatalf("the stage lights %d cells, the card alone lights %d — nothing else may paint", opaqueCells(sp), opaqueCells(Art()))
		}
	})
	t.Run("happy: an exact-fit stage holds the whole card", func(t *testing.T) {
		l := New()
		l.Start(Cols, Rows)
		got := l.Render().GlyphRows()
		want := Art().GlyphRows()
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("row %d on the exact fit:\n got %q\nwant %q", i, got[i], want[i])
			}
		}
	})
	t.Run("happy: the logo holds still — updates change nothing", func(t *testing.T) {
		l := New()
		l.Start(40, 12)
		a := l.Render().GlyphRows()
		l.Update(3.7)
		l.Update(-2)
		l.Update(0)
		b := l.Render().GlyphRows()
		for i := range a {
			if a[i] != b[i] {
				t.Fatalf("row %d moved on a still card:\n%q\n%q", i, a[i], b[i])
			}
		}
	})
	t.Run("happy: stop empties the stage; a restart refits", func(t *testing.T) {
		l := New()
		l.Start(80, 19)
		if opaqueCells(l.Render()) == 0 {
			t.Fatal("test premise: a started logo must show")
		}
		l.Stop()
		if sp := l.Render(); sp.Width != 0 || sp.Height != 0 {
			t.Fatalf("a stopped logo rendered %dx%d", sp.Width, sp.Height)
		}
		l.Start(40, 10)
		sp := l.Render()
		if sp.Width != 40 || sp.Height != 10 {
			t.Fatalf("a restaged logo rendered %dx%d, want 40x10", sp.Width, sp.Height)
		}
		top, left := (10-Rows)/2, (40-Cols)/2
		if got := sp.At(top+3, left+6); got.FG != BlueInk {
			t.Fatalf("the refit crossbar (%d,%d) wears fg %d, want the IE blue", top+3, left+6, got.FG)
		}
	})
	t.Run("unhappy: rendering before the first start is an empty stage", func(t *testing.T) {
		if sp := New().Render(); sp.Width != 0 || sp.Height != 0 {
			t.Fatalf("an unstarted logo rendered %dx%d", sp.Width, sp.Height)
		}
	})
	t.Run("unhappy: a stage too small sits the show out without panicking", func(t *testing.T) {
		for _, d := range [][2]int{{Cols - 1, Rows}, {Cols, Rows - 1}, {1, 1}} {
			l := New()
			l.Start(d[0], d[1])
			l.Update(1)
			sp := l.Render()
			if sp.Width != d[0] || sp.Height != d[1] {
				t.Fatalf("a %dx%d stage rendered %dx%d", d[0], d[1], sp.Width, sp.Height)
			}
			if n := opaqueCells(sp); n != 0 {
				t.Fatalf("a card that cannot fit %dx%d lit %d cells", d[0], d[1], n)
			}
		}
	})
	t.Run("unhappy: a nil logo skips every cue", func(t *testing.T) {
		var ghost *Logo
		ghost.Start(4, 2)
		ghost.Update(1)
		ghost.Render()
		ghost.Stop()
	})
}
