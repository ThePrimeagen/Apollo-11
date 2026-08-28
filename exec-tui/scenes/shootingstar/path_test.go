package shootingstar

// Tests written FIRST: the flight paths. Circle and square are the
// tuner previews — optional closed loops so the persist tail is
// readable. RandomCrossing is the scene: always right-to-left, from
// near the top of the right edge to near the bottom of the left edge.
// A light bend keeps it from being a ruler. At(t) returns position
// and heading in unit space.

import (
	"math"
	"testing"

	"github.com/theprimeagen/apollo-11/exec-tui/components/particle"
)

const (
	boxW = 80.0
	boxH = 40.0
)

func TestCircleAt(t *testing.T) {
	t.Run("happy: a lap stays on the radius and the heading is tangent", func(t *testing.T) {
		cx, cy, r := 40.0, 20.0, 12.0
		p0, h0 := CircleAt(cx, cy, r, 0)
		if math.Abs(p0.X-(cx+r)) > 1e-9 || math.Abs(p0.Y-cy) > 1e-9 {
			t.Fatalf("angle 0 landed %+v, want (%v,%v)", p0, cx+r, cy)
		}
		if math.Abs(h0.Len()-1) > 1e-9 {
			t.Fatalf("heading %+v is not unit", h0)
		}
		// tangent at angle 0 (pointing +Y in screen space? we use
		// counterclockwise: heading is (-sin, cos) so at 0 it's (0, 1)
		if math.Abs(h0.X) > 1e-9 || math.Abs(h0.Y-1) > 1e-9 {
			t.Fatalf("tangent at 0 is %+v, want (0,1)", h0)
		}
		p1, _ := CircleAt(cx, cy, r, math.Pi/2)
		d := math.Hypot(p1.X-cx, p1.Y-cy)
		if math.Abs(d-r) > 1e-9 {
			t.Fatalf("quarter-turn radius %.3f, want %.3f", d, r)
		}
	})
	t.Run("unhappy: a zero radius sits on the center with a default heading", func(t *testing.T) {
		p, h := CircleAt(10, 8, 0, 1.2)
		if p != (particle.Vec2{X: 10, Y: 8}) {
			t.Fatalf("zero radius landed %+v, want the center", p)
		}
		if h == (particle.Vec2{}) {
			t.Fatal("even a parked circle must hand back a heading")
		}
	})
}

func TestSquareAt(t *testing.T) {
	t.Run("happy: t walks the four sides and the heading follows the side", func(t *testing.T) {
		x0, y0, x1, y1 := 10.0, 8.0, 30.0, 24.0
		p, h := SquareAt(x0, y0, x1, y1, 0)
		if math.Abs(p.X-x0) > 1e-9 || math.Abs(p.Y-y0) > 1e-9 {
			t.Fatalf("t=0 landed %+v, want the top-left (%v,%v)", p, x0, y0)
		}
		if math.Abs(h.X-1) > 1e-9 || math.Abs(h.Y) > 1e-9 {
			t.Fatalf("first side heading %+v, want (1,0)", h)
		}
		p, h = SquareAt(x0, y0, x1, y1, 0.25)
		if math.Abs(p.X-x1) > 1e-9 || math.Abs(p.Y-y0) > 1e-9 {
			t.Fatalf("t=0.25 landed %+v, want top-right", p)
		}
		if math.Abs(h.X) > 1e-9 || math.Abs(h.Y-1) > 1e-9 {
			t.Fatalf("second side heading %+v, want (0,1)", h)
		}
		p, _ = SquareAt(x0, y0, x1, y1, 0.5)
		if math.Abs(p.X-x1) > 1e-9 || math.Abs(p.Y-y1) > 1e-9 {
			t.Fatalf("t=0.5 landed %+v, want bottom-right", p)
		}
		p, _ = SquareAt(x0, y0, x1, y1, 1)
		if math.Abs(p.X-x0) > 1e-9 || math.Abs(p.Y-y0) > 1e-9 {
			t.Fatalf("t=1 wraps home, landed %+v", p)
		}
	})
	t.Run("unhappy: a degenerate square sits on the origin with a default heading", func(t *testing.T) {
		p, h := SquareAt(5, 5, 5, 5, 0.3)
		if p != (particle.Vec2{X: 5, Y: 5}) {
			t.Fatalf("degenerate square landed %+v", p)
		}
		if h == (particle.Vec2{}) {
			t.Fatal("a degenerate square must still hand back a heading")
		}
	})
}

func TestRandomCrossing(t *testing.T) {
	t.Run("happy: every seed falls right-to-left, high on the right to low on the left", func(t *testing.T) {
		for seed := int64(1); seed <= 40; seed++ {
			c := RandomCrossing(seed, boxW, boxH)
			if math.Abs(c.Start.X-boxW) > 1e-6 {
				t.Fatalf("seed %d start X=%.2f, want the right edge %.0f", seed, c.Start.X, boxW)
			}
			if math.Abs(c.End.X) > 1e-6 {
				t.Fatalf("seed %d end X=%.2f, want the left edge", seed, c.End.X)
			}
			if c.Start.Y > boxH*0.40 {
				t.Fatalf("seed %d start Y=%.2f is not near the top of the right edge", seed, c.Start.Y)
			}
			if c.End.Y < boxH*0.55 {
				t.Fatalf("seed %d end Y=%.2f is not near the bottom of the left edge", seed, c.End.Y)
			}
			if c.Start.Y >= c.End.Y {
				t.Fatalf("seed %d must fall downward, start Y=%.2f end Y=%.2f", seed, c.Start.Y, c.End.Y)
			}
			p0, h0 := c.At(0)
			if math.Abs(p0.X-c.Start.X) > 1e-9 || math.Abs(p0.Y-c.Start.Y) > 1e-9 {
				t.Fatalf("At(0) %+v, want start %+v", p0, c.Start)
			}
			if h0.X >= 0 {
				t.Fatalf("seed %d heading at start %+v must point left", seed, h0)
			}
			p1, h1 := c.At(1)
			if math.Abs(p1.X-c.End.X) > 1e-9 || math.Abs(p1.Y-c.End.Y) > 1e-9 {
				t.Fatalf("At(1) %+v, want end %+v", p1, c.End)
			}
			if h1 == (particle.Vec2{}) {
				t.Fatal("the heading at t=1 must still point along the flight")
			}
			mid, _ := c.At(0.5)
			if mid.X >= c.Start.X || mid.X <= c.End.X {
				t.Fatalf("seed %d mid X=%.2f left the right-to-left corridor", seed, mid.X)
			}
		}
	})
	t.Run("happy: the same seed draws the same crossing, a different seed draws another", func(t *testing.T) {
		a := RandomCrossing(4, boxW, boxH)
		b := RandomCrossing(4, boxW, boxH)
		c := RandomCrossing(5, boxW, boxH)
		if a != b {
			t.Fatalf("same seed diverged %+v vs %+v", a, b)
		}
		if a == c {
			t.Fatal("a different seed must pick a different crossing")
		}
	})
	t.Run("unhappy: a crossing never runs left-to-right or up, and a zero box does not panic", func(t *testing.T) {
		for seed := int64(1); seed <= 30; seed++ {
			c := RandomCrossing(seed, boxW, boxH)
			if c.Start.X <= c.End.X {
				t.Fatalf("seed %d ran left-to-right: %+v → %+v", seed, c.Start, c.End)
			}
			if c.Start.Y >= c.End.Y {
				t.Fatalf("seed %d ran up or level: %+v → %+v", seed, c.Start, c.End)
			}
			if sameEdge(c.Start, c.End, boxW, boxH) {
				t.Fatalf("seed %d stayed on one edge: %+v → %+v", seed, c.Start, c.End)
			}
		}
		c := RandomCrossing(1, 0, 0)
		p, h := c.At(0.5)
		_ = p
		if h == (particle.Vec2{}) {
			// a collapsed box may park; it must not panic.
		}
		c.At(-1)
		c.At(2)
	})
}

func TestOnceCrossing(t *testing.T) {
	t.Run("happy: one fall from top mid-right to bottom mid-left", func(t *testing.T) {
		c := OnceCrossing(boxW, boxH)
		if c.Start.X <= boxW*0.55 || c.Start.X >= boxW*0.95 {
			t.Fatalf("start X=%.2f is not mid-right of a %.0f-wide box", c.Start.X, boxW)
		}
		if c.Start.Y >= boxH*0.30 {
			t.Fatalf("start Y=%.2f is not near the top", c.Start.Y)
		}
		if c.End.X >= boxW*0.45 || c.End.X <= boxW*0.05 {
			t.Fatalf("end X=%.2f is not mid-left of a %.0f-wide box", c.End.X, boxW)
		}
		if c.End.Y <= boxH*0.55 {
			t.Fatalf("end Y=%.2f is not near the bottom", c.End.Y)
		}
		if c.Start.X <= c.End.X {
			t.Fatalf("must run right-to-left, %+v → %+v", c.Start, c.End)
		}
		if c.Start.Y >= c.End.Y {
			t.Fatalf("must fall downward, %+v → %+v", c.Start, c.End)
		}
		p0, h0 := c.At(0)
		if math.Abs(p0.X-c.Start.X) > 1e-9 || math.Abs(p0.Y-c.Start.Y) > 1e-9 {
			t.Fatalf("At(0) %+v, want start %+v", p0, c.Start)
		}
		if h0.X >= 0 {
			t.Fatalf("heading at start %+v must point left", h0)
		}
		p1, h1 := c.At(1)
		if math.Abs(p1.X-c.End.X) > 1e-9 || math.Abs(p1.Y-c.End.Y) > 1e-9 {
			t.Fatalf("At(1) %+v, want end %+v", p1, c.End)
		}
		if h1 == (particle.Vec2{}) {
			t.Fatal("the heading at t=1 must still point along the flight")
		}
		same := OnceCrossing(boxW, boxH)
		if c != same {
			t.Fatalf("the same box must draw the same once-crossing, %+v vs %+v", c, same)
		}
	})
	t.Run("unhappy: the once-crossing never runs left-to-right or up, and a zero box does not panic", func(t *testing.T) {
		c := OnceCrossing(boxW, boxH)
		if c.Start.X <= c.End.X {
			t.Fatalf("ran left-to-right: %+v → %+v", c.Start, c.End)
		}
		if c.Start.Y >= c.End.Y {
			t.Fatalf("ran up or level: %+v → %+v", c.Start, c.End)
		}
		if math.Abs(c.Start.X-boxW) < 1e-6 || math.Abs(c.End.X) < 1e-6 {
			t.Fatal("the once-crossing is mid-right to mid-left, not edge to edge")
		}
		z := OnceCrossing(0, 0)
		p, h := z.At(0.5)
		_ = p
		if h == (particle.Vec2{}) {
			// a collapsed box may park; it must not panic.
		}
		z.At(-1)
		z.At(2)
	})
}

func onEdge(p particle.Vec2, w, h float64) bool {
	const eps = 1e-6
	return p.X <= eps || p.X >= w-eps || p.Y <= eps || p.Y >= h-eps
}

func sameEdge(a, b particle.Vec2, w, h float64) bool {
	const eps = 1e-6
	left := a.X <= eps && b.X <= eps
	right := a.X >= w-eps && b.X >= w-eps
	top := a.Y <= eps && b.Y <= eps
	bot := a.Y >= h-eps && b.Y >= h-eps
	return left || right || top || bot
}

func distToChord(p, a, b particle.Vec2) float64 {
	dx, dy := b.X-a.X, b.Y-a.Y
	n := math.Hypot(dx, dy)
	if n < 1e-9 {
		return math.Hypot(p.X-a.X, p.Y-a.Y)
	}
	t := ((p.X-a.X)*dx + (p.Y-a.Y)*dy) / (n * n)
	qx, qy := a.X+t*dx, a.Y+t*dy
	return math.Hypot(p.X-qx, p.Y-qy)
}
