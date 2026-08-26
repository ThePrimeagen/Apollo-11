// Package particle is a small 2D particle engine. You give it a box
// (width × height), a nozzle (origin), a direction, a count and a period.
//
// Update is the only clock: live particles move, expire, and leave the box;
// new ones spawn at the origin on each period. Occupancy counts how many
// live particles sit in each terminal cell so a caller can color by density.
// The package does not draw.
package particle

import (
	"errors"
	"math"
	"math/rand"
)

// A unit is one cell-width or half a cell-height. Terminal cells are twice
// as tall as they are wide, so a unit is a square of visual space.
const (
	CellWidthUnits  = 1.0
	CellHeightUnits = 2.0
)

var (
	ErrSize      = errors.New("particle: width and height must be positive")
	ErrDirection = errors.New("particle: direction must be a non-zero vector")
	ErrOrigin    = errors.New("particle: origin is outside the box")
	ErrSpeed     = errors.New("particle: min speed is greater than max speed")
	ErrLife      = errors.New("particle: min life is greater than max life")
	ErrPeriod    = errors.New("particle: period must be non-negative")
	ErrCount     = errors.New("particle: count must be non-negative")
	ErrSpread    = errors.New("particle: spread must be non-negative")
	ErrNozzle    = errors.New("particle: nozzle must be non-negative")
	ErrNegative  = errors.New("particle: speed and life must be non-negative")
	ErrDistance  = errors.New("particle: max distance must be non-negative")
	ErrNil       = errors.New("particle: nil engine")
)

// Vec2 is a point or a direction in unit space.
type Vec2 struct {
	X, Y float64
}

// Len is the Euclidean length.
func (v Vec2) Len() float64 { return math.Hypot(v.X, v.Y) }

// Scale multiplies v by s.
func (v Vec2) Scale(s float64) Vec2 { return Vec2{X: v.X * s, Y: v.Y * s} }

// Normalize returns a unit vector in the same direction, or a zero vector.
func (v Vec2) Normalize() Vec2 {
	n := v.Len()
	if n == 0 {
		return Vec2{}
	}
	return v.Scale(1 / n)
}

// Rotate returns v turned counterclockwise by rad radians.
func (v Vec2) Rotate(rad float64) Vec2 {
	s, c := math.Sincos(rad)
	return Vec2{X: v.X*c - v.Y*s, Y: v.X*s + v.Y*c}
}

// Config is the world the engine runs in.
type Config struct {
	Width, Height      float64 // live box in units; leave it and you die
	Origin             Vec2    // nozzle; every emit starts here
	Direction          Vec2    // exhaust axis; New stores it normalized
	Count              int     // particles spawned each period
	Period             float64 // seconds between emits; 0 means no auto-emit
	MinLife, MaxLife   float64 // seconds, inclusive, rolled per particle
	MinSpeed, MaxSpeed float64 // units/sec along the jittered heading
	Spread             float64 // stddev of a normal, in radians, around Direction
	Nozzle             float64 // spawn thickness in units, perpendicular to Direction
	MaxDistance        float64 // 0 means unlimited; else die when farther from Origin
}

// Validate reports the first thing wrong with c.
func (c Config) Validate() error {
	if c.Width <= 0 || c.Height <= 0 {
		return ErrSize
	}
	if c.Direction == (Vec2{}) {
		return ErrDirection
	}
	if c.Origin.X < 0 || c.Origin.X > c.Width || c.Origin.Y < 0 || c.Origin.Y > c.Height {
		return ErrOrigin
	}
	if c.MinSpeed < 0 || c.MaxSpeed < 0 || c.MinLife < 0 || c.MaxLife < 0 {
		return ErrNegative
	}
	if c.MinSpeed > c.MaxSpeed {
		return ErrSpeed
	}
	if c.MinLife > c.MaxLife {
		return ErrLife
	}
	if c.Period < 0 {
		return ErrPeriod
	}
	if c.Count < 0 {
		return ErrCount
	}
	if c.Spread < 0 {
		return ErrSpread
	}
	if c.Nozzle < 0 {
		return ErrNozzle
	}
	if c.MaxDistance < 0 {
		return ErrDistance
	}
	return nil
}

// Particle is one live (or just-expired) speck.
type Particle struct {
	Pos  Vec2
	Vel  Vec2
	Life float64 // remaining seconds
}

// Cell is one terminal cell in unit space.
type Cell struct {
	Col, Row int
}

// CellOf maps a unit-space point onto the cell it sits in.
func CellOf(x, y float64) Cell {
	return Cell{Col: int(math.Floor(x)), Row: int(math.Floor(y / CellHeightUnits))}
}

// Engine owns the live particle list and the period clock.
type Engine struct {
	Cfg       Config
	Particles []Particle

	rng *rand.Rand
	acc float64
}

// New builds an engine that has not yet emitted. The first Update with a
// positive dt and Period > 0 fires immediately; later emits wait Period.
func New(seed int64, cfg Config) *Engine {
	cfg.Direction = cfg.Direction.Normalize()
	return &Engine{
		Cfg: cfg,
		rng: rand.New(rand.NewSource(seed)),
		acc: cfg.Period,
	}
}

// Validate checks the current config.
func (e *Engine) Validate() error { return e.Cfg.Validate() }

// Config is a copy of the running world. Mutate the copy and pass it
// to SetConfig; assigning fields on the copy does not change the engine.
func (e *Engine) Config() Config {
	if e == nil {
		return Config{}
	}
	return e.Cfg
}

// SetConfig replaces the running world with cfg. Invalid configs are
// rejected and the engine is left as it was. Live particles keep
// flying under the new rules on the next Update — a tighter
// MaxDistance will kill anyone already past it.
func (e *Engine) SetConfig(cfg Config) error {
	if e == nil {
		return ErrNil
	}
	if err := cfg.Validate(); err != nil {
		return err
	}
	cfg.Direction = cfg.Direction.Normalize()
	e.Cfg = cfg
	return nil
}

// Update advances the clock by dt seconds: move, expire, emit.
// dt <= 0 is a no-op.
func (e *Engine) Update(dt float64) {
	if dt <= 0 {
		return
	}
	e.advance(dt)
	e.emitDue(dt)
}

func (e *Engine) advance(dt float64) {
	n := 0
	for _, p := range e.Particles {
		p.Pos = Vec2{X: p.Pos.X + p.Vel.X*dt, Y: p.Pos.Y + p.Vel.Y*dt}
		p.Life -= dt
		if p.Life > 0 && e.inRange(p.Pos) {
			e.Particles[n] = p
			n++
		}
	}
	e.Particles = e.Particles[:n]
}

func (e *Engine) emitDue(dt float64) {
	if e.Cfg.Period <= 0 || e.Cfg.Count <= 0 {
		return
	}
	e.acc += dt
	for e.acc >= e.Cfg.Period {
		e.emit()
		e.acc -= e.Cfg.Period
	}
}

func (e *Engine) emit() {
	dir := e.Cfg.Direction.Normalize()
	if dir == (Vec2{}) {
		return
	}
	for i := 0; i < e.Cfg.Count; i++ {
		angle := e.rng.NormFloat64() * e.Cfg.Spread
		heading := dir.Rotate(angle)
		speed := e.between(e.Cfg.MinSpeed, e.Cfg.MaxSpeed)
		life := e.between(e.Cfg.MinLife, e.Cfg.MaxLife)
		pos := e.Cfg.Origin
		if e.Cfg.Nozzle > 0 {
			perp := Vec2{X: -dir.Y, Y: dir.X}
			off := (e.rng.Float64() - 0.5) * e.Cfg.Nozzle
			pos = Vec2{X: pos.X + perp.X*off, Y: pos.Y + perp.Y*off}
		}
		p := Particle{
			Pos:  pos,
			Vel:  heading.Scale(speed),
			Life: life,
		}
		if p.Life > 0 && e.inRange(p.Pos) {
			e.Particles = append(e.Particles, p)
		}
	}
}

func (e *Engine) between(min, max float64) float64 {
	if max <= min {
		return min
	}
	return min + e.rng.Float64()*(max-min)
}

func (e *Engine) inside(p Vec2) bool {
	return p.X >= 0 && p.X <= e.Cfg.Width && p.Y >= 0 && p.Y <= e.Cfg.Height
}

func (e *Engine) inRange(p Vec2) bool {
	if !e.inside(p) {
		return false
	}
	if e.Cfg.MaxDistance <= 0 {
		return true
	}
	d := math.Hypot(p.X-e.Cfg.Origin.X, p.Y-e.Cfg.Origin.Y)
	return d <= e.Cfg.MaxDistance
}

// Occupancy counts live particles in each terminal cell.
func (e *Engine) Occupancy() map[Cell]int {
	occ := make(map[Cell]int)
	for _, p := range e.Particles {
		if p.Life <= 0 {
			continue
		}
		occ[CellOf(p.Pos.X, p.Pos.Y)]++
	}
	return occ
}
