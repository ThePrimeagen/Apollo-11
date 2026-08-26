package sprite

import "testing"

func TestFlipV(t *testing.T) {
	t.Run("happy: rows reverse and block glyphs remap top to bottom", func(t *testing.T) {
		sp := New(3, 2)
		sp.Set(0, 0, Cell{Ch: '▀', FG: 1, BG: 2})
		sp.Set(0, 1, Cell{Ch: '▟', FG: 3, BG: -1})
		sp.Set(1, 2, Cell{Ch: '▁', FG: 4, BG: -1})

		got := FlipV(sp)
		if got.Width != 3 || got.Height != 2 {
			t.Fatalf("dims %dx%d, want 3x2", got.Width, got.Height)
		}
		if c := got.At(1, 0); c.Ch != '▄' || c.FG != 1 || c.BG != 2 {
			t.Fatalf("top-left ▀ should land bottom-left as ▄, got %+v", c)
		}
		if c := got.At(1, 1); c.Ch != '▜' {
			t.Fatalf("▟ should flip to ▜, got %q", string(c.Ch))
		}
		if c := got.At(0, 2); c.Ch != '▔' || c.FG != 4 {
			t.Fatalf("bottom-right ▁ should land top-right as ▔, got %+v", c)
		}
	})
	t.Run("happy: box-drawing slashes swap so upside-down legs still flare out", func(t *testing.T) {
		sp := New(3, 2)
		sp.Set(1, 0, Cell{Ch: '╱', FG: 252, BG: -1})
		sp.Set(1, 2, Cell{Ch: '╲', FG: 252, BG: -1})

		got := FlipV(sp)
		if c := got.At(0, 0); c.Ch != '╲' {
			t.Fatalf("left ╱ must become ╲ after FlipV, got %q", string(c.Ch))
		}
		if c := got.At(0, 2); c.Ch != '╱' {
			t.Fatalf("right ╲ must become ╱ after FlipV, got %q", string(c.Ch))
		}
	})
	t.Run("unhappy: an unmapped rune stays itself after the flip", func(t *testing.T) {
		sp := New(1, 2)
		sp.Set(0, 0, Cell{Ch: 'Ω', FG: 9, BG: -1})
		got := FlipV(sp)
		if c := got.At(1, 0); c.Ch != 'Ω' || c.FG != 9 {
			t.Fatalf("unknown rune must survive FlipV, got %+v", c)
		}
	})
	t.Run("unhappy: ASCII slash is not a box-drawing leg and does not remap", func(t *testing.T) {
		sp := New(2, 2)
		sp.Set(0, 0, Cell{Ch: '/', FG: 1, BG: -1})
		sp.Set(0, 1, Cell{Ch: '\\', FG: 1, BG: -1})
		got := FlipV(sp)
		if c := got.At(1, 0); c.Ch != '/' {
			t.Fatalf("ASCII / must stay /, got %q", string(c.Ch))
		}
		if c := got.At(1, 1); c.Ch != '\\' {
			t.Fatalf("ASCII \\ must stay \\, got %q", string(c.Ch))
		}
	})
}
