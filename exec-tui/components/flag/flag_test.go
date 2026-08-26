package flag

// Tests written FIRST: the flag component is the full-screened American
// flag — thirteen stripes, the blue canton, all fifty stars — painted
// across every cell of the stage, fading in from pure black over
// FadeSeconds. The fade owns only the colors: the layout is fixed at
// Start, every cell walks its own ramp from black to its finished ink,
// and dt <= 0 never moves the clock. Before Start and after Stop the
// stage is empty; a resize keeps the fade clock so the flag never
// restarts from black mid-scene.

import (
	"testing"

	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
)

const (
	stageW = 72
	stageH = 26
)

// tick advances the fade the way a 30fps runner would.
func tick(f *Flag, seconds float64) {
	const dt = 1.0 / 30
	for t := 0.0; t < seconds-dt/2; t += dt {
		f.Update(dt)
	}
}

// starCount tallies the star glyphs on a rendered stage.
func starCount(sp sprite.Sprite) int {
	n := 0
	for r := 0; r < sp.Height; r++ {
		for c := 0; c < sp.Width; c++ {
			if sp.At(r, c).Ch == StarGlyph {
				n++
			}
		}
	}
	return n
}

func TestFlagGeometry(t *testing.T) {
	t.Run("happy: the canton spans two fifths of the width and seven stripes of the height", func(t *testing.T) {
		if got := CantonCols(72); got != 29 {
			t.Fatalf("CantonCols(72) = %d, want 29", got)
		}
		if got := CantonRows(26); got != 14 {
			t.Fatalf("CantonRows(26) = %d, want 14 — seven of thirteen stripes", got)
		}
		if got := CantonRows(27); got != 15 {
			t.Fatalf("CantonRows(27) = %d, want 15", got)
		}
	})
	t.Run("unhappy: an empty stage has no canton", func(t *testing.T) {
		if got := CantonCols(0); got != 0 {
			t.Fatalf("CantonCols(0) = %d, want 0", got)
		}
		if got := CantonRows(0); got != 0 {
			t.Fatalf("CantonRows(0) = %d, want 0", got)
		}
	})
}

func TestFlagFinished(t *testing.T) {
	t.Run("happy: the finished flag fills every cell of the stage", func(t *testing.T) {
		f := New(2)
		f.Start(stageW, stageH)
		tick(f, 2.5)
		sp := f.Render()
		if sp.Width != stageW || sp.Height != stageH {
			t.Fatalf("the flag renders %dx%d, want the %dx%d stage", sp.Width, sp.Height, stageW, stageH)
		}
		for r := 0; r < stageH; r++ {
			for c := 0; c < stageW; c++ {
				if sp.At(r, c).Transparent() {
					t.Fatalf("cell (%d,%d) is transparent — the flag must be full-screen", r, c)
				}
			}
		}
	})
	t.Run("happy: thirteen stripes — red on top, red on the bottom, white between", func(t *testing.T) {
		f := New(2)
		f.Start(stageW, stageH)
		tick(f, 2.5)
		sp := f.Render()
		if got := sp.At(0, stageW-1).BG; got != RedInk {
			t.Fatalf("the top stripe wears %d, want red %d", got, RedInk)
		}
		if got := sp.At(stageH-1, stageW-1).BG; got != RedInk {
			t.Fatalf("the bottom stripe wears %d, want red %d", got, RedInk)
		}
		// stageH 26 is two rows a stripe: rows 2-3 are stripe two, white.
		if got := sp.At(2, stageW-1).BG; got != WhiteInk {
			t.Fatalf("the second stripe wears %d, want white %d", got, WhiteInk)
		}
	})
	t.Run("happy: the canton is blue and carries all fifty stars", func(t *testing.T) {
		f := New(2)
		f.Start(stageW, stageH)
		tick(f, 2.5)
		sp := f.Render()
		if got := sp.At(0, 0).BG; got != BlueInk {
			t.Fatalf("the canton wears %d, want blue %d", got, BlueInk)
		}
		if got := starCount(sp); got != 50 {
			t.Fatalf("the canton carries %d stars, want 50", got)
		}
		cw, ch := CantonCols(stageW), CantonRows(stageH)
		for r := 0; r < stageH; r++ {
			for c := 0; c < stageW; c++ {
				cell := sp.At(r, c)
				if cell.Ch != StarGlyph {
					continue
				}
				if r >= ch || c >= cw {
					t.Fatalf("a star sits outside the canton at (%d,%d)", r, c)
				}
				if cell.FG != StarInk {
					t.Fatalf("the star at (%d,%d) wears %d, want %d", r, c, cell.FG, StarInk)
				}
			}
		}
	})
	t.Run("happy: a zero fade is at full color from the first frame", func(t *testing.T) {
		f := New(0)
		f.Start(stageW, stageH)
		sp := f.Render()
		if got := sp.At(0, stageW-1).BG; got != RedInk {
			t.Fatalf("a zero fade opens on %d, want red %d", got, RedInk)
		}
	})
}

func TestFlagFade(t *testing.T) {
	t.Run("happy: the curtain rises on pure black", func(t *testing.T) {
		f := New(6)
		f.Start(stageW, stageH)
		sp := f.Render()
		for r := 0; r < stageH; r++ {
			for c := 0; c < stageW; c++ {
				cell := sp.At(r, c)
				if cell.BG != Black {
					t.Fatalf("cell (%d,%d) opens on bg %d, want black %d", r, c, cell.BG, Black)
				}
				if cell.Ch != ' ' && cell.FG != Black {
					t.Fatalf("glyph at (%d,%d) opens on fg %d, want black %d", r, c, cell.FG, Black)
				}
			}
		}
	})
	t.Run("happy: mid-fade the stripes are dim — no longer black, not yet red", func(t *testing.T) {
		f := New(6)
		f.Start(stageW, stageH)
		tick(f, 3)
		sp := f.Render()
		got := sp.At(0, stageW-1).BG
		if got == Black || got == RedInk {
			t.Fatalf("mid-fade the top stripe wears %d — it must sit between black %d and red %d", got, Black, RedInk)
		}
		blue := sp.At(0, 0).BG
		if blue == Black || blue == BlueInk {
			t.Fatalf("mid-fade the canton wears %d — it must sit between black %d and blue %d", blue, Black, BlueInk)
		}
	})
	t.Run("happy: past FadeSeconds the flag holds its finished colors", func(t *testing.T) {
		f := New(1)
		f.Start(stageW, stageH)
		tick(f, 1.2)
		before := sprite.Render(f.Render())
		tick(f, 2)
		if sprite.Render(f.Render()) != before {
			t.Fatal("a finished fade must hold — the flag never drifts past full color")
		}
		if got := f.Render().At(0, stageW-1).BG; got != RedInk {
			t.Fatalf("past the fade the top stripe wears %d, want red %d", got, RedInk)
		}
	})
	t.Run("happy: a resize keeps the fade clock — the flag never falls back to black", func(t *testing.T) {
		f := New(6)
		f.Start(40, 13)
		tick(f, 3)
		f.Stop()
		f.Start(stageW, stageH)
		if got := f.Render().At(0, stageW-1).BG; got == Black {
			t.Fatal("a mid-fade resize must keep the clock, not restart from black")
		}
	})
	t.Run("unhappy: dt <= 0 never moves the fade", func(t *testing.T) {
		f := New(6)
		f.Start(stageW, stageH)
		f.Update(-1)
		f.Update(0)
		if got := f.Render().At(0, stageW-1).BG; got != Black {
			t.Fatalf("dt <= 0 moved the fade to %d — time never runs backwards", got)
		}
	})
}

func TestFlagLifecycle(t *testing.T) {
	t.Run("unhappy: before Start and after Stop the stage is empty", func(t *testing.T) {
		f := New(6)
		if sp := f.Render(); sp.Width != 0 || sp.Height != 0 {
			t.Fatalf("before Start the flag renders %dx%d, want nothing", sp.Width, sp.Height)
		}
		f.Start(stageW, stageH)
		f.Stop()
		if sp := f.Render(); sp.Width != 0 || sp.Height != 0 {
			t.Fatalf("after Stop the flag renders %dx%d, want nothing", sp.Width, sp.Height)
		}
	})
	t.Run("unhappy: a tiny stage is still fully painted, without panic", func(t *testing.T) {
		f := New(1)
		f.Start(3, 2)
		tick(f, 1.5)
		sp := f.Render()
		if sp.Width != 3 || sp.Height != 2 {
			t.Fatalf("the tiny flag renders %dx%d, want 3x2", sp.Width, sp.Height)
		}
		for r := 0; r < 2; r++ {
			for c := 0; c < 3; c++ {
				if sp.At(r, c).Transparent() {
					t.Fatalf("tiny cell (%d,%d) is transparent — the flag must cover the stage", r, c)
				}
			}
		}
	})
	t.Run("unhappy: a zero-size stage renders empty, without panic", func(t *testing.T) {
		f := New(1)
		f.Start(0, 0)
		f.Update(1)
		if sp := f.Render(); sp.Width != 0 || sp.Height != 0 {
			t.Fatalf("a 0x0 stage renders %dx%d, want nothing", sp.Width, sp.Height)
		}
	})
	t.Run("unhappy: a nil flag never panics", func(t *testing.T) {
		var f *Flag
		f.Start(10, 5)
		f.Update(1)
		_ = f.Render()
		f.Stop()
	})
}
