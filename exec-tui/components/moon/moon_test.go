package moon

// Tests written FIRST: the moon and the orbit are two separate,
// composable performers. Moon is the pixelated disc alone — a static
// card any scene can reuse. Orbit is the lone gold craft circling the
// moon clockwise — eastward over the top — with no line drawn around
// the moon: the craft alone traces the path, painted over
// transparency so it lays cleanly on top of a Moon (or anything
// else). An arriving orbit streaks the craft in fast off the left
// wing and brakes smoothly into orbital speed — never to a stall —
// before the first lap. Geometry speaks half-cell "pixels" — a
// terminal cell is one pixel wide and two tall — so the circles read
// round on a real terminal. Both components: Start pins the stage,
// Render hands back a stage-sized sprite, Stop empties the stage; the
// orbit's clock carries across restarts so a resize never rewinds the
// lap.

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

// The compile-time pins: the moon and the orbit both play as
// screenplay components.
var (
	_ screenplay.Component = (*Moon)(nil)
	_ screenplay.Component = (*Orbit)(nil)
)

// surfaceGlyphs is every glyph the disc may wear.
var surfaceGlyphs = map[rune]bool{'▓': true, '▒': true, '░': true}

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

func TestArrivalPath(t *testing.T) {
	cx, cy, _, ringR := Geometry(stageW, stageH)
	topRow, topCol := cy-int(float64(ringR)/2+0.5), cx
	t.Run("happy: the ship opens off the left wing, at orbit height", func(t *testing.T) {
		row, col := ArrivalAt(stageW, stageH, 0)
		if col >= 0 {
			t.Fatalf("t=0 col %d — the ship must still be off stage", col)
		}
		if row != topRow {
			t.Fatalf("t=0 row %d, want the top of the ring (%d)", row, topRow)
		}
	})
	t.Run("happy: the streak runs east along orbit height and merges at the top", func(t *testing.T) {
		rowA, colA := ArrivalAt(stageW, stageH, ArriveSeconds*0.3)
		rowB, colB := ArrivalAt(stageW, stageH, ArriveSeconds*0.7)
		if rowA != topRow || rowB != topRow {
			t.Fatalf("streak rows %d/%d, want level flight at %d", rowA, rowB, topRow)
		}
		if colB <= colA {
			t.Fatalf("cols %d → %d — the streak must fly east", colA, colB)
		}
		if colB >= topCol {
			t.Fatalf("col %d before the merge, must still be west of the top (%d)", colB, topCol)
		}
		row, col := ArrivalAt(stageW, stageH, ArriveSeconds)
		if row != topRow || col != topCol {
			t.Fatalf("merge at (%d,%d), want the top of the ring (%d,%d)", row, col, topRow, topCol)
		}
	})
	t.Run("happy: after the merge it rides the ring clockwise, forever", func(t *testing.T) {
		_, colEast := ArrivalAt(stageW, stageH, ArriveSeconds+1)
		if colEast <= topCol {
			t.Fatalf("one beat past the merge the ship sits at col %d — clockwise means east of the top", colEast)
		}
		for tt := ArriveSeconds; tt < ArriveSeconds+OrbitSeconds; tt += 0.5 {
			row, col := ArrivalAt(stageW, stageH, tt)
			if d := pxDist(cx, cy, row, col); math.Abs(d-float64(ringR)) > 1.6 {
				t.Fatalf("t=%.1f the ship left the ring: %.2fpx from center, ring at %dpx", tt, d, ringR)
			}
		}
		r0, c0 := ArrivalAt(stageW, stageH, ArriveSeconds+0.8)
		r1, c1 := ArrivalAt(stageW, stageH, ArriveSeconds+0.8+OrbitSeconds)
		if r0 != r1 || c0 != c1 {
			t.Fatalf("the orbit drifted between laps: (%d,%d) → (%d,%d)", r0, c0, r1, c1)
		}
	})
	t.Run("happy: no stall at the merge — the streak brakes into orbital speed, never to a halt", func(t *testing.T) {
		colAt := func(tt float64) int {
			_, c := ArrivalAt(stageW, stageH, tt)
			return c
		}
		enter := colAt(0.5) - colAt(0)
		brake := colAt(ArriveSeconds) - colAt(ArriveSeconds-0.5)
		orbit := colAt(ArriveSeconds+0.5) - colAt(ArriveSeconds)
		if orbit < 2 {
			t.Fatalf("test premise: the orbit must carry the craft east off the top, moved %d", orbit)
		}
		if brake*2 < orbit {
			t.Fatalf("the last half second covers %d cols against the orbit's %d — the old awkward pause", brake, orbit)
		}
		if enter < brake {
			t.Fatalf("the entry opens at %d cols per half second, the brake at %d — the streak must open fast", enter, brake)
		}
	})
	t.Run("unhappy: time before the curtain clamps to the start", func(t *testing.T) {
		r0, c0 := ArrivalAt(stageW, stageH, 0)
		r1, c1 := ArrivalAt(stageW, stageH, -5)
		if r0 != r1 || c0 != c1 {
			t.Fatalf("negative time moved the ship: (%d,%d) vs (%d,%d)", r0, c0, r1, c1)
		}
	})
	t.Run("unhappy: no geometry means no arrival — the off-stage sentinel", func(t *testing.T) {
		if r, c := ArrivalAt(8, 3, 2); r != -1 || c != -1 {
			t.Fatalf("ArrivalAt on a tiny stage = (%d,%d), want (-1,-1)", r, c)
		}
	})
}

func TestMoonComponent(t *testing.T) {
	t.Run("happy: the moon is the disc alone — filled, centered, nothing else", func(t *testing.T) {
		cx, cy, moonR, _ := Geometry(stageW, stageH)
		m := New()
		m.Start(stageW, stageH)
		sp := m.Render()
		if sp.Width != stageW || sp.Height != stageH {
			t.Fatalf("stage %dx%d, want %dx%d", sp.Width, sp.Height, stageW, stageH)
		}
		for r := 0; r < stageH; r++ {
			for c := 0; c < stageW; c++ {
				cell := sp.At(r, c)
				if pxDist(cx, cy, r, c) <= float64(moonR)-2 && cell.Transparent() {
					t.Fatalf("hole in the moon at (%d,%d)", r, c)
				}
				if cell.Transparent() {
					continue
				}
				if !surfaceGlyphs[cell.Ch] {
					t.Fatalf("the moon alone wears surface glyphs only, found %q at (%d,%d)", cell.Ch, r, c)
				}
			}
		}
		for _, corner := range [][2]int{{0, 0}, {0, stageW - 1}, {stageH - 1, 0}, {stageH - 1, stageW - 1}} {
			if !sp.At(corner[0], corner[1]).Transparent() {
				t.Fatalf("corner (%d,%d) is painted — the sky must show through", corner[0], corner[1])
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
	t.Run("happy: the moon holds still — updates change nothing", func(t *testing.T) {
		m := New()
		m.Start(stageW, stageH)
		a := m.Render().GlyphRows()
		m.Update(3.7)
		b := m.Render().GlyphRows()
		for i := range a {
			if a[i] != b[i] {
				t.Fatalf("row %d moved on a static card:\n%q\n%q", i, a[i], b[i])
			}
		}
	})
	t.Run("happy: stop empties the stage; a restart refits", func(t *testing.T) {
		m := New()
		m.Start(stageW, stageH)
		if opaqueCells(m.Render()) == 0 {
			t.Fatal("test premise: a started moon must show")
		}
		m.Stop()
		if sp := m.Render(); sp.Width != 0 || sp.Height != 0 {
			t.Fatalf("a stopped moon rendered %dx%d", sp.Width, sp.Height)
		}
		m.Start(64, 24)
		if sp := m.Render(); sp.Width != 64 || sp.Height != 24 || opaqueCells(sp) == 0 {
			t.Fatalf("a restaged moon rendered %dx%d with %d cells", sp.Width, sp.Height, opaqueCells(sp))
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

func TestOrbitComponent(t *testing.T) {
	t.Run("happy: the orbit is the craft alone — one gold cell, no line around the moon", func(t *testing.T) {
		o := NewOrbit()
		o.Start(stageW, stageH)
		sp := o.Render()
		if sp.Width != stageW || sp.Height != stageH {
			t.Fatalf("stage %dx%d, want %dx%d", sp.Width, sp.Height, stageW, stageH)
		}
		if n := opaqueCells(sp); n != 1 {
			t.Fatalf("the orbit lit %d cells — the craft alone draws the path in your eye", n)
		}
		r0, c0 := MarkerAt(stageW, stageH, 0)
		if got := sp.At(r0, c0); got.Ch != MarkerGlyph || got.FG != MarkerInk {
			t.Fatalf("the one cell must be the gold craft at (%d,%d), got %q fg %d", r0, c0, got.Ch, got.FG)
		}
	})
	t.Run("happy: the craft opens on the ring and update flies it on", func(t *testing.T) {
		o := NewOrbit()
		o.Start(stageW, stageH)
		sp := o.Render()
		r0, c0 := MarkerAt(stageW, stageH, 0)
		if got := sp.At(r0, c0); got.Ch != MarkerGlyph || got.FG != MarkerInk {
			t.Fatalf("cell (%d,%d) holds %q fg %d, want the gold craft", r0, c0, got.Ch, got.FG)
		}
		o.Update(3)
		sp = o.Render()
		r1, c1 := MarkerAt(stageW, stageH, 3)
		if r1 == r0 && c1 == c0 {
			t.Fatal("test premise: three seconds must move the craft cell")
		}
		if got := sp.At(r1, c1); got.Ch != MarkerGlyph {
			t.Fatalf("after 3s the craft must sit at (%d,%d), cell holds %q", r1, c1, got.Ch)
		}
		if got := sp.At(r0, c0); got.Ch == MarkerGlyph {
			t.Fatal("the craft must leave its old cell behind")
		}
	})
	t.Run("happy: renders between updates are identical — the orbit is deterministic", func(t *testing.T) {
		o := NewOrbit()
		o.Start(stageW, stageH)
		o.Update(1.3)
		a := o.Render().GlyphRows()
		b := o.Render().GlyphRows()
		for i := range a {
			if a[i] != b[i] {
				t.Fatalf("row %d changed without an update:\n%q\n%q", i, a[i], b[i])
			}
		}
	})
	t.Run("happy: stop empties the stage; a restart refits — and never rewinds", func(t *testing.T) {
		o := NewOrbit()
		o.Start(stageW, stageH)
		o.Update(2)
		o.Stop()
		if sp := o.Render(); sp.Width != 0 || sp.Height != 0 {
			t.Fatalf("a stopped orbit rendered %dx%d", sp.Width, sp.Height)
		}
		o.Start(64, 24)
		sp := o.Render()
		if sp.Width != 64 || sp.Height != 24 {
			t.Fatalf("a restaged orbit rendered %dx%d, want 64x24", sp.Width, sp.Height)
		}
		r, c := MarkerAt(64, 24, 2)
		if got := sp.At(r, c); got.Ch != MarkerGlyph {
			t.Fatalf("the clock must carry across restarts: no craft at (%d,%d), cell holds %q", r, c, got.Ch)
		}
	})
	t.Run("unhappy: dt <= 0 holds the orbit still", func(t *testing.T) {
		o := NewOrbit()
		o.Start(stageW, stageH)
		r0, c0 := MarkerAt(stageW, stageH, 0)
		o.Update(0)
		o.Update(-4)
		if got := o.Render().At(r0, c0); got.Ch != MarkerGlyph {
			t.Fatal("zero and negative dt must not move the craft")
		}
	})
	t.Run("unhappy: rendering before the first start is an empty stage", func(t *testing.T) {
		if sp := NewOrbit().Render(); sp.Width != 0 || sp.Height != 0 {
			t.Fatalf("an unstarted orbit rendered %dx%d", sp.Width, sp.Height)
		}
	})
	t.Run("unhappy: a stage too small sits the show out without panicking", func(t *testing.T) {
		o := NewOrbit()
		o.Start(8, 3)
		o.Update(1)
		sp := o.Render()
		if sp.Width != 8 || sp.Height != 3 {
			t.Fatalf("a tiny stage rendered %dx%d, want 8x3", sp.Width, sp.Height)
		}
		if n := opaqueCells(sp); n != 0 {
			t.Fatalf("an orbit that cannot fit lit %d cells", n)
		}
	})
	t.Run("unhappy: a nil orbit skips every cue", func(t *testing.T) {
		var ghost *Orbit
		ghost.Start(4, 2)
		ghost.Update(1)
		ghost.Render()
		ghost.Stop()
	})
}

func TestArrivingOrbit(t *testing.T) {
	t.Run("happy: the curtain opens empty; updates fly the craft in and onto the orbit", func(t *testing.T) {
		o := NewOrbit().Arrive()
		o.Start(stageW, stageH)
		if n := opaqueCells(o.Render()); n != 0 {
			t.Fatalf("an arriving orbit lit %d cells at t=0 — no line, and the craft is still off stage", n)
		}
		o.Update(ArriveSeconds / 2)
		r0, c0 := ArrivalAt(stageW, stageH, ArriveSeconds/2)
		if got := o.Render().At(r0, c0); got.Ch != MarkerGlyph || got.FG != MarkerInk {
			t.Fatalf("mid-streak cell (%d,%d) holds %q fg %d, want the gold craft", r0, c0, got.Ch, got.FG)
		}
		o.Update(ArriveSeconds/2 + 2)
		r1, c1 := ArrivalAt(stageW, stageH, ArriveSeconds+2)
		if got := o.Render().At(r1, c1); got.Ch != MarkerGlyph {
			t.Fatalf("on orbit the craft must sit at (%d,%d), cell holds %q", r1, c1, got.Ch)
		}
	})
	t.Run("unhappy: arrive on a nil orbit skips the cue", func(t *testing.T) {
		var ghost *Orbit
		if ghost.Arrive() != nil {
			t.Fatal("a nil orbit must stay nil")
		}
		ghost.Arrive().Start(4, 2)
		ghost.Render()
	})
}

// Tests written FIRST: Pace retunes one orbit instance — how long the
// arriving streak takes and how long one lap lasts — without touching
// the package consts the stock paths fly. The numbers are the
// operator's, verbatim: zero and negative paces never panic and never
// get rewritten; they simply fly the math they ask for. An unpaced
// orbit is the stock orbit, cell for cell.
func TestOrbitPace(t *testing.T) {
	geometry := func() (cx, cy, ringR int) {
		cx, cy, _, ringR = Geometry(stageW, stageH)
		if ringR < 1 {
			t.Fatal("test premise: the test stage must carry a ring")
		}
		return cx, cy, ringR
	}
	t.Run("happy: a paced lap flies the ring on its own clock", func(t *testing.T) {
		cx, cy, ringR := geometry()
		o := NewOrbit().Pace(ArriveSeconds, 4)
		o.Start(stageW, stageH)
		o.Update(1) // a quarter of a 4s lap
		r, c := ringCell(cx, cy, ringR, startAngle-math.Pi/2)
		if r0, c0 := MarkerAt(stageW, stageH, 1); r0 == r && c0 == c {
			t.Fatal("test premise: the stock lap must sit elsewhere at t=1")
		}
		if got := o.Render().At(r, c); got.Ch != MarkerGlyph {
			t.Fatalf("a quarter lap in, the paced craft must sit at (%d,%d), cell holds %q", r, c, got.Ch)
		}
	})
	t.Run("happy: a paced arrival merges on its own clock and lap", func(t *testing.T) {
		cx, cy, ringR := geometry()
		o := NewOrbit().Arrive().Pace(1, 8)
		o.Start(stageW, stageH)
		o.Update(3) // merged at 1s, then a quarter of an 8s lap
		r, c := ringCell(cx, cy, ringR, 0)
		if r0, c0 := ArrivalAt(stageW, stageH, 3); r0 == r && c0 == c {
			t.Fatal("test premise: the stock arrival must sit elsewhere at t=3")
		}
		if got := o.Render().At(r, c); got.Ch != MarkerGlyph {
			t.Fatalf("the paced arrival must ride the ring's east point (%d,%d), cell holds %q", r, c, got.Ch)
		}
	})
	t.Run("unhappy: an unpaced orbit keeps the stock pace, cell for cell", func(t *testing.T) {
		o := NewOrbit()
		o.Start(stageW, stageH)
		o.Update(3)
		r, c := MarkerAt(stageW, stageH, 3)
		if got := o.Render().At(r, c); got.Ch != MarkerGlyph {
			t.Fatalf("the stock craft must sit at MarkerAt (%d,%d), cell holds %q", r, c, got.Ch)
		}
	})
	t.Run("unhappy: zero and negative paces are the operator's — no panic, stage intact", func(t *testing.T) {
		o := NewOrbit().Pace(0, 0)
		o.Start(stageW, stageH)
		o.Render()
		o.Update(0.5)
		if sp := o.Render(); sp.Width != stageW || sp.Height != stageH {
			t.Fatalf("a zero pace rendered %dx%d, want the stage", sp.Width, sp.Height)
		}
		back := NewOrbit().Arrive().Pace(-1, -4)
		back.Start(stageW, stageH)
		back.Update(1)
		if sp := back.Render(); sp.Width != stageW || sp.Height != stageH {
			t.Fatalf("a negative pace rendered %dx%d, want the stage", sp.Width, sp.Height)
		}
	})
	t.Run("unhappy: pace on a nil orbit skips the cue", func(t *testing.T) {
		var ghost *Orbit
		if ghost.Pace(1, 2) != nil {
			t.Fatal("a nil orbit must stay nil")
		}
		ghost.Pace(1, 2).Start(4, 2)
		ghost.Render()
	})
}

// The compile-time pin: a Horizon plays as a screenplay component.
var _ screenplay.Component = (*Horizon)(nil)

func TestHorizon(t *testing.T) {
	t.Run("happy: the surface is 5 rows at center and 1 row at the edges", func(t *testing.T) {
		if HorizonTop(stageW, stageH, 0) != stageH-HorizonEdgeRows {
			t.Fatalf("left edge top %d, want %d", HorizonTop(stageW, stageH, 0), stageH-HorizonEdgeRows)
		}
		if HorizonTop(stageW, stageH, stageW-1) != stageH-HorizonEdgeRows {
			t.Fatalf("right edge top %d, want %d", HorizonTop(stageW, stageH, stageW-1), stageH-HorizonEdgeRows)
		}
		if HorizonTop(stageW, stageH, stageW/2) != stageH-HorizonCenterRows {
			t.Fatalf("center top %d, want %d", HorizonTop(stageW, stageH, stageW/2), stageH-HorizonCenterRows)
		}
		h := NewHorizon()
		h.Start(stageW, stageH)
		defer h.Stop()
		sp := h.Render()
		edge := 0
		for r := 0; r < stageH; r++ {
			if !sp.At(r, 0).Transparent() {
				edge++
			}
		}
		if edge != HorizonEdgeRows {
			t.Fatalf("left edge holds %d moon rows, want %d", edge, HorizonEdgeRows)
		}
		center := 0
		for r := 0; r < stageH; r++ {
			if !sp.At(r, stageW/2).Transparent() {
				center++
			}
		}
		if center != HorizonCenterRows {
			t.Fatalf("center holds %d moon rows, want %d", center, HorizonCenterRows)
		}
	})
	t.Run("happy: the horizon is a colored floor — background inks, no terrain glyphs in the way of fire", func(t *testing.T) {
		h := NewHorizon()
		h.Start(stageW, stageH)
		defer h.Stop()
		sp := h.Render()
		foundBG := map[int]bool{}
		for r := 0; r < stageH; r++ {
			for c := 0; c < stageW; c++ {
				cell := sp.At(r, c)
				if cell.Transparent() {
					continue
				}
				if r < stageH/2 {
					t.Fatalf("horizon painted row %d — the surface must stay on the bottom half", r)
				}
				if cell.Ch != ' ' && cell.Ch != 0 {
					t.Fatalf("horizon cell (%d,%d) wears glyph %q — the floor must be background so fire can sit on it", r, c, string(cell.Ch))
				}
				if cell.BG < 0 {
					t.Fatalf("horizon cell (%d,%d) has no background", r, c)
				}
				foundBG[cell.BG] = true
			}
		}
		if !foundBG[surfaceInk] {
			t.Fatal("the horizon must show the sunlit moon body as a background color")
		}
		h.Update(3)
		if opaqueCells(h.Render()) != opaqueCells(sp) {
			t.Fatal("the horizon holds still — nothing on the surface moves")
		}
	})
	t.Run("happy: fire painted on the horizon keeps the moon floor underneath", func(t *testing.T) {
		h := NewHorizon()
		h.Start(stageW, stageH)
		defer h.Stop()
		stage := h.Render()
		top := HorizonTop(stageW, stageH, stageW/2)
		floor := stage.At(top, stageW/2)
		if floor.BG < 0 {
			t.Fatal("the ridge must be a background color")
		}
		flame := sprite.New(1, 1)
		flame.Set(0, 0, sprite.Cell{Ch: '⠁', FG: 88, BG: -1})
		sprite.Blit(stage, stageW/2, top, flame)
		got := stage.At(top, stageW/2)
		if got.Ch != '⠁' {
			t.Fatalf("fire must sit on the moon, got %q", string(got.Ch))
		}
		if got.BG != floor.BG {
			t.Fatalf("moon floor %d must stay under the fire, got bg %d", floor.BG, got.BG)
		}
	})
	t.Run("unhappy: a tiny stage and a nil horizon never panic", func(t *testing.T) {
		h := NewHorizon()
		h.Start(2, 1)
		h.Update(1)
		h.Render()
		h.Stop()
		var ghost *Horizon
		ghost.Start(4, 2)
		ghost.Update(1)
		ghost.Render()
		ghost.Stop()
	})
}
