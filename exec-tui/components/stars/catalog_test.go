package stars

// Tests written FIRST: a Catalog is the cached scatter for one stage —
// every star's home cell, laid out once. A component builds it on
// Start, paints it every frame with just a tick and a strategy, and
// drops it on Stop. The catalog must paint exactly what a Field of the
// same shape paints, so the one-shot and cached paths never drift.

import "testing"

type paintGrid map[[2]int]rune

func catalogPaint(c *Catalog, tick int, s Strategy) paintGrid {
	got := paintGrid{}
	c.Paint(tick, s, func(row, col int, ch rune, fg int) {
		got[[2]int{row, col}] = ch
	})
	return got
}

func fieldPaint(f Field) paintGrid {
	got := paintGrid{}
	f.Paint(func(row, col int, ch rune, fg int) {
		got[[2]int{row, col}] = ch
	})
	return got
}

func gridsEqual(a, b paintGrid) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}

func TestCatalog(t *testing.T) {
	t.Run("happy: a cached catalog paints exactly what a field paints", func(t *testing.T) {
		c := NewCatalog(60, 24, [4]int{})
		for _, s := range []Strategy{DustRush, Drift, Hyperspace, Still} {
			for _, tick := range []int{0, 7, 300} {
				want := fieldPaint(Field{Width: 60, Height: 24, Tick: tick, Strategy: s})
				got := catalogPaint(c, tick, s)
				if !gridsEqual(want, got) {
					t.Fatalf("catalog drifted from field at %s tick %d", s.Name, tick)
				}
			}
		}
	})
	t.Run("happy: one catalog serves every frame — homes hold, stars fly", func(t *testing.T) {
		c := NewCatalog(60, 24, [4]int{})
		first := catalogPaint(c, 0, DustRush)
		again := catalogPaint(c, 0, DustRush)
		if !gridsEqual(first, again) {
			t.Fatal("the same tick must paint the same sky from a cached catalog")
		}
		flown := catalogPaint(c, 9, DustRush)
		if gridsEqual(first, flown) {
			t.Fatal("nine ticks of dust-rush left every star in place")
		}
		fresh := catalogPaint(NewCatalog(60, 24, [4]int{}), 9, DustRush)
		if !gridsEqual(flown, fresh) {
			t.Fatal("a cached catalog must fly exactly like a fresh scatter — the homes never drift")
		}
	})
	t.Run("happy: the catalog honors per-layer density", func(t *testing.T) {
		thin := NewCatalog(60, 24, [4]int{})
		thick := NewCatalog(60, 24, [4]int{0, 0, 0, 120})
		count := func(g paintGrid, glyph rune) int {
			n := 0
			for _, ch := range g {
				if ch == glyph {
					n++
				}
			}
			return n
		}
		a := count(catalogPaint(thin, 0, Still), Glyphs[3])
		b := count(catalogPaint(thick, 0, Still), Glyphs[3])
		if b <= a*3 {
			t.Fatalf("near density 120 scattered ✦%d -> ✦%d, want a much thicker layer", a, b)
		}
	})
	t.Run("happy: a still strategy and a negative tick both freeze the sky", func(t *testing.T) {
		c := NewCatalog(40, 12, [4]int{})
		base := catalogPaint(c, 0, DustRush)
		if !gridsEqual(base, catalogPaint(c, 500, Still)) {
			t.Fatal("a still sky must hold its opening frame forever")
		}
		if !gridsEqual(base, catalogPaint(c, -3, DustRush)) {
			t.Fatal("time never runs backwards: negative ticks freeze at the opening frame")
		}
	})
	t.Run("unhappy: zero and negative stages hold no stars", func(t *testing.T) {
		for _, dim := range [][2]int{{0, 10}, {10, 0}, {-3, 5}} {
			c := NewCatalog(dim[0], dim[1], [4]int{})
			if got := catalogPaint(c, 0, DustRush); len(got) != 0 {
				t.Fatalf("stage %v painted %d stars", dim, len(got))
			}
		}
	})
	t.Run("unhappy: nil puts and nil catalogs are no-ops", func(t *testing.T) {
		NewCatalog(10, 5, [4]int{}).Paint(0, DustRush, nil)
		var ghost *Catalog
		ghost.Paint(0, DustRush, func(int, int, rune, int) { t.Fatal("a nil catalog painted") })
	})
}
