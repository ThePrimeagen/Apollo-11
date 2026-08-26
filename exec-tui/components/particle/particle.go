// Package particle is a small 2D particle engine. You give it a box
// (width × height), a nozzle (origin), a direction, a count and a period.
//
// Update is the only clock: live particles move, expire, and leave the box;
// new ones spawn at the origin on each period. Occupancy counts how many
// live particles sit in each terminal cell so a caller can color by density.
// The package does not draw.
//
// The engine flies under one of two modes, carried on the Config. The
// default, ModeStraight, keeps every particle on its velocity. ModeSwirl —
// switched on with Config.SideSwirl — is the cartoon wind: every particle
// curves toward the loop side as it flies, and every second one sweeps one
// full loop before flying on.
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
	ErrMode      = errors.New("particle: unknown mode")
	ErrNil       = errors.New("particle: nil engine")
)

// Mode picks the update rule live particles fly under.
type Mode int

const (
	// ModeStraight is the default: particles fly straight along their velocity.
	ModeStraight Mode = iota
	// ModeSwirl is the cartoon wind: every particle curves toward the
	// loop side, and every second one sweeps a full loop before flying on.
	ModeSwirl
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
	Mode               Mode    // update rule; the zero value is ModeStraight
	SwirlUp            bool    // swirl mode: curls aim at the top of the screen
}

// SideSwirl is the swirl switch: the same world, but particles curl to
// the side as they fly — half sweep one full cartoon-wind loop, half
// just curve out. up aims the curl at the top of the screen; false
// mirrors it downward.
func (c Config) SideSwirl(up bool) Config {
	c.Mode = ModeSwirl
	c.SwirlUp = up
	return c
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
	if c.Mode != ModeStraight && c.Mode != ModeSwirl {
		return ErrMode
	}
	return nil
}

// Curl is one particle's swirl plan: after Delay seconds of straight
// flight the heading turns Rate radians per second until Turn radians
// have been swept. The zero value never turns.
type Curl struct {
	Delay float64 // straight flight before the curl, seconds
	Rate  float64 // signed turn speed, radians per second
	Turn  float64 // total radians to sweep; 2π is one full loop
}

// Particle is one live (or just-expired) speck.
type Particle struct {
	Pos  Vec2
	Vel  Vec2
	Life float64 // remaining seconds
	Age  float64 // seconds since spawn
	Curl Curl    // swirl plan; the zero value flies straight
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

// advance moves the live particles under the config's mode.
func (e *Engine) advance(dt float64) {
	if e.Cfg.Mode == ModeSwirl {
		e.advanceSwirl(dt)
		return
	}
	e.advanceStraight(dt)
}

// advanceStraight is the standard rule: every particle keeps its velocity.
func (e *Engine) advanceStraight(dt float64) {
	n := 0
	for _, p := range e.Particles {
		p.Pos = Vec2{X: p.Pos.X + p.Vel.X*dt, Y: p.Pos.Y + p.Vel.Y*dt}
		p.Life -= dt
		p.Age += dt
		if p.Life > 0 && e.inRange(p.Pos) {
			e.Particles[n] = p
			n++
		}
	}
	e.Particles = e.Particles[:n]
}

// advanceSwirl is the wind rule: inside its curl window a particle's
// heading turns at its dealt rate — only the overlap of [Age, Age+dt]
// with the window turns, so a loop sweeps exactly Turn radians at any
// frame rate — and then it moves, expires, and leaves like any other.
func (e *Engine) advanceSwirl(dt float64) {
	n := 0
	for _, p := range e.Particles {
		if p.Curl.Rate != 0 && p.Curl.Turn > 0 {
			from := math.Max(p.Age, p.Curl.Delay)
			to := math.Min(p.Age+dt, p.Curl.Delay+p.Curl.Turn/math.Abs(p.Curl.Rate))
			if to > from {
				p.Vel = p.Vel.Rotate(p.Curl.Rate * (to - from))
			}
		}
		p.Pos = Vec2{X: p.Pos.X + p.Vel.X*dt, Y: p.Pos.Y + p.Vel.Y*dt}
		p.Life -= dt
		p.Age += dt
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
		if e.Cfg.Mode == ModeSwirl {
			p.Curl = e.rollCurl(i, life)
		}
		if p.Life > 0 && e.inRange(p.Pos) {
			e.Particles = append(e.Particles, p)
		}
	}
}

// The swirl deal, in fractions of a particle's life: loopers fly
// straight for a beat, sweep one full loop, and still have air left;
// arcs curve gently the whole way out.
const (
	loopTurn     = 2 * math.Pi
	loopDelayMin = 0.15
	loopDelayMax = 0.35
	loopSpan     = 0.35
	arcTurnMin   = math.Pi / 8
	arcTurnMax   = math.Pi / 3
	arcSpan      = 0.9
)

// rollCurl deals particle i its swirl plan: even deals sweep one full
// loop mid-flight, odd deals curve gently out — half and half.
func (e *Engine) rollCurl(i int, life float64) Curl {
	if life <= 0 {
		return Curl{}
	}
	sign := loopSign(e.Cfg.Direction, e.Cfg.SwirlUp)
	if i%2 == 0 {
		return Curl{
			Delay: e.between(loopDelayMin*life, loopDelayMax*life),
			Rate:  sign * loopTurn / (loopSpan * life),
			Turn:  loopTurn,
		}
	}
	turn := e.between(arcTurnMin, arcTurnMax)
	return Curl{Rate: sign * turn / (arcSpan * life), Turn: turn}
}

// loopSign is the turn sense that curls a flight toward the top of the
// screen (Y grows downward): leftward flights turn positive, rightward
// ones negative, and a vertical flight — with no up side of its own —
// deterministically takes the positive turn. Down loops mirror.
func loopSign(dir Vec2, up bool) float64 {
	s := 1.0
	if dir.X > 0 {
		s = -1
	}
	if !up {
		s = -s
	}
	return s
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
