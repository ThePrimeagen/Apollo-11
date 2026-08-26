package sprite

// Tests written FIRST: Blit composes sprite onto sprite — a component
// lays its parts onto its own stage before the scene composites the
// stages. Transparent source cells spare the destination; edges clip.

import "testing"

func blitStamp() Sprite {
	sp := New(2, 2)
	sp.Set(0, 0, Cell{Ch: 'A', FG: 10, BG: -1})
	sp.Set(1, 1, Cell{Ch: 'B', FG: 20, BG: 30})
	return sp
}

func opaqueCount(sp Sprite) int {
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

func TestBlit(t *testing.T) {
	t.Run("happy: opaque cells land with their colors, transparent cells spare the layer below", func(t *testing.T) {
		dst := New(6, 3)
		dst.Set(1, 2, Cell{Ch: '*', FG: 100, BG: -1}) // under the stamp's transparent (0,1)
		Blit(dst, 1, 1, blitStamp())
		if got := dst.At(1, 1); got.Ch != 'A' || got.FG != 10 {
			t.Fatalf("stamp corner = %+v, want a colored A", got)
		}
		if got := dst.At(2, 2); got.Ch != 'B' || got.FG != 20 || got.BG != 30 {
			t.Fatalf("stamp corner = %+v, want a colored B", got)
		}
		if got := dst.At(1, 2); got.Ch != '*' {
			t.Fatalf("transparent stamp cell erased the layer below: %+v", got)
		}
	})
	t.Run("happy: a glyph with no background keeps the floor color underneath", func(t *testing.T) {
		dst := New(3, 2)
		dst.Set(0, 0, Cell{Ch: ' ', FG: -1, BG: 251})
		src := New(1, 1)
		src.Set(0, 0, Cell{Ch: '⠁', FG: 88, BG: -1})
		Blit(dst, 0, 0, src)
		got := dst.At(0, 0)
		if got.Ch != '⠁' || got.FG != 88 {
			t.Fatalf("fire glyph missing: %+v", got)
		}
		if got.BG != 251 {
			t.Fatalf("moon floor must stay under the fire, bg=%d", got.BG)
		}
	})
	t.Run("unhappy: a source that carries its own background replaces the floor", func(t *testing.T) {
		dst := New(3, 2)
		dst.Set(0, 0, Cell{Ch: ' ', FG: -1, BG: 251})
		src := New(1, 1)
		src.Set(0, 0, Cell{Ch: '█', FG: 226, BG: 220})
		Blit(dst, 0, 0, src)
		got := dst.At(0, 0)
		if got.BG != 220 {
			t.Fatalf("hot fire must paint its own bg, got %d", got.BG)
		}
	})
	t.Run("unhappy: blits clip at every edge instead of wrapping", func(t *testing.T) {
		dst := New(4, 3)
		Blit(dst, -1, -1, blitStamp()) // only the stamp's (1,1) survives at (0,0)
		if got := dst.At(0, 0); got.Ch != 'B' {
			t.Fatalf("neg-offset blit put %q at origin, want B", string(got.Ch))
		}
		if opaqueCount(dst) != 1 {
			t.Fatalf("neg-offset blit landed %d cells, want 1", opaqueCount(dst))
		}
		dst2 := New(4, 3)
		Blit(dst2, 3, 2, blitStamp()) // only the stamp's (0,0) fits
		if got := dst2.At(2, 3); got.Ch != 'A' {
			t.Fatalf("edge blit put %q, want A", string(got.Ch))
		}
		if opaqueCount(dst2) != 1 {
			t.Fatalf("edge blit landed %d cells, want 1", opaqueCount(dst2))
		}
		dst3 := New(4, 3)
		Blit(dst3, 99, 99, blitStamp())
		if opaqueCount(dst3) != 0 {
			t.Fatalf("fully off-stage blit landed %d cells", opaqueCount(dst3))
		}
	})
	t.Run("unhappy: empty sprites on either side are harmless", func(t *testing.T) {
		dst := New(3, 2)
		Blit(dst, 0, 0, Sprite{})
		if opaqueCount(dst) != 0 {
			t.Fatalf("empty source landed %d cells", opaqueCount(dst))
		}
		Blit(Sprite{}, 0, 0, blitStamp())
	})
}
