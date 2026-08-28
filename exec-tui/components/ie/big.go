package ie

import (
	"math"

	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
)

// BigMinRadius is the smallest e worth staging, in half-cell pixels;
// below it the big logo sits the scene out.
const BigMinRadius = 8

// The swoosh relative to the e's radius: a tilted ellipse — semi-major
// along the tilt, semi-minor across it — whose front arc crosses the e
// below center and whose back arc shows only beyond the disc.
const (
	swooshTilt  = 18 * math.Pi / 180
	swooshMajor = 1.5
	swooshMinor = 0.5
)

// BigGeometry is the big logo's layout for a w×h stage in half-cell
// pixels — one per column, two per row, so the circles read round: the
// shared center (cx in columns, cy in pixel rows on the 2h-tall grid)
// and the e's outer radius. The margins keep the tilted swoosh inside
// the frame; a stage too small for a BigMinRadius e reports all zeros:
// no geometry, no show.
func BigGeometry(w, h int) (cx, cy, r int) {
	if w < 1 || h < 1 {
		return 0, 0, 0
	}
	// The rotated ring reaches about 1.44 radii sideways; the e itself
	// owns the vertical extent.
	rw := (float64(w)/2 - 3) / 1.44
	rh := float64(h - 3)
	r = int(math.Min(rw, rh))
	if r < BigMinRadius {
		return 0, 0, 0
	}
	return w / 2, h, r
}

// bigBrush is every measure of one staging, precomputed from the
// radius: the stroke of the e, the ring's thickness, and the ellipse.
type bigBrush struct {
	R, t, ts   float64
	a, b       float64
	sinB, cosB float64
}

func newBigBrush(r int) bigBrush {
	R := float64(r)
	return bigBrush{
		R:    R,
		t:    math.Max(3, math.Round(0.34*R)),
		ts:   math.Max(2, math.Round(0.18*R)),
		a:    swooshMajor * R,
		b:    swooshMinor * R,
		sinB: math.Sin(swooshTilt),
		cosB: math.Cos(swooshTilt),
	}
}

// pixel is the ink of one half-cell pixel at (x, y) from the center,
// y up: the swoosh's front arc first — it crosses in front of the e —
// then the e itself, then the back arc where it clears the disc,
// wearing the faded gold as it emerges from behind the letter.
func (bb bigBrush) pixel(x, y float64) int {
	onRing, front := bb.ring(x, y)
	if onRing && front {
		return GoldInk
	}
	d := math.Hypot(x, y)
	if bb.e(x, y, d) {
		return BlueInk
	}
	if onRing && d > bb.R+1 {
		if d < bb.R+bb.ts+2 {
			return GoldFade
		}
		return GoldInk
	}
	return -1
}

// ring reports whether (x, y) sits on the swoosh band and whether that
// point is on the front half of the tilted orbit.
func (bb bigBrush) ring(x, y float64) (on, front bool) {
	xr := x*bb.cosB + y*bb.sinB
	yr := -x*bb.sinB + y*bb.cosB
	f := (xr/bb.a)*(xr/bb.a) + (yr/bb.b)*(yr/bb.b) - 1
	g := 2 * math.Hypot(xr/(bb.a*bb.a), yr/(bb.b*bb.b))
	if g <= 0 {
		return false, false
	}
	return math.Abs(f)/g <= bb.ts/2, yr < 0
}

// e reports whether (x, y) — at distance d from center — belongs to
// the letter: the crossbar spanning the disc, or the annulus with the
// mouth cut open on the lower right.
func (bb bigBrush) e(x, y, d float64) bool {
	if d > bb.R {
		return false
	}
	if math.Abs(y) <= bb.t/2 {
		return true
	}
	if d < bb.R-bb.t {
		return false
	}
	deg := math.Atan2(y, x) * 180 / math.Pi
	return deg <= -62 || deg >= -8
}

// BigArt paints the big logo still for a w×h stage: a stage-sized
// sprite of half-block cells, the blue e under the golden swoosh,
// centered with its margins. A stage with no geometry stays fully
// transparent.
func BigArt(w, h int) sprite.Sprite {
	stage := sprite.New(w, h)
	cx, cy, r := BigGeometry(w, h)
	if r < 1 {
		return stage
	}
	bb := newBigBrush(r)
	for row := 0; row < h; row++ {
		for col := 0; col < w; col++ {
			x := float64(col - cx)
			top := bb.pixel(x, float64(cy-2*row))
			bot := bb.pixel(x, float64(cy-(2*row+1)))
			switch {
			case top < 0 && bot < 0:
				// empty sky — the stage shows through
			case bot < 0:
				stage.Set(row, col, sprite.Cell{Ch: '▀', FG: top, BG: -1})
			case top < 0:
				stage.Set(row, col, sprite.Cell{Ch: '▄', FG: bot, BG: -1})
			case top == bot:
				stage.Set(row, col, sprite.Cell{Ch: '█', FG: top, BG: -1})
			default:
				stage.Set(row, col, sprite.Cell{Ch: '▀', FG: top, BG: bot})
			}
		}
	}
	return stage
}

// Big is the moon-sized logo as a scene component. Start paints and
// caches the still for its stage; Update moves nothing — the logo
// holds still; Render lays the cached still on a stage-sized sprite;
// Stop drops it so a stopped logo holds no allocation.
type Big struct {
	art    sprite.Sprite
	w, h   int
	staged bool
}

// NewBig binds nothing yet: the curtain owns the allocation.
func NewBig() *Big {
	return &Big{}
}

// Start paints the still for a w×h stage.
func (b *Big) Start(w, h int) {
	if b == nil {
		return
	}
	b.w, b.h = w, h
	b.art = BigArt(w, h)
	b.staged = true
}

// Update is a no-op: the logo is a still.
func (b *Big) Update(dt float64) {}

// Render is the cached still on a stage-sized sprite. Before Start
// and after Stop the logo is off, so the stage is empty; a stage with
// no geometry stays transparent.
func (b *Big) Render() sprite.Sprite {
	if b == nil || !b.staged || b.w < 1 || b.h < 1 {
		return sprite.Sprite{}
	}
	stage := sprite.New(b.w, b.h)
	sprite.Blit(stage, 0, 0, b.art)
	return stage
}

// Stop drops the still for the collector; a fresh Start repaints it.
func (b *Big) Stop() {
	if b == nil {
		return
	}
	b.art = sprite.Sprite{}
	b.staged = false
}
