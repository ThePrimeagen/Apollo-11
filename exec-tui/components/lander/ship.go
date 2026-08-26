package lander

import (
	"math"

	"github.com/theprimeagen/apollo-11/exec-tui/components/fire"
	"github.com/theprimeagen/apollo-11/exec-tui/components/particle"
	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
)

const (
	// BodyCols/BodyRows is the size-4 frame: the full zoomed-in craft.
	BodyCols = 26
	BodyRows = 10
	// FlyInHoldSeconds is how long the craft waits offstage before
	// the fly-in starts.
	FlyInHoldSeconds = 3.0
	// FlyInSeconds is how long the slide from the right wing to center
	// stage takes, after the hold.
	FlyInSeconds = 4.0
	// BobPeriodSeconds is one full up-and-down of the parked bobble.
	BobPeriodSeconds = 10.0
	// BobAmplitudeCells is how far the bobble rides from center: one
	// full cell up and one down. (Half a cell would need half-shifted
	// art the atlas doesn't have yet.)
	BobAmplitudeCells = 1
	// FlameRow/FlameCol hang the 16×6 booster box off the tail,
	// relative to the hull's top-left, so the plume is vertically
	// flush with the size-4 west grey nozzle.
	FlameRow = 4
	FlameCol = 19
	// NorthFlameRow/NorthFlameCol hang the south-firing plume under
	// the size-4 north engine bell, matching the rocket card.
	NorthFlameRow = 8
	NorthFlameCol = 7
	// DropSeconds is how long the north-facing fall from off the top
	// of the stage to off the bottom takes.
	DropSeconds = 6.0
	// LandSeconds is how long the north-facing drop from off the top
	// onto the moon horizon pad takes.
	LandSeconds = 5.0
	// landSurfaceRows is the moon horizon's center thickness — the
	// hull parks with its feet on that ridge.
	landSurfaceRows = 5
)

// Ship is the Apollo craft as a scene component: the size-4 W-heading
// frame with its baked tilde plume stripped and, unless Dark, a live
// left-to-right booster fire trailing from the tail. It slides in from
// the right wing, parks at center stage, and bobbles on a slow sine.
// Start builds the hull and arms the fire for its stage; Stop drops
// both so a stopped ship holds no allocation, and a later Start
// rebuilds them.
type Ship struct {
	Body    sprite.Sprite
	Flame   *fire.Flame
	seed    int64
	clock   float64
	w, h    int
	dark    bool
	hold    float64
	heading sprite.Heading
	dropSec float64
	landSec float64
}

// NewShip binds the craft to its fire seed. Nothing is built until
// Start — the curtain owns the allocation.
func NewShip(seed int64) *Ship {
	return &Ship{seed: seed}
}

// Start builds the hull from the atlas and arms a fresh booster fire
// for a w×h stage. The clock carries across restarts, so a resize
// never replays the fly-in.
func (s *Ship) Start(w, h int) {
	if s == nil {
		return
	}
	s.w, s.h = w, h
	heading := s.heading
	if heading == "" {
		heading = sprite.W
	}
	s.Body = stripPlume(DefaultAtlas().MustFrame(sprite.Size4, heading))
	if s.dark {
		s.Flame = nil
		return
	}
	if heading == sprite.N {
		s.Flame = fire.Toward(s.seed, particle.Vec2{X: 0, Y: 1})
		return
	}
	s.Flame = &fire.Flame{Eng: particle.New(s.seed, shipFlameConfig())}
}

// shipFlameConfig slims the stock left-to-right booster to a cruise
// plume: a 16×6-unit box (three rows) so the beam hugs the engine bell,
// a lighter emission, a tighter jet, and speeds that let the tail taper
// out inside the box instead of dying on a wall.
func shipFlameConfig() particle.Config {
	cfg := fire.BoosterConfig()
	cfg.Width = 16 - 0.01
	cfg.Height = 6 - 0.01
	cfg.Origin = particle.Vec2{X: 1.0, Y: 3.0}
	cfg.Direction = particle.Vec2{X: 1, Y: 0}
	cfg.Count = 2
	cfg.MinSpeed, cfg.MaxSpeed = 11, 22
	cfg.Spread = 0.18
	cfg.Nozzle = 1.2
	return cfg
}

// Update moves the ship's clock and burns the fire. dt <= 0 holds.
func (s *Ship) Update(dt float64) {
	if s == nil || dt <= 0 {
		return
	}
	s.clock += dt
	if s.Flame != nil {
		s.Flame.Update(dt)
	}
}

// Clock is how many seconds of scene time the ship has played.
func (s *Ship) Clock() float64 {
	if s == nil {
		return 0
	}
	return s.clock
}

// Render composes fire first, hull second, into a stage-sized sprite,
// so the hull always wins the overlap at the tail and the plume
// appears from behind the bell. Before Start and after Stop there is
// nothing built, so the stage is empty.
func (s *Ship) Render() sprite.Sprite {
	if s == nil || s.w < 1 || s.h < 1 || s.Body.Width < 1 {
		return sprite.Sprite{}
	}
	stage := sprite.New(s.w, s.h)
	row, col := s.position()
	if s.Flame != nil {
		fr, fc := FlameRow, FlameCol
		if s.heading == sprite.N {
			fr, fc = NorthFlameRow, NorthFlameCol
		}
		sprite.Blit(stage, col+fc, row+fr, s.Flame.Sprite())
	}
	sprite.Blit(stage, col, row, s.Body)
	return stage
}

// Dark is a hull-only ship: Start still builds the body, but the
// booster stays cold. Nil-safe.
func (s *Ship) Dark() *Ship {
	if s == nil {
		return nil
	}
	s.dark = true
	s.Flame = nil
	return s
}

// Hold waits seconds offstage before the fly-in starts. Call before
// Start. Nil-safe.
func (s *Ship) Hold(seconds float64) *Ship {
	if s == nil {
		return nil
	}
	s.hold = seconds
	return s
}

// North flies the size-4 north-facing frame with a down-firing
// booster. Call before Start. Nil-safe.
func (s *Ship) North() *Ship {
	if s == nil {
		return nil
	}
	s.heading = sprite.N
	return s
}

// Drop falls from fully off the top of the stage to fully off the
// bottom over seconds. Call before Start. Nil-safe.
func (s *Ship) Drop(seconds float64) *Ship {
	if s == nil {
		return nil
	}
	s.dropSec = seconds
	return s
}

// Land falls from fully off the top onto the moon-horizon pad over
// seconds, then stays put. Call before Start. Nil-safe.
func (s *Ship) Land(seconds float64) *Ship {
	if s == nil {
		return nil
	}
	s.landSec = seconds
	return s
}

// position is this frame's hull top-left: a landing path, a drop, or
// the westbound fly-in, depending on how the ship was asked to fly.
func (s *Ship) position() (row, col int) {
	t := s.clock - s.hold
	if s.landSec > 0 {
		return LandPath(s.w, s.h, t, s.landSec)
	}
	if s.dropSec > 0 {
		return DropPath(s.w, s.h, t, s.dropSec)
	}
	return FlightPath(s.w, s.h, t)
}

// Parked starts the clock at the fly-in park so the first frame is
// already center-stage, skipping any Hold. Nil-safe.
func (s *Ship) Parked() *Ship {
	if s == nil {
		return nil
	}
	s.clock = s.hold + FlyInSeconds
	return s
}

// Stop drops the hull and the fire for the collector; a fresh Start
// rebuilds both.
func (s *Ship) Stop() {
	if s == nil {
		return
	}
	s.Body = sprite.Sprite{}
	s.Flame = nil
}

// FlightPath is the hull's top-left at t seconds into the scene, on a
// stageW×stageH stage. The craft holds level at center height: fully
// off the right wing at t=0, an eased slide that parks at center stage
// by FlyInSeconds, then a ±BobAmplitudeCells sine bobble with a
// BobPeriodSeconds period.
func FlightPath(stageW, stageH int, t float64) (row, col int) {
	if t < 0 {
		t = 0
	}
	row = (stageH - BodyRows) / 2
	park := (stageW - BodyCols) / 2
	if t < FlyInSeconds {
		// ease-out cubic: fast off the wing, gentle into the park.
		p := t / FlyInSeconds
		eased := 1 - math.Pow(1-p, 3)
		return row, stageW + int(math.Round(eased*float64(park-stageW)))
	}
	phase := 2 * math.Pi * (t - FlyInSeconds) / BobPeriodSeconds
	bob := int(math.Round(BobAmplitudeCells * math.Sin(phase)))
	return row - bob, park
}

// DropPath is the hull's top-left at t seconds of a seconds-long fall
// on a stageW×stageH stage: fully off the top at t=0, fully off the
// bottom at t=seconds, centered horizontally. Time before the curtain
// clamps to the start.
func DropPath(stageW, stageH int, t, seconds float64) (row, col int) {
	if t < 0 {
		t = 0
	}
	col = (stageW - BodyCols) / 2
	start, end := -BodyRows, stageH
	if seconds <= 0 || t >= seconds {
		return end, col
	}
	p := t / seconds
	return start + int(math.Round(p*float64(end-start))), col
}

// LandPadRow is where the hull's top-left parks on a landing: the
// moon horizon's center ridge minus the hull height, so the feet sit
// on the surface.
func LandPadRow(stageH int) int {
	return stageH - landSurfaceRows - BodyRows
}

// LandPath is the hull's top-left at t seconds of a seconds-long
// landing: fully off the top at t=0, parked on the horizon pad by
// t=seconds, then held there. Time before the curtain clamps to the
// start.
func LandPath(stageW, stageH int, t, seconds float64) (row, col int) {
	if t < 0 {
		t = 0
	}
	col = (stageW - BodyCols) / 2
	start, end := -BodyRows, LandPadRow(stageH)
	if seconds <= 0 || t >= seconds {
		return end, col
	}
	p := t / seconds
	return start + int(math.Round(p*float64(end-start))), col
}

// stripPlume drops the art's baked '~'/'≈' exhaust; the live particle
// fire is the plume here.
func stripPlume(sp sprite.Sprite) sprite.Sprite {
	out := sprite.New(sp.Width, sp.Height)
	for r := 0; r < sp.Height; r++ {
		for c := 0; c < sp.Width; c++ {
			cell := sp.At(r, c)
			if cell.Ch == '~' || cell.Ch == '≈' {
				continue
			}
			out.Set(r, c, cell)
		}
	}
	return out
}
