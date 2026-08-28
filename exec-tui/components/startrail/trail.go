// Package startrail is the persist-particle comet wake. It emits at
// Origin on each period, the specks stay where they were born, and
// they fade as life runs out. Follow moves the nozzle so a moving
// star leaves a trail behind it. NewOrbit is the tuner/viewer
// preview: the nozzle walks a circle so the tail shape is readable.
package startrail

import (
	"math"

	"github.com/theprimeagen/apollo-11/exec-tui/components/particle"
	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
)

// Trail is the persist wake as a scene component.
type Trail struct {
	Eng     *particle.Engine
	Origin  particle.Vec2
	Heading particle.Vec2

	seed     int64
	w, h     int
	followed bool
}

// New binds a trail to its particle seed. Follow before Start to
// pin the first drop; otherwise the nozzle sits at stage center
// aiming right.
func New(seed int64) *Trail {
	return &Trail{seed: seed, Heading: particle.Vec2{X: 1, Y: 0}}
}

// Follow moves the nozzle. The next Update drops specks here; specks
// already born keep their cells.
func (t *Trail) Follow(origin, heading particle.Vec2) {
	if t == nil {
		return
	}
	t.Origin = origin
	t.followed = true
	if heading != (particle.Vec2{}) {
		t.Heading = heading
	}
}

func (t *Trail) Start(w, h int) {
	if t == nil {
		return
	}
	t.w, t.h = w, h
	uw := float64(w)*particle.CellWidthUnits - 0.01
	uh := float64(h)*particle.CellHeightUnits - 0.01
	if !t.followed {
		t.Origin = particle.Vec2{X: uw / 2, Y: uh / 2}
	}
	if t.Heading == (particle.Vec2{}) {
		t.Heading = particle.Vec2{X: 1, Y: 0}
	}
	t.Eng = particle.New(t.seed, Active().ParticleConfig(uw, uh, t.Origin, t.Heading))
}

func (t *Trail) Update(dt float64) {
	if t == nil || dt <= 0 || t.Eng == nil {
		return
	}
	uw := float64(t.w)*particle.CellWidthUnits - 0.01
	uh := float64(t.h)*particle.CellHeightUnits - 0.01
	cfg := Active().ParticleConfig(uw, uh, t.Origin, t.Heading)
	t.Eng.Cfg = cfg
	t.Eng.Update(dt)
}

func (t *Trail) Render() sprite.Sprite {
	if t == nil || t.Eng == nil || t.w < 1 || t.h < 1 {
		return sprite.Sprite{}
	}
	return paint(t.Eng, t.w, t.h)
}

func (t *Trail) Stop() {
	if t == nil {
		return
	}
	t.Eng = nil
}

func paint(eng *particle.Engine, cols, rows int) sprite.Sprite {
	sp := sprite.New(cols, rows)
	if eng == nil {
		return sp
	}
	for _, p := range eng.Particles {
		if p.Life <= 0 {
			continue
		}
		cell := particle.CellOf(p.Pos.X, p.Pos.Y)
		total := p.Life + p.Age
		frac := 1.0
		if total > 0 {
			frac = p.Life / total
		}
		sp.Set(cell.Row, cell.Col, spark(frac))
	}
	return sp
}

func spark(frac float64) sprite.Cell {
	switch {
	case frac > 0.7:
		return sprite.Cell{Ch: '✦', FG: 255, BG: -1}
	case frac > 0.45:
		return sprite.Cell{Ch: '*', FG: 229, BG: -1}
	case frac > 0.2:
		return sprite.Cell{Ch: '˚', FG: 195, BG: -1}
	default:
		return sprite.Cell{Ch: '·', FG: 245, BG: -1}
	}
}

// Orbit is a Trail whose nozzle walks a circle — the viewer preview
// of a persist wake.
type Orbit struct {
	*Trail
	clock  float64
	cx, cy float64
	r      float64
	w, h   int
}

// NewOrbit binds a circling trail to its seed.
func NewOrbit(seed int64) *Orbit {
	return &Orbit{Trail: New(seed)}
}

func (o *Orbit) Start(w, h int) {
	if o == nil {
		return
	}
	o.w, o.h = w, h
	o.cx = float64(w) * particle.CellWidthUnits / 2
	o.cy = float64(h) * particle.CellHeightUnits / 2
	o.r = math.Min(o.cx, o.cy) * 0.45
	if o.r < 4 {
		o.r = 4
	}
	o.clock = 0
	o.place()
	o.Trail.Start(w, h)
}

func (o *Orbit) Update(dt float64) {
	if o == nil || dt <= 0 {
		return
	}
	o.clock += dt
	o.place()
	o.Trail.Update(dt)
}

func (o *Orbit) place() {
	if o == nil || o.Trail == nil {
		return
	}
	s, c := math.Sincos(o.clock * 1.4)
	o.Follow(
		particle.Vec2{X: o.cx + c*o.r, Y: o.cy + s*o.r},
		particle.Vec2{X: -s, Y: c},
	)
}

func (o *Orbit) Render() sprite.Sprite {
	if o == nil || o.Trail == nil {
		return sprite.Sprite{}
	}
	return o.Trail.Render()
}

func (o *Orbit) Stop() {
	if o == nil {
		return
	}
	o.Trail.Stop()
}
