package ie

// Tests written FIRST: Big is the moon-sized sibling of the fixed
// card — the same old Internet Explorer logo, drawn from geometry
// instead of a pixel map so it fills whatever stage it is given, the
// way the moon's disc does. BigGeometry speaks half-cell pixels (one
// per column, two per row, so circles read round): the shared center
// and the e's outer radius, with margins so the golden swoosh never
// grazes the frame, and all zeros on a stage too small for the show.
// BigArt paints the whole stage-sized still: the bold blue e —
// annulus, crossbar, open mouth, hollow counter — under the golden
// swoosh, whose front arc crosses the e below center and whose wings
// reach beyond the disc on both sides. Big is the still screenplay
// component wrapping it: Start paints and caches, Update moves
// nothing, Stop empties the stage, and every cue is nil-safe.

import (
	"testing"

	"github.com/theprimeagen/apollo-11/exec-tui/screenplay"
)

// The compile-time pin: the big logo plays as a screenplay component.
var _ screenplay.Component = (*Big)(nil)

const (
	bigW = 80
	bigH = 19
)

func isGold(fg int) bool { return fg == GoldInk || fg == GoldFade }

func TestBigGeometry(t *testing.T) {
	t.Run("happy: the logo centers on the stage and the radius is worth the show", func(t *testing.T) {
		cx, cy, r := BigGeometry(bigW, bigH)
		if cx != bigW/2 || cy != bigH {
			t.Fatalf("center (%d,%d)px, want the stage center (%d,%d)", cx, cy, bigW/2, bigH)
		}
		if r < BigMinRadius {
			t.Fatalf("radius %dpx on the default stage — the big logo deserves its name", r)
		}
	})
	t.Run("happy: a bigger stage paints a bigger e", func(t *testing.T) {
		_, _, small := BigGeometry(60, 16)
		_, _, big := BigGeometry(120, 30)
		if small < BigMinRadius {
			t.Fatalf("test premise: 60x16 must fit a logo, radius %d", small)
		}
		if big <= small {
			t.Fatalf("radius %d on 120x30 vs %d on 60x16 — the logo must grow with the stage", big, small)
		}
	})
	t.Run("unhappy: a stage too small for the show reports zero geometry", func(t *testing.T) {
		for _, d := range [][2]int{{20, 6}, {0, 0}, {-5, 12}, {12, -5}, {8, 40}} {
			cx, cy, r := BigGeometry(d[0], d[1])
			if cx != 0 || cy != 0 || r != 0 {
				t.Fatalf("BigGeometry(%d,%d) = (%d,%d,%d), want all zeros", d[0], d[1], cx, cy, r)
			}
		}
	})
}

func TestBigArt(t *testing.T) {
	t.Run("happy: the still is stage-sized, half-block pixels, the IE inks and nothing else", func(t *testing.T) {
		a := BigArt(bigW, bigH)
		if a.Width != bigW || a.Height != bigH {
			t.Fatalf("still is %dx%d, want the %dx%d stage", a.Width, a.Height, bigW, bigH)
		}
		for r := 0; r < a.Height; r++ {
			for c := 0; c < a.Width; c++ {
				cell := a.At(r, c)
				if !cardGlyphs[cell.Ch] {
					t.Fatalf("cell (%d,%d) wears glyph %q — half-cell pixels are blocks only", r, c, cell.Ch)
				}
				if !cardInks[cell.FG] || !cardInks[cell.BG] {
					t.Fatalf("cell (%d,%d) wears fg %d bg %d — the logo knows only the IE blue and the golds", r, c, cell.FG, cell.BG)
				}
			}
		}
	})
	t.Run("happy: the e reads — blue crossbar at center, hollow counter above it", func(t *testing.T) {
		cx, cy, r := BigGeometry(bigW, bigH)
		a := BigArt(bigW, bigH)
		if got := a.At(cy/2, cx); got.Ch != '█' || got.FG != BlueInk {
			t.Fatalf("the crossbar's heart (%d,%d) holds %q fg %d, want a full blue block", cy/2, cx, got.Ch, got.FG)
		}
		hole := (cy - r/2) / 2
		if got := a.At(hole, cx); !got.Transparent() {
			t.Fatalf("the counter at (%d,%d) is painted %q fg %d bg %d — the hole makes the letter", hole, cx, got.Ch, got.FG, got.BG)
		}
		blueLeft, blueRight := bigW, -1
		for row := 0; row < a.Height; row++ {
			for col := 0; col < a.Width; col++ {
				if a.At(row, col).FG != BlueInk && a.At(row, col).BG != BlueInk {
					continue
				}
				if col < blueLeft {
					blueLeft = col
				}
				if col > blueRight {
					blueRight = col
				}
			}
		}
		if blueLeft < cx-r-1 || blueRight > cx+r+1 {
			t.Fatalf("blue spans cols %d..%d — the e stays inside its disc (%d..%d)", blueLeft, blueRight, cx-r, cx+r)
		}
	})
	t.Run("happy: the swoosh orbits — front arc across the lower e, wings beyond both sides", func(t *testing.T) {
		cx, cy, r := BigGeometry(bigW, bigH)
		a := BigArt(bigW, bigH)
		front, left, right := false, false, false
		for row := 0; row < a.Height; row++ {
			for col := 0; col < a.Width; col++ {
				cell := a.At(row, col)
				if !isGold(cell.FG) && !isGold(cell.BG) {
					continue
				}
				if row > cy/2 && col > cx-r && col < cx+r {
					front = true
				}
				if col < cx-r-1 {
					left = true
				}
				if col > cx+r+1 {
					right = true
				}
			}
		}
		if !front {
			t.Fatal("the swoosh's front arc must cross the e below center")
		}
		if !left || !right {
			t.Fatalf("the swoosh's wings must reach beyond the disc: left %v right %v", left, right)
		}
	})
	t.Run("happy: the show keeps its margins — the stage frame stays clear", func(t *testing.T) {
		a := BigArt(bigW, bigH)
		for col := 0; col < a.Width; col++ {
			if !a.At(0, col).Transparent() || !a.At(a.Height-1, col).Transparent() {
				t.Fatalf("col %d paints the frame — the logo breathes inside its margins", col)
			}
		}
		for row := 0; row < a.Height; row++ {
			if !a.At(row, 0).Transparent() || !a.At(row, a.Width-1).Transparent() {
				t.Fatalf("row %d paints the frame — the logo breathes inside its margins", row)
			}
		}
	})
	t.Run("happy: a bigger stage lights more cells", func(t *testing.T) {
		small := opaqueCells(BigArt(60, 16))
		big := opaqueCells(BigArt(120, 30))
		if small == 0 {
			t.Fatal("test premise: 60x16 must paint a logo")
		}
		if big <= small {
			t.Fatalf("120x30 lights %d cells vs %d on 60x16 — the logo must scale", big, small)
		}
	})
	t.Run("unhappy: a stage with no geometry stays transparent without panicking", func(t *testing.T) {
		a := BigArt(20, 6)
		if a.Width != 20 || a.Height != 6 {
			t.Fatalf("a tiny stage painted %dx%d, want 20x6", a.Width, a.Height)
		}
		if n := opaqueCells(a); n != 0 {
			t.Fatalf("a logo that cannot fit lit %d cells", n)
		}
		if a := BigArt(0, 0); a.Width != 0 || a.Height != 0 {
			t.Fatalf("BigArt(0,0) painted %dx%d", a.Width, a.Height)
		}
		_ = BigArt(-3, -1)
	})
}

func TestBigComponent(t *testing.T) {
	t.Run("happy: the component is the cached still, stage-sized", func(t *testing.T) {
		b := NewBig()
		b.Start(bigW, bigH)
		got := b.Render().GlyphRows()
		want := BigArt(bigW, bigH).GlyphRows()
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("row %d differs from the still:\n got %q\nwant %q", i, got[i], want[i])
			}
		}
	})
	t.Run("happy: the logo holds still — updates change nothing", func(t *testing.T) {
		b := NewBig()
		b.Start(bigW, bigH)
		before := b.Render().GlyphRows()
		b.Update(4.2)
		b.Update(-1)
		after := b.Render().GlyphRows()
		for i := range before {
			if before[i] != after[i] {
				t.Fatalf("row %d moved on a still card:\n%q\n%q", i, before[i], after[i])
			}
		}
	})
	t.Run("happy: stop empties the stage; a restart refits", func(t *testing.T) {
		b := NewBig()
		b.Start(bigW, bigH)
		if opaqueCells(b.Render()) == 0 {
			t.Fatal("test premise: a started big logo must show")
		}
		b.Stop()
		if sp := b.Render(); sp.Width != 0 || sp.Height != 0 {
			t.Fatalf("a stopped logo rendered %dx%d", sp.Width, sp.Height)
		}
		b.Start(60, 16)
		sp := b.Render()
		if sp.Width != 60 || sp.Height != 16 || opaqueCells(sp) == 0 {
			t.Fatalf("a restaged logo rendered %dx%d with %d cells", sp.Width, sp.Height, opaqueCells(sp))
		}
	})
	t.Run("unhappy: rendering before the first start is an empty stage", func(t *testing.T) {
		if sp := NewBig().Render(); sp.Width != 0 || sp.Height != 0 {
			t.Fatalf("an unstarted logo rendered %dx%d", sp.Width, sp.Height)
		}
	})
	t.Run("unhappy: a stage too small sits the show out without panicking", func(t *testing.T) {
		b := NewBig()
		b.Start(20, 6)
		b.Update(1)
		sp := b.Render()
		if sp.Width != 20 || sp.Height != 6 {
			t.Fatalf("a tiny stage rendered %dx%d, want 20x6", sp.Width, sp.Height)
		}
		if n := opaqueCells(sp); n != 0 {
			t.Fatalf("a logo that cannot fit lit %d cells", n)
		}
	})
	t.Run("unhappy: a nil big logo skips every cue", func(t *testing.T) {
		var ghost *Big
		ghost.Start(4, 2)
		ghost.Update(1)
		ghost.Render()
		ghost.Stop()
	})
}
