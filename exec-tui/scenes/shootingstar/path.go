package shootingstar

import (
	"math"
	"math/rand"

	"github.com/theprimeagen/apollo-11/exec-tui/components/particle"
)

// Crossing is one random meteor: a quadratic bezier from Start to End
// through Ctrl. Ctrl sits off the chord so the flight is never a
// straight line.
type Crossing struct {
	Start, Ctrl, End particle.Vec2
}

// CircleAt is a point on a circle of radius r about (cx, cy) at
// angle radians, plus the counterclockwise tangent.
func CircleAt(cx, cy, r, angle float64) (pos, heading particle.Vec2) {
	if r <= 0 {
		return particle.Vec2{X: cx, Y: cy}, particle.Vec2{X: 1, Y: 0}
	}
	s, c := math.Sincos(angle)
	pos = particle.Vec2{X: cx + c*r, Y: cy + s*r}
	heading = particle.Vec2{X: -s, Y: c}
	return pos, heading
}

// SquareAt walks the rectangle (x0,y0)-(x1,y1). t in [0,1] is four
// equal sides — the corners land on 0, 0.25, 0.5, 0.75 — and wraps.
func SquareAt(x0, y0, x1, y1, t float64) (pos, heading particle.Vec2) {
	if x1 <= x0 || y1 <= y0 {
		return particle.Vec2{X: x0, Y: y0}, particle.Vec2{X: 1, Y: 0}
	}
	s := t - math.Floor(t)
	if s < 0 {
		s += 1
	}
	side := int(s * 4)
	if side > 3 {
		side = 0
	}
	u := s*4 - float64(side)
	switch side {
	case 0:
		return particle.Vec2{X: x0 + u*(x1-x0), Y: y0}, particle.Vec2{X: 1, Y: 0}
	case 1:
		return particle.Vec2{X: x1, Y: y0 + u*(y1-y0)}, particle.Vec2{X: 0, Y: 1}
	case 2:
		return particle.Vec2{X: x1 - u*(x1-x0), Y: y1}, particle.Vec2{X: -1, Y: 0}
	default:
		return particle.Vec2{X: x0, Y: y1 - u*(y1-y0)}, particle.Vec2{X: 0, Y: -1}
	}
}

// RandomCrossing draws one meteor: right edge, near the top, to left
// edge, near the bottom. The same seed always draws the same fall. A
// light bend keeps it from being a ruler; the heading stays leftward.
func RandomCrossing(seed int64, w, h float64) Crossing {
	if w <= 0 || h <= 0 {
		return Crossing{}
	}
	rng := rand.New(rand.NewSource(seed))
	start := particle.Vec2{X: w, Y: (0.06 + rng.Float64()*0.26) * h}
	end := particle.Vec2{X: 0, Y: (0.62 + rng.Float64()*0.30) * h}
	if end.Y <= start.Y {
		end.Y = math.Min(h, start.Y+0.20*h)
	}
	mid := particle.Vec2{X: (start.X + end.X) / 2, Y: (start.Y + end.Y) / 2}
	dx, dy := end.X-start.X, end.Y-start.Y
	n := math.Hypot(dx, dy)
	if n < 1e-6 {
		return Crossing{Start: start, Ctrl: mid, End: end}
	}
	px, py := -dy/n, dx/n
	if rng.Intn(2) == 0 {
		px, py = -px, -py
	}
	mag := (0.08 + rng.Float64()*0.10) * n
	ctrl := particle.Vec2{X: mid.X + px*mag, Y: mid.Y + py*mag}
	ctrl.X = clamp(ctrl.X, end.X+0.15*w, start.X-0.15*w)
	ctrl.Y = clamp(ctrl.Y, 0, h)
	return Crossing{Start: start, Ctrl: ctrl, End: end}
}

// DiagonalCrossing is one meteor from high on the left to low on the
// right — the landing's shooting star. A collapsed box parks.
func DiagonalCrossing(w, h float64) Crossing {
	if w <= 0 || h <= 0 {
		return Crossing{}
	}
	start := particle.Vec2{X: 0.04 * w, Y: 0.08 * h}
	end := particle.Vec2{X: 0.96 * w, Y: 0.88 * h}
	ctrl := particle.Vec2{X: (start.X + end.X) / 2, Y: (start.Y + end.Y) / 2}
	return Crossing{Start: start, Ctrl: ctrl, End: end}
}

// OnceCrossing is the Big E meteor: one fall from top mid-right to
// bottom mid-left. The same box always draws the same path. A light
// bend keeps it from being a ruler; the heading stays leftward.
func OnceCrossing(w, h float64) Crossing {
	if w <= 0 || h <= 0 {
		return Crossing{}
	}
	start := particle.Vec2{X: w * 0.72, Y: h * 0.12}
	end := particle.Vec2{X: w * 0.28, Y: h * 0.82}
	mid := particle.Vec2{X: (start.X + end.X) / 2, Y: (start.Y + end.Y) / 2}
	dx, dy := end.X-start.X, end.Y-start.Y
	n := math.Hypot(dx, dy)
	if n < 1e-6 {
		return Crossing{Start: start, Ctrl: mid, End: end}
	}
	px, py := -dy/n, dx/n
	mag := 0.10 * n
	ctrl := particle.Vec2{X: mid.X + px*mag, Y: mid.Y + py*mag}
	ctrl.X = clamp(ctrl.X, 0, w)
	ctrl.Y = clamp(ctrl.Y, 0, h)
	return Crossing{Start: start, Ctrl: ctrl, End: end}
}

// WithStartY moves the start (and the bend) to frac of stage
// height. frac 0 keeps the path as drawn — stock walkthrough and
// the current fall. The operator's number is not clamped.
func (c Crossing) WithStartY(frac, h float64) Crossing {
	if frac == 0 || h <= 0 {
		return c
	}
	dy := frac*h - c.Start.Y
	c.Start.Y += dy
	c.Ctrl.Y += dy
	return c
}

// At samples the bezier at t in [0,1] and the heading along it.
func (c Crossing) At(t float64) (pos, heading particle.Vec2) {
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	u := 1 - t
	pos = particle.Vec2{
		X: u*u*c.Start.X + 2*u*t*c.Ctrl.X + t*t*c.End.X,
		Y: u*u*c.Start.Y + 2*u*t*c.Ctrl.Y + t*t*c.End.Y,
	}
	d := particle.Vec2{
		X: 2*u*(c.Ctrl.X-c.Start.X) + 2*t*(c.End.X-c.Ctrl.X),
		Y: 2*u*(c.Ctrl.Y-c.Start.Y) + 2*t*(c.End.Y-c.Ctrl.Y),
	}
	heading = d.Normalize()
	if heading == (particle.Vec2{}) {
		heading = particle.Vec2{X: 1, Y: 0}
	}
	return pos, heading
}

func (c Crossing) length() float64 {
	prev, _ := c.At(0)
	var n float64
	for i := 1; i <= 16; i++ {
		p, _ := c.At(float64(i) / 16)
		n += math.Hypot(p.X-prev.X, p.Y-prev.Y)
		prev = p
	}
	return n
}
