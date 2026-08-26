package moon

// Tests written FIRST: the descent-orbit card — a pixelated moon
// centered on stage with a wide dotted ring circling it (the descent
// path) and a lone gold marker riding that ring eastward over the top
// — sideways, no fire behind it. The disc stays a little small so the
// orbit flies wide of the surface. Geometry speaks half-cell "pixels"
// — a terminal cell is one pixel wide and two tall — so both circles
// read round on a real terminal. As a component: Start pins the
// stage, Update runs the orbit clock, Render hands back the
// stage-sized card, Stop empties the stage; the clock carries across
// restarts so a resize never rewinds the orbit.

import (
	"math"
	"testing"

	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
	"github.com/theprimeagen/apollo-11/exec-tui/screenplay"
)

const (
	stageW = 72
	stageH = 27
)

// The compile-time pin: a Moon plays as a screenplay component.
var _ screenplay.Component = (*Moon)(nil)

// pxDist is a cell's distance from center in half-cell pixels: one
// pixel per column, two per row.
func pxDist(cx, cy, row, col int) float64 {
	dx := float64(col - cx)
	dy := float64(2 * (row - cy))
	return math.Hypot(dx, dy)
}

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

func TestGeometry(t *testing.T) {
	t.Run("happy: the ring encircles the disc and both fit the stage", func(t *testing.T) {
		cx, cy, moonR, ringR := Geometry(stageW, stageH)
		if cx != stageW/2 || cy != stageH/2 {
			t.Fatalf("center (%d,%d), want the stage center (%d,%d)", cx, cy, stageW/2, stageH/2)
		}
		if moonR < 8 {
			t.Fatalf("moonR %dpx — the default stage deserves a real moon", moonR)
		}
		if ringR-moonR < 6 {
			t.Fatalf("ring %dpx hugs the disc %dpx — the moon stays small so the orbit flies wide", ringR, moonR)
		}
		if cx-ringR < 1 || cx+ringR > stageW-2 {
			t.Fatalf("ring spills the stage horizontally: cx %d ringR %d width %d", cx, ringR, stageW)
		}
		if cy-ringR/2 < 1 || cy+ringR/2 > stageH-2 {
			t.Fatalf("ring spills the stage vertically: cy %d ringR %d height %d", cy, ringR, stageH)
		}
	})
	t.Run("unhappy: a stage too small for the show reports zero geometry", func(t *testing.T) {
		for _, d := range [][2]int{{8, 3}, {0, 0}, {-5, 12}, {12, -5}, {3, 40}} {
			cx, cy, moonR, ringR := Geometry(d[0], d[1])
			if cx != 0 || cy != 0 || moonR != 0 || ringR != 0 {
				t.Fatalf("Geometry(%d,%d) = (%d,%d,%d,%d), want all zeros", d[0], d[1], cx, cy, moonR, ringR)
			}
		}
	})
}

func TestMarkerPath(t *testing.T) {
	cx, cy, moonR, ringR := Geometry(stageW, stageH)
	t.Run("happy: the marker rides the ring, never the surface", func(t *testing.T) {
		for tt := 0.0; tt < OrbitSeconds; tt += 0.5 {
			row, col := MarkerAt(stageW, stageH, tt)
			d := pxDist(cx, cy, row, col)
			if math.Abs(d-float64(ringR)) > 1.6 {
				t.Fatalf("t=%.1f marker at (%d,%d) is %.2fpx out, ring at %dpx", tt, row, col, d, ringR)
			}
			if d <= float64(moonR)+1 {
				t.Fatalf("t=%.1f the marker grazes the surface", tt)
			}
		}
	})
	t.Run("happy: over the top it flies east — still sideways, the other way around", func(t *testing.T) {
		// The marker opens on the upper-left arc and crosses the top a
		// beat later; across that crossing its column must rise.
		topT := OrbitSeconds / 8
		_, before := MarkerAt(stageW, stageH, topT-0.4)
		rowTop, _ := MarkerAt(stageW, stageH, topT)
		_, after := MarkerAt(stageW, stageH, topT+0.4)
		if after <= before {
			t.Fatalf("cols %d → %d across the top — the marker must fly east", before, after)
		}
		if rowTop > cy-ringR/2+1 {
			t.Fatalf("row %d at the crossing, want the top of the ring (~%d)", rowTop, cy-ringR/2)
		}
	})
	t.Run("happy: one full lap returns home", func(t *testing.T) {
		r0, c0 := MarkerAt(stageW, stageH, 0.7)
		r1, c1 := MarkerAt(stageW, stageH, 0.7+OrbitSeconds)
		if r0 != r1 || c0 != c1 {
			t.Fatalf("the lap drifted: (%d,%d) → (%d,%d)", r0, c0, r1, c1)
		}
	})
	t.Run("unhappy: time before the curtain clamps to the start", func(t *testing.T) {
		r0, c0 := MarkerAt(stageW, stageH, 0)
		r1, c1 := MarkerAt(stageW, stageH, -3)
		if r0 != r1 || c0 != c1 {
			t.Fatalf("negative time moved the marker: (%d,%d) vs (%d,%d)", r0, c0, r1, c1)
		}
	})
	t.Run("unhappy: no geometry means no path — the off-stage sentinel", func(t *testing.T) {
		if r, c := MarkerAt(8, 3, 2); r != -1 || c != -1 {
			t.Fatalf("MarkerAt on a tiny stage = (%d,%d), want (-1,-1)", r, c)
		}
	})
}

func TestMoonOnStage(t *testing.T) {
	t.Run("happy: the disc is a filled, centered circle under a dotted ring", func(t *testing.T) {
		cx, cy, moonR, ringR := Geometry(stageW, stageH)
		m := New()
		m.Start(stageW, stageH)
		sp := m.Render()
		if sp.Width != stageW || sp.Height != stageH {
			t.Fatalf("stage %dx%d, want %dx%d", sp.Width, sp.Height, stageW, stageH)
		}
		for r := 0; r < stageH; r++ {
			for c := 0; c < stageW; c++ {
				if pxDist(cx, cy, r, c) <= float64(moonR)-2 && sp.At(r, c).Transparent() {
					t.Fatalf("hole in the moon at (%d,%d)", r, c)
				}
			}
		}
		dots := 0
		for r := 0; r < stageH; r++ {
			for c := 0; c < stageW; c++ {
				if sp.At(r, c).Ch != RingGlyph {
					continue
				}
				dots++
				d := pxDist(cx, cy, r, c)
				if math.Abs(d-float64(ringR)) > 1.6 {
					t.Fatalf("ring dot (%d,%d) sits %.2fpx out, ring at %dpx", r, c, d, ringR)
				}
				if d < float64(moonR)+2 {
					t.Fatalf("ring dot (%d,%d) touches the surface", r, c)
				}
			}
		}
		if dots < 12 {
			t.Fatalf("only %d ring dots — the descent path must read as a circle", dots)
		}
		for _, corner := range [][2]int{{0, 0}, {0, stageW - 1}, {stageH - 1, 0}, {stageH - 1, stageW - 1}} {
			if !sp.At(corner[0], corner[1]).Transparent() {
				t.Fatalf("corner (%d,%d) is painted — the stars must show through", corner[0], corner[1])
			}
		}
	})
	t.Run("happy: the disc reads round — about twice as wide as tall", func(t *testing.T) {
		cx, cy, moonR, _ := Geometry(stageW, stageH)
		m := New()
		m.Start(stageW, stageH)
		sp := m.Render()
		wSpan, hSpan := 0, 0
		for c := 0; c < stageW; c++ {
			if !sp.At(cy, c).Transparent() && pxDist(cx, cy, cy, c) <= float64(moonR) {
				wSpan++
			}
		}
		for r := 0; r < stageH; r++ {
			if !sp.At(r, cx).Transparent() && pxDist(cx, cy, r, cx) <= float64(moonR) {
				hSpan++
			}
		}
		if wSpan < moonR {
			t.Fatalf("the center row holds %d moon cells — the disc must be filled", wSpan)
		}
		if wSpan < 2*hSpan-3 || wSpan > 2*hSpan+3 {
			t.Fatalf("disc spans %d cols × %d rows — want ~2:1 so it reads round", wSpan, hSpan)
		}
	})
	t.Run("happy: the gold marker opens on the ring and update flies it on", func(t *testing.T) {
		m := New()
		m.Start(stageW, stageH)
		sp := m.Render()
		r0, c0 := MarkerAt(stageW, stageH, 0)
		if got := sp.At(r0, c0); got.Ch != MarkerGlyph || got.FG != MarkerInk {
			t.Fatalf("cell (%d,%d) holds %q fg %d, want the gold marker", r0, c0, got.Ch, got.FG)
		}
		m.Update(3)
		sp = m.Render()
		r1, c1 := MarkerAt(stageW, stageH, 3)
		if r1 == r0 && c1 == c0 {
			t.Fatal("test premise: three seconds must move the marker cell")
		}
		if got := sp.At(r1, c1); got.Ch != MarkerGlyph {
			t.Fatalf("after 3s the marker must sit at (%d,%d), cell holds %q", r1, c1, got.Ch)
		}
		if got := sp.At(r0, c0); got.Ch == MarkerGlyph {
			t.Fatal("the marker must leave its old cell behind")
		}
	})
	t.Run("happy: the marker flies alone — no fire trail, nothing stray", func(t *testing.T) {
		known := map[rune]bool{'▓': true, '▒': true, '░': true, RingGlyph: true, MarkerGlyph: true}
		m := New()
		m.Start(stageW, stageH)
		for _, dt := range []float64{0, 1.7, 4.2} {
			m.Update(dt)
			sp := m.Render()
			markers := 0
			for r := 0; r < stageH; r++ {
				for c := 0; c < stageW; c++ {
					cell := sp.At(r, c)
					if cell.Transparent() {
						continue
					}
					if !known[cell.Ch] {
						t.Fatalf("stray glyph %q at (%d,%d) — the craft flies with no trail", cell.Ch, r, c)
					}
					if cell.Ch == MarkerGlyph {
						markers++
					}
				}
			}
			if markers != 1 {
				t.Fatalf("%d marker cells — the craft is one lone gold diamond", markers)
			}
		}
	})
	t.Run("happy: renders between updates are identical — the card is deterministic", func(t *testing.T) {
		m := New()
		m.Start(stageW, stageH)
		m.Update(1.3)
		a := m.Render().GlyphRows()
		b := m.Render().GlyphRows()
		for i := range a {
			if a[i] != b[i] {
				t.Fatalf("row %d changed without an update:\n%q\n%q", i, a[i], b[i])
			}
		}
	})
	t.Run("unhappy: dt <= 0 holds the orbit still", func(t *testing.T) {
		m := New()
		m.Start(stageW, stageH)
		r0, c0 := MarkerAt(stageW, stageH, 0)
		m.Update(0)
		m.Update(-4)
		if got := m.Render().At(r0, c0); got.Ch != MarkerGlyph {
			t.Fatal("zero and negative dt must not move the marker")
		}
	})
}

func TestMoonLifecycle(t *testing.T) {
	t.Run("happy: stop empties the stage; a restart refits — and never rewinds", func(t *testing.T) {
		m := New()
		m.Start(stageW, stageH)
		if opaqueCells(m.Render()) == 0 {
			t.Fatal("test premise: a started moon must show")
		}
		m.Update(2)
		m.Stop()
		if sp := m.Render(); sp.Width != 0 || sp.Height != 0 {
			t.Fatalf("a stopped moon rendered %dx%d", sp.Width, sp.Height)
		}
		m.Start(64, 24)
		sp := m.Render()
		if sp.Width != 64 || sp.Height != 24 {
			t.Fatalf("a restaged moon rendered %dx%d, want 64x24", sp.Width, sp.Height)
		}
		r, c := MarkerAt(64, 24, 2)
		if got := sp.At(r, c); got.Ch != MarkerGlyph {
			t.Fatalf("the clock must carry across restarts: no marker at (%d,%d), cell holds %q", r, c, got.Ch)
		}
	})
	t.Run("unhappy: rendering before the first start is an empty stage", func(t *testing.T) {
		if sp := New().Render(); sp.Width != 0 || sp.Height != 0 {
			t.Fatalf("an unstarted moon rendered %dx%d", sp.Width, sp.Height)
		}
	})
	t.Run("unhappy: a stage too small sits the show out without panicking", func(t *testing.T) {
		m := New()
		m.Start(8, 3)
		m.Update(1)
		sp := m.Render()
		if sp.Width != 8 || sp.Height != 3 {
			t.Fatalf("a tiny stage rendered %dx%d, want 8x3", sp.Width, sp.Height)
		}
		if n := opaqueCells(sp); n != 0 {
			t.Fatalf("a moon that cannot fit lit %d cells", n)
		}
	})
	t.Run("unhappy: a nil moon skips every cue", func(t *testing.T) {
		var ghost *Moon
		ghost.Start(4, 2)
		ghost.Update(1)
		ghost.Render()
		ghost.Stop()
	})
}
