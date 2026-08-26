// Package moon holds two separate, composable performers. Moon is the
// pixelated disc alone — a static card any scene can reuse. Orbit is
// the wide dotted ring circling it — the descent path — and the lone
// gold craft riding that ring eastward over the top; cast it over a
// Moon and the card answers the arrival scene's question: this is
// where the craft was, and why it flies sideways.
//
// All circle math runs in half-cell "pixels": a terminal cell is about
// twice as tall as it is wide, so one column counts one pixel and one
// row counts two. A radius of R pixels spans R columns but only R/2
// rows, and both the disc and the ring read round on a real terminal.
// Both performers share Geometry, so any orbit fits any moon on the
// same stage.
package moon

import (
	"math"

	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
)

const (
	// OrbitSeconds is one full lap of the marker around the ring.
	OrbitSeconds = 12.0
	// ArriveSeconds is the fast streak from off the left wing to the
	// top of the ring, where the orbit begins — long enough that the
	// entry reads on screen, short enough to stay a streak.
	ArriveSeconds = 2.5
	// RingGlyph is one dot of the descent path.
	RingGlyph = '◦'
	// MarkerGlyph is the craft riding the path.
	MarkerGlyph = '◆'
	// MarkerInk is the marker's xterm-256 color: the mission gold,
	// the same ink the title cards wear.
	MarkerInk = 178
	// RingInk is the dotted path's gray — bright enough to survive a
	// small font and a compressed tape.
	RingInk = 247
)

// The surface's xterm-256 grays: the sunlit body, the darkened limb,
// the maria plains, the crater pits, and the speckle grain.
const (
	surfaceInk = 251
	limbInk    = 247
	mariaInk   = 243
	craterInk  = 240
	speckleInk = 249
)

const (
	// startAngle opens the marker on the upper-left arc, a beat
	// before the top, so the eastward crossing plays early.
	startAngle = 3 * math.Pi / 4
	// ringGap is how many pixels of empty space sit between the
	// surface and the descent path: the moon stays a little small so
	// the orbit flies wide.
	ringGap = 8
	// minMoonR is the smallest moon worth staging; below it the
	// component sits the scene out.
	minMoonR = 4
)

// patch is one round terrain feature, in fractions of the moon radius
// with y growing upward.
type patch struct{ x, y, r float64 }

// maria are the dark plains, laid out to evoke the near side —
// Imbrium high on the left, Serenitatis and Tranquillitatis to the
// right of center, Nubium low, Crisium at the eastern edge.
var maria = []patch{
	{-0.38, 0.44, 0.26},
	{0.22, 0.30, 0.18},
	{0.46, 0.00, 0.20},
	{-0.24, -0.36, 0.18},
	{0.72, 0.26, 0.10},
}

// pits are the two landmark craters, Tycho south and Copernicus west.
var pits = []patch{
	{-0.02, -0.72, 0.09},
	{-0.52, 0.02, 0.07},
}

// Geometry is the card's layout for a w×h stage: the shared center
// cell and the disc and ring radii in pixels. The ring keeps two
// cells of margin from every stage edge — the show breathes instead
// of grazing the frame — and a stage too small for a minMoonR moon
// reports all zeros: no geometry, no show.
func Geometry(w, h int) (cx, cy, moonR, ringR int) {
	if w < 1 || h < 1 {
		return 0, 0, 0, 0
	}
	cx, cy = w/2, h/2
	ringR = min(min(cx, w-1-cx)-2, 2*(min(cy, h-1-cy)-2))
	moonR = ringR - ringGap
	if moonR < minMoonR {
		return 0, 0, 0, 0
	}
	return cx, cy, moonR, ringR
}

// MarkerAt is the marker's cell at t seconds into the scene: riding
// the ring clockwise — eastward over the top — one lap every
// OrbitSeconds. Time before the curtain clamps to the start. On a
// stage with no geometry there is no path: (-1, -1), the off-stage
// sentinel every Set ignores.
func MarkerAt(w, h int, t float64) (row, col int) {
	cx, cy, _, ringR := Geometry(w, h)
	if ringR < 1 {
		return -1, -1
	}
	if t < 0 {
		t = 0
	}
	return ringCell(cx, cy, ringR, angleAt(t))
}

// ArrivalAt is the arriving ship's cell at t seconds into the scene:
// a fast eased streak from off the left wing, flying east at orbit
// height, that merges onto the top of the ring at ArriveSeconds and
// rides the clockwise orbit from there, lap after lap. Time before
// the curtain clamps to the start; a stage with no geometry has no
// arrival: (-1, -1).
func ArrivalAt(w, h int, t float64) (row, col int) {
	cx, cy, _, ringR := Geometry(w, h)
	if ringR < 1 {
		return -1, -1
	}
	if t < 0 {
		t = 0
	}
	if t >= ArriveSeconds {
		theta := math.Pi/2 - 2*math.Pi*(t-ArriveSeconds)/OrbitSeconds
		return ringCell(cx, cy, ringR, theta)
	}
	row, _ = ringCell(cx, cy, ringR, math.Pi/2)
	// The same ease-out cubic the premiere's craft flies: fast off
	// the wing, gentle into the merge.
	p := t / ArriveSeconds
	eased := 1 - math.Pow(1-p, 3)
	const start = -3.0
	return row, int(math.Round(start + eased*(float64(cx)-start)))
}

// angleAt winds the clock backwards through the angles: a clockwise
// lap on screen.
func angleAt(t float64) float64 {
	return startAngle - 2*math.Pi*t/OrbitSeconds
}

// ringCell is the cell nearest the ring point at theta, with rows
// halved so the circle survives the cell aspect.
func ringCell(cx, cy, ringR int, theta float64) (row, col int) {
	col = cx + int(math.Round(float64(ringR)*math.Cos(theta)))
	row = cy - int(math.Round(float64(ringR)*math.Sin(theta)/2))
	return row, col
}

// Moon is the card as a scene component. Start paints and caches the
// still life — the disc and the dotted ring — for its stage; Update
// runs the orbit clock; Render lays the marker over the cached base;
// Stop drops the base so a stopped moon holds no allocation. The
// clock carries across restarts, so a resize never rewinds the orbit.
type Moon struct {
	base   sprite.Sprite
	w, h   int
	staged bool
}

// New binds nothing yet: the curtain owns the allocation.
func New() *Moon {
	return &Moon{}
}

// Start paints the surface for a w×h stage.
func (m *Moon) Start(w, h int) {
	if m == nil {
		return
	}
	m.w, m.h = w, h
	m.base = paintDisc(w, h)
	m.staged = true
}

// Update is a no-op: the moon holds still — motion belongs to the
// performers cast around it.
func (m *Moon) Update(dt float64) {}

// Render is the cached disc on a stage-sized sprite. Before Start and
// after Stop the card is off, so the stage is empty; a stage with no
// geometry stays transparent for the sky behind it.
func (m *Moon) Render() sprite.Sprite {
	if m == nil || !m.staged || m.w < 1 || m.h < 1 {
		return sprite.Sprite{}
	}
	stage := sprite.New(m.w, m.h)
	sprite.Blit(stage, 0, 0, m.base)
	return stage
}

// Stop drops the surface for the collector; a fresh Start repaints it.
func (m *Moon) Stop() {
	if m == nil {
		return
	}
	m.base = sprite.Sprite{}
	m.staged = false
}

// Orbit is the descent path and the craft riding it, painted over
// transparency so it lays cleanly on top of a Moon — or anything
// else on the same stage. Start paints and caches the dotted ring;
// Update runs the orbit clock; Render lays the craft over the ring —
// on the ring from the first frame, or streaking in first for an
// arriving orbit; Stop drops the ring so a stopped orbit holds no
// allocation. The clock carries across restarts, so a resize never
// rewinds the lap.
type Orbit struct {
	ring   sprite.Sprite
	clock  float64
	w, h   int
	staged bool
	arrive bool
}

// NewOrbit binds nothing yet: the curtain owns the allocation.
func NewOrbit() *Orbit {
	return &Orbit{}
}

// Arrive flies the craft in before the orbit: a fast streak off the
// left wing at orbit height that merges onto the top of the ring and
// loops the clockwise orbit until the cut. Call before Start.
// Nil-safe.
func (o *Orbit) Arrive() *Orbit {
	if o == nil {
		return nil
	}
	o.arrive = true
	return o
}

// Start paints the dotted descent path for a w×h stage.
func (o *Orbit) Start(w, h int) {
	if o == nil {
		return
	}
	o.w, o.h = w, h
	o.ring = paintRing(w, h)
	o.staged = true
}

// Update advances the orbit clock. dt <= 0 holds the craft still.
func (o *Orbit) Update(dt float64) {
	if o == nil || dt <= 0 {
		return
	}
	o.clock += dt
}

// Render lays the craft over the cached ring — at MarkerAt for the
// stock orbit, along ArrivalAt for an arriving one. Before Start and
// after Stop the orbit is off, so the stage is empty; a stage with no
// geometry stays transparent.
func (o *Orbit) Render() sprite.Sprite {
	if o == nil || !o.staged || o.w < 1 || o.h < 1 {
		return sprite.Sprite{}
	}
	stage := sprite.New(o.w, o.h)
	sprite.Blit(stage, 0, 0, o.ring)
	cx, cy, _, ringR := Geometry(o.w, o.h)
	if ringR < 1 {
		return stage
	}
	row, col := ringCell(cx, cy, ringR, angleAt(o.clock))
	if o.arrive {
		row, col = ArrivalAt(o.w, o.h, o.clock)
	}
	stage.Set(row, col, sprite.Cell{Ch: MarkerGlyph, FG: MarkerInk, BG: -1})
	return stage
}

// Stop drops the ring for the collector; a fresh Start repaints it.
// The clock stays, so the next stage picks the orbit up mid-flight.
func (o *Orbit) Stop() {
	if o == nil {
		return
	}
	o.ring = sprite.Sprite{}
	o.staged = false
}

// paintDisc is the moon's still life: the pixelated surface alone.
// Deterministic — the same stage always paints the same moon.
func paintDisc(w, h int) sprite.Sprite {
	stage := sprite.New(w, h)
	cx, cy, moonR, _ := Geometry(w, h)
	if moonR < 1 {
		return stage
	}
	for r := 0; r < h; r++ {
		for c := 0; c < w; c++ {
			px := float64(c - cx)
			py := float64(2 * (r - cy))
			if px*px+py*py > float64(moonR*moonR) {
				continue
			}
			stage.Set(r, c, surfaceCell(px/float64(moonR), -py/float64(moonR), r, c))
		}
	}
	return stage
}

// paintRing is the orbit's still life: the dotted descent path over
// transparency. Deterministic — the same stage always paints the same
// ring.
func paintRing(w, h int) sprite.Sprite {
	stage := sprite.New(w, h)
	cx, cy, moonR, ringR := Geometry(w, h)
	if moonR < 1 {
		return stage
	}
	steps := max(20, ringR*2)
	for i := 0; i < steps; i++ {
		theta := 2 * math.Pi * float64(i) / float64(steps)
		row, col := ringCell(cx, cy, ringR, theta)
		stage.Set(row, col, sprite.Cell{Ch: RingGlyph, FG: RingInk, BG: -1})
	}
	return stage
}

// surfaceCell picks one cell of terrain from unit-disc coordinates
// (x east, y up, both in fractions of the radius): crater pits first,
// then the maria plains, the darkened limb, a hashed speckle grain,
// and finally the plain sunlit body.
func surfaceCell(x, y float64, row, col int) sprite.Cell {
	for _, p := range pits {
		if inPatch(x, y, p) {
			return sprite.Cell{Ch: '░', FG: craterInk, BG: -1}
		}
	}
	for _, p := range maria {
		if inPatch(x, y, p) {
			return sprite.Cell{Ch: '▒', FG: mariaInk, BG: -1}
		}
	}
	if x*x+y*y > 0.92*0.92 {
		return sprite.Cell{Ch: '▒', FG: limbInk, BG: -1}
	}
	if (row*73+col*29)%19 == 0 {
		return sprite.Cell{Ch: '▒', FG: speckleInk, BG: -1}
	}
	return sprite.Cell{Ch: '▓', FG: surfaceInk, BG: -1}
}

func inPatch(x, y float64, p patch) bool {
	dx, dy := x-p.x, y-p.y
	return dx*dx+dy*dy <= p.r*p.r
}
